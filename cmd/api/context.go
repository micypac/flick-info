// Contains helper methods for reading/writing the User struct to and from the request context.
package main

import (
	"context"
	"net/http"

	"github.com/micypac/flick-info/internal/data"
)

type contextKey string

// Convert the string 'user' to a contextKey type and assign it to userContextKey constant.
// Use this constant as the key for getting and setting user info from request context.
const userContextKey = contextKey("user")


// This method returns a new copy of the request with the provided User struct added to the context.
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}


// The contextGetUser method retrieves the User struct from the request context.
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
