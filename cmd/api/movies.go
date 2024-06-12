package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/micypac/flick-info/internal/data"
	"github.com/micypac/flick-info/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Declare an anonymous struct to hold the info we expect to be in the request body.
	var input struct{
		Title 		string 				`json:"title"`
		Year 			int32 				`json:"year"`
		Runtime 	data.Runtime 	`json:"runtime"`
		Genres 		[]string 			`json:"genres"`
	}

	// Use the readJSON() helper method to decode the request body into the input struct.
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Copy the values from input struct to new Movie struct.
	movie := &data.Movie{
		Title: input.Title,
		Year: input.Year,
		Runtime: input.Runtime,
		Genres: input.Genres,
	}

	// Initialize a new Validator instance.
	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Write the contents of the input struct to the HTTP response.
	fmt.Fprintf(w, "%+v\n", input)

}


func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Read "id" URL parameter.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Create new instance of Movie struct, containing the ID we extracted from URL parameter and some dummy data.
	movie := data.Movie{
		ID: id,
		CreatedAt: time.Now(),
		Title: "Casablanca",
		Runtime: 102,
		Genres: []string{"drama", "romance", "war"},
		Version: 1,
	}

	// Encode the struct to JSON and send it as the HTTP response. Enclose the Movie struct instance to 'envelope' type.
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}



