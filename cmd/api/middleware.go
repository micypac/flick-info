package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

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
	// Client struct to hold the rate limiter and last seen time for each client(IP address).
	type client struct {
		limiter *rate.Limiter
		lastSeen time.Time
	}

	// Declare a mutex and a map to hold the clients' struct.
	var (
		mu sync.Mutex
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
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

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
