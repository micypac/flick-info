package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/micypac/flick-info/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

// Custom ErrDuplicateEmail error to represent a violation of the "users_email_key" constraint.
var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

// Definition of User struct to represent individual user records.
type User struct {
	ID int64 `json:"id"`	
	CreatedAt time.Time `json:"created_at"`
	Name string `json:"name"`
	Email string `json:"email"`
	Password password `json:"-"`
	Activated bool `json:"activated"`
	Version int `json:"-"`
}

// Custom password type containing the plain text and hashed versions of the password.
type password struct {
	plaintext *string
	hash []byte
}


// Set() method calculates the bcrypt hash of the plaintext password and stores both the plain and hashed version in the struct.
func (p *password) Set(plaintextPW string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPW), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPW
	p.hash = hash

	return nil
}


// The Matches() method checks whether the provided plaintext password matches the hashed password stored in the struct.
func (p *password) Matches(plaintextPW string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPW))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}


func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	// If the password plaintext is not nil, call the ValidatePasswordPlaintext() helper.
	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}


// UserModel struct to hold the methods for querying and modifying user records in the database.
type UserModel struct {
	DB *sql.DB
}

// Insert() method to add a new user record to the users table.
func (m UserModel) Insert(user *User) error {
	stmt := `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`

	args := []interface{}{user.Name, user.Email, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// If the table already contains a user with the same email address, the query will fail with a UNIQUE constraint.
	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}


// Retrieve the user details from the db based on the email address.
func (m UserModel) GetByEmail(email string) (*User, error) {
	stmt := `
		SELECT id, created_at, name, email, password_hash, activated, version
		FROM users
		WHERE email = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}


// Update user information in the db.
func (m UserModel) Update(user *User) error {
	stmt := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version`

	args := []interface{}{
		user.Name, 
		user.Email, 
		user.Password.hash, 
		user.Activated, 
		user.ID, 
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}


func (m UserModel) GetForToken(tokenScope, TokenPlaintext string) (*User, error) {
	// Calculate SHA-256 hash of the plaintext token.
	tokenHash := sha256.Sum256([]byte(TokenPlaintext))

	stmt := `
		SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.activated, users.version
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3
	`
	
	// Create a slice containing the query arguments.
	args := []interface{}{tokenHash[:], tokenScope, time.Now()}

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query and scan the result into the user struct.
	err := m.DB.QueryRowContext(ctx, stmt, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
