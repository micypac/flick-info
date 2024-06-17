package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/micypac/flick-info/internal/validator"

	"github.com/lib/pq"
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


type MovieModel struct {
	DB *sql.DB
}


// GetAll() return a slice of movies.
func(m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, error) {
	stmt := fmt.Sprintf(`
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
	`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, stmt, title, pq.Array(genres))
	if err != nil {
		return nil, err
	}

	// Defer rows.Close() to ensure the resultset is closed before method returns.
	defer rows.Close()

	// Initialize empty slice to hold the movies data.
	movies := []*Movie{}

	for rows.Next() {
		// Init empty Movie struct to hold data for a movie.
		var movie Movie

		err := rows.Scan(
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)

		if err != nil {
			return nil, err
		}

		// Add the Movie struct to the movie slice.
		movies = append(movies, &movie)
	}

	// When rows.Next() loop finished, call rows.Err() to retrieve any error that 
	// was encounterd during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil


}

// Insert method accepts a pointer to a Movie struct which contain data for the new record.
func (m MovieModel) Insert(movie *Movie) error {
	stmt := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`

	// Create a slice containing the values for the placeholder parameters from the Movie struct.
	args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)

	defer cancel()

	// Use the QueryRow() method to execute the SQL statement on the connection pool, passing in the args
	// as a variadic parameter and scanning the system-generated values into the movie struct.
	return m.DB.QueryRowContext(ctx, stmt, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}


func (m MovieModel) Get(id int64) (*Movie, error) {
	// The PostgreSQL bigserial type for the movie ID starts auto-incrementing at 1 by default.
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	stmt := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1
	`
	// Declare a Movie struct that will hold the returned data.
	var movie Movie

	// Use context.WithTimeout() function to create a context w/c carries a 3sec timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)

	// Use defer to make sure we cancel the context before the Get() method returns.
	defer cancel()

	// Use QueryRowContext() method to exec the query, passing in the context with deadline.
	err := m.DB.QueryRowContext(ctx, stmt, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}


func (m MovieModel) Update(movie *Movie) error {
	stmt := `
		UPDATE movies 
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
	`

	args := []interface{}{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}


func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	stmt := `
		DELETE FROM movies
		WHERE id = $1	
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	// Call the RowsAffected() method to get the number of rows affected by the query.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

