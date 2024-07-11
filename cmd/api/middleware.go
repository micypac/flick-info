package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/micypac/flick-info/internal/data"
	"github.com/micypac/flick-info/internal/validator"
	"github.com/tomasen/realip"
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

func (app *application) rateLimit(next http.Handler) http.Handler {
	// Client struct to hold the rate limiter and last seen time for each client(IP address).
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// Declare a mutex and a map to hold the clients' struct.
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Launch a background goroutine to remove old entries from the clients map once every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from happening while the cleanup is taking place.
			mu.Lock()

			// Loop through the map and remove any entries where the last seen time is older than 3 minutes.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			// Unlock the mutex.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Carry out the rate limiting checks if the limiter is enabled.
		if app.config.limiter.enabled {

			// Extract the clients IP address from the request.
			ip := realip.FromRequest(r)

			// Lock the mutex to ensure that the map access is safe.
			mu.Lock()

			// Check if the IP address already exists in the map.
			// If it doesnt, create a new client instance with rate limiter to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}

			// Update the last seen time for the client.
			clients[ip].lastSeen = time.Now()

			// Call the Allow() method on the rate limiter for the current IP address.
			// If the request is not allowed, unlock the mutex and send a 429 Too Many Requests response.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceedResponse(w, r)
				return
			}

			// Unlock the mutex before calling the next handler in the chain.
			// DON'T use defer to unlock the mutex, as that would mean that the mutex isn't unlocked until all
			// the handlers downstream of this middleware have also returned.
			mu.Unlock()

		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the 'Vary: Authorization' header to the response. This indicates to any caches that the response
		// may vary based on the value of the Authorization header in the request.
		w.Header().Add("Vary", "Authorization")

		// Rerieve the value of the Authorization header from the request. Empty string "" is returned if the header is not present.
		authorizationHeader := r.Header.Get("Authorization")

		// If there is no Authorization header found, use the contextSetUser() helper to add the AnonymousUser to the request context
		// then call the next handler in the chain and return.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Otherwise, we expect the value of the Authorization header to be in the format 'Bearer <token>'.
		// Split this into it constituent parts, and if its not in the expected format, return 401 Unauthorized response.
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract the actual authentication token from the header parts.
		token := headerParts[1]

		// Validate the token.
		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Retrieve the details of the user associated with the authentication token.
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// Call the contextSetUser() helper to add the user info to the request context.
		r = app.contextSetUser(r, user)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		// If anonymous user, call the authenticationRequiredResponse().
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		// Check that a user is activated.
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	// Wrap fn with the requireAuthenticatedUser() middleware.
	return app.requireAuthenticatedUser(fn)
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the user from the request context.
		user := app.contextGetUser(r)

		// Get the permissions slice for the user.
		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// Check if the slice includes the require permission code.
		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireActivatedUser(fn)
}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the "Vary: Origin" header.
		w.Header().Set("Vary", "Origin")

		// Add the "Vary: Access-Control-Request-Method" header.
		w.Header().Set("Vary", "Access-Control-Request-Method")

		// Get the value of the request's Origin header.
		origin := r.Header.Get("Origin")

		// Check if Origin request header is not empty AND at least one trusted origin is configured.
		if origin != "" && len(app.config.cors.trustedOrigins) != 0 {
			for i := range app.config.cors.trustedOrigins {
				// If the Origin header matches a trusted origin, add the Access-Control-Allow-Origin header to the response.
				if origin == app.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}

				// If request has the HTTP method OPTIONS and contains the 'Access-Control-Request-Method'
				// header then it's a preflight request.
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					// Add the 'Access-Control-Allow-Methods' header to the response.
					w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
					// Add the 'Access-Control-Allow-Headers' header to the response.
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

					// Write the response with a 200 OK status and return from the middleware.
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		// w.Header().Set("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(w, r)
	})
}

func (app *application) metrics(next http.Handler) http.Handler {
	// Init the new expvar variables.
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Increment the totalRequestsReceived counter by 1.
		totalRequestsReceived.Add(1)

		// Call the httpsnoop.CaptureMetrics() passing in the handler in the chain along
		// with the existing ResponseWriter and Request.
		metrics := httpsnoop.CaptureMetrics(next, w, r)

		// On the way back up the middleware chain, increment the totalResponsesSent counter by 1.
		totalResponsesSent.Add(1)

		// Calculate the number of microseconds since the start of the request and
		// incement the totalProcessingTimeMicroseconds counter by that amount.
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())

		// Increment the count for the given status code by 1.
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	})
}
