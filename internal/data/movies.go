package data

import "time"

type Movie struct {
	ID 				int64				`json:"id"`									// Unique integer id for the movie.
	CreatedAt time.Time		`json:"-"`									// Timestamp when the movie is added to the db. '-' struct tag directive to hide in the output.
	Title 		string			`json:"title"`		
	Year 			int32				`json:"year,omitempty"`			// Release year. 'omitempty' struct directive to hide field in the output if the it is zero value.
	Runtime 	Runtime			`json:"runtime,omitempty"`	// Runtime (in minutes).
	Genres 		[]string		`json:"genres,omitempty"`		// Genres of the movie.
	Version 	int32				`json:"version"`						// Version starts at 1 and incremented when movie info is updated.
}
