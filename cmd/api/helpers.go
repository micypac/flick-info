package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// Retrieve the "id" URL parameter from the current request context, convert it
// integer and return it. If operation fails, return 0 and error.
func (app *application) readIDParam(r *http.Request) (int64, error) {
	// Any interpolated URL parameters will be stored in the request context. 
	// httprouter.ParamsFromContext() will retrieve a slice containing parameter names and values.
	params := httprouter.ParamsFromContext(r.Context())

	// Use ByName() method to get the value of the "id" parameter from the slice, its returned as a string.
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid ID parameter")
	}

	return id, nil
}
