package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)


const version = "1.0.0"

// Holds all configuration settings for the app. 
// Read these config settings from command-line flags when the app starts.
// port - the network port the server is listening on
// env - current operating env for the app(dev, staging, prod, etc.)
type config struct {
	port int
	env string
}

// App struct holds the dependencies for HTTP handlers, helpers, and middleware. 
type application struct {
	config config
	logger *log.Logger
}


func main() {
	var cfg config

	// Read the value of command-line flags into the config struct. 
	// Port# 4000 and "dev" environment default if no corresponding flags are provided.
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Initialize a new logger which writes messages to the standard out stream, prefixed with current date and time.
	logger := log.New(os.Stdout, "", log.Ldate | log.Ltime)

	// Declare an instance of the application struct, containing the config struct and the logger.
	app := &application{
		config: cfg,
		logger: logger,
	}

	// HTTP server with timeout settings w/c listens to config port and uses the app.routes() as the handler.
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.port),
		Handler: app.routes(),
		IdleTimeout: time.Minute,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start the HTTP server.
	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err := srv.ListenAndServe()
	logger.Fatal(err)

}
