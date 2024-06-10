package main

import (
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Create a map which holds the information that we want to send in the response.
	data := map[string]string{
		"status": "available",
		"environment": app.config.env,
		"version": version,
	}

	// Pass the map to the json.Marshal() function. This returns a []byte slice containing the encoded JSON.
	err := app.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		app.logger.Println(err)
		http.Error(w, "The server encountered an error and could not process your request", http.StatusInternalServerError)
	}
}
