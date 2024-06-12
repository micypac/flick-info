package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Custom error UnmarshalJSON() can return if parsing JSON failed.
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// Declare a custom Runtime type, which has underlying type int32.
// This is used as a field in Movie struct and to customize the output format using the MarshalJSON method.
type Runtime int32

// Implement MarshalJSON() method on the Runtime type so it satisfies the json.Marshaler interface.
// This should return the JSON-encoded value of the Movie 'Runtime' field, '<runtime> mins'.
func (r Runtime) MarshalJSON() ([]byte, error) {
	// Generate a string containing the movie runtime in the desired format.
	jsonValue := fmt.Sprintf("%d mins", r)

	// Use the strconv.Quote() function on the string to wrap it in double quotes.
	// It needs to be surrounded in double quotes in order to be a valid JSON string.
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// Implement UnmarshalJSON() method on the Runtime type so it satisfies the json.Unmarshaler interface.

func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")

	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	*r = Runtime(i)

	return nil
}
