package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/micypac/flick-info/internal/data"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Create a new movie")
}


func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Read "id" URL parameter.
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(w, r)
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

	// Encode the struct to JSON and send it as the HTTP response.
	err = app.writeJSON(w, http.StatusOK, movie, nil)
	if err != nil {
		app.logger.Println(err)
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}
