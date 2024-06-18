package main

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")

				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}


//
func (app *application) rateLimit(next http.Handler) http.Handler {
	// Init new rate limiter which allows an average of 2req/sec, with a max of 4req in a single 'bursts'.
	limiter := rate.NewLimiter(2, 4)

	// Return a function closure which 'closes' over the limiter verbiage.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Call limiter.Allow() to see if the request is permitted. 
		// If its not, return a 429 Too Many Requests response.
		if !limiter.Allow() {
			app.rateLimitExceedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
