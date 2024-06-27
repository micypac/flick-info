package main

import (
	"context"
	"errors"
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

	// Create a shutdownError channel. Use this to receive any errors returned by the graceful Shutdown() function.
	shutdownError := make(chan error)

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
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// Create a context with a 5-second timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer cancel()

		// Call the Shutdown() method on our server, passing in the context.
		// Shutdown() will return nil if the graceful shutdown was successful or an error (may happen
		// because of problems closing the listener or the shutdown didn't happen before the 5sec deadline).
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		// Log a message to say that we're waiting for any background goroutines to complete.
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		// Call Wait() to block until WaitGroup counter is zero. Then return nil
		// on the shutdownError channel, to inidicate the shutdown completed without any issues.
		app.wg.Wait()
		shutdownError <- nil
	}()

	// Log the starting server message.
	app.logger.PrintInfo("starting server", map[string]string{
		"env": app.config.env,
		"addr": srv.Addr,
	})

	// Calling server Shutdown() will cause ListenAndServe() to immediately return a http.ErrServerClosed error.
	// This is an indication that the graceful shutdown has been initiated. Check specifically for this error
	// only returning it if it is not http.ErrServerClosed.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise wait to receive the return value from Shutdown() on the shutdownError channel.
	// If the return value is an error, there was a problem with the graceful shutdown and we return it.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point, the graceful shutdown was successful.
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
