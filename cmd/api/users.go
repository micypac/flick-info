package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/micypac/flick-info/internal/data"
	"github.com/micypac/flick-info/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// Anonymous input struct to hold the exprected data from the request body.
	var input struct {
		Name string `json:"name"`
		Email string `json:"email"`
		Password string `json:"password"`
	}

	// Parse the request body and store the result in the input struct.
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Copy the values from the input struct to a new User struct.
	user := &data.User{
		Name: input.Name,
		Email: input.Email,
		Activated: false,
	}

	// Use the Password Set() method to generate the hashed version of the password.
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	// After a new user record has been created, generate a new activation token for the user.
	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}


	// Use the background() helper to execute an anonymous function that sends the welcome email.
	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID": user.ID,
		}


		// Call the Send() method on the Mailer, passing in the user's email address,
		// name of the template file, and the User struct containing the dynamic data.
		err = app.mailer.Send(user.Email, "user_welcome.tmpl.html", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	
	})

	
	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}	
}


func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the activation token from the request body.
	var input struct {
		TokenPlaintext	string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate the plaintext token provided by the client.
	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve the details of the user associated with the token using the GetForToken() method.
	// If no matching record is found, let the client know the token provided is invalid.
	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
			case errors.Is(err, data.ErrRecordNotFound):
				v.AddError("token", "invalid or expired activation token")
				app.failedValidationResponse(w, r, v.Errors)
			default:
				app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Update the user's activated status to true.
	user.Activated = true

	// Save the updated user record in the db, checking for any edit conflicts.
	err = app.models.Users.Update(user)
	if err != nil {
		switch {
			case errors.Is(err, data.ErrEditConflict):
				app.editConflictResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Delete all activation tokens for the user if everything is successful.
	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send updated user details in the JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
