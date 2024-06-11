package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// Define an envelope type.
type envelope map[string]interface{}

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


// Helper method for sending JSON responses. It takes the destination ResponseWriter, HTTP status code to send, 
// the data to encode to JSON, and header map containing HTTP headers to set.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Encode the data to JSON by passing to the json.Marshal() function. This returns a []byte slice containing the encoded JSON.
	// Use MarshalIndent() so that whitespace is added to the encoded JSON.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// Append newline to the JSON to make it easier to view in terminal apps.
	js = append(js, '\n')

	// Loop through the headers map and add each to the response header.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Set HTTP header 'Content-Type' as 'application/json' and write the status code and response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Send the []byte slice containing the JSON as response body.
	w.Write(js)

	return nil
}
