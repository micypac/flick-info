package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/micypac/flick-info/internal/data"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Declare an anonymous struct to hold the info we expect to be in the request body.
	var input struct{
		Title 		string 		`json:"title"`
		Year 			int32 		`json:"year"`
		Runtime 	int32 		`json:"runtime"`
		Genres 		[]string 	`json:"genres"`
	}

	// Initialize a new json.Decoder instance that reads from the request body, and then 
	// use the Decode() method to decode the body contents into the pointer input struct.
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
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



