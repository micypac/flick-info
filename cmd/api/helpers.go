package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/micypac/flick-info/internal/validator"
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


// Helper method for reading JSON request. Decode the JSON from the request body then triage the errors and
// replace them with custom message if necessary.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Initialize a new json.Decoder that reads from the request body and call the DisallowUnknownFields() before decoding.
	// If the JSON request have fields that cannot be mapped to the target destination, it will error.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	
	// Use the Decode() method to decode the body contents into the pointer input struct.
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
			
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// JSON has field that is unmappable in target destination.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// Request body exceeds 1MB in size.
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	// Call Decode again using a pointer to an empty anonymous struct as destination.
	// If we received a single JSON value, this will return an io.EOF error.
	// Anything else means there is additional data in the request body and we return an error.
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}


// readString() helper returns a string value from the query string, or provided default value 
// if no matching key could be found.
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string. If no key exists, this will return empty string "".
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}


// readCSV() helper returns a string slice from the comma-separated query string values.
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}

	return strings.Split(csv, ",")
}


// readInt helper returns an int value from query string.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}
