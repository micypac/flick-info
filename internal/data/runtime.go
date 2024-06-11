package data

import (
	"fmt"
	"strconv"
)

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
