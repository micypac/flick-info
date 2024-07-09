package main

import (
	"flag"
	"log"
	"net/http"
)

const html = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
</head>
<body>
	<h1>Simple CORS</h1>
	<div id="output"></div>
	<script>
		document.addEventListener('DOMContentLoaded', function() {
			fetch("http://localhost:4000/v1/healthcheck").then(
				
				function(response) {
					response.text().then((text) => {
						document.getElementById("output").innerHTML = text;
					});
				},

				function(error) {
					document.getElementById("output").innerHTML = error;
				}
			);
		});
	</script>
</body>
`

func main() {
	// Make the server address configurable at runtime using command-line flag.
	addr := flag.String("addr", ":9000", "Server address")
	flag.Parse()

	log.Printf("starting server on %s", *addr)

	err := http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		w.Write([]byte(html))
	}))

	log.Fatal(err)

}
