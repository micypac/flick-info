package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {

	// HTTP server with timeout settings w/c listens to config port and uses the app.routes() as the handler.
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", app.config.port),
		Handler: app.routes(),
		IdleTimeout: time.Minute,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start a background goroutine.
	go func() {
		// Create a quit channel which carries os.Signal values.
		quit := make(chan os.Signal, 1)

		// Use signal.Notify() to subscribe to the SIGINT and SIGTERM signals and relay them to the quit channel.
		// Any other signals received will not be relayed to the quit channel and retain their default behavior.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Read the signal from the quit channel. 
		// This code will block until a signal is received.
		s := <-quit

		// Log a message to say that the signal has been caught.
		app.logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})

		// Exit the application with a 0 (success) status code.
		os.Exit(0)
	}()

	// Log the starting server message.
	app.logger.PrintInfo("starting server", map[string]string{
		"env": app.config.env,
		"addr": srv.Addr,
	})

	return srv.ListenAndServe()
}
