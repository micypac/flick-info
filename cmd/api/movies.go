package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Create a new movie")
}


func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Any interpolated URL parameters will be stored in the request context. 
	// httprouter.ParamsFromContext() will retrieve a slice containing parameter names and values.
	params := httprouter.ParamsFromContext(r.Context())

	// Use ByName() method to get the value of the "id" parameter from the slice, its returned as a string.
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "Showing the details of move %d\n", id)
}
