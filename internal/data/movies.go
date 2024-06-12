package data

import (
	"time"

	"github.com/micypac/flick-info/internal/validator"
)

type Movie struct {
	ID 				int64				`json:"id"`									// Unique integer id for the movie.
	CreatedAt time.Time		`json:"-"`									// Timestamp when the movie is added to the db. '-' struct tag directive to hide in the output.
	Title 		string			`json:"title"`		
	Year 			int32				`json:"year,omitempty"`			// Release year. 'omitempty' struct directive to hide field in the output if the it is zero value.
	Runtime 	Runtime			`json:"runtime,omitempty"`	// Runtime (in minutes).
	Genres 		[]string		`json:"genres,omitempty"`		// Genres of the movie.
	Version 	int32				`json:"version"`						// Version starts at 1 and incremented when movie info is updated.
}


func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")

	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}
