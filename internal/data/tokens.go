package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/micypac/flick-info/internal/validator"
)

// Define constants for the token scope.
const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

// Token struct definition that holds the data for a token.
// This includes plaintext and hashed versions of the token, associated user ID, expiry time, and scope.
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	// Create Token instance containing the userID, expiry, and scope information.
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	// Initialize a zero-value byte slice with a length of 16 bytes.
	randomBytes := make([]byte, 16)

	// User Read() function from the crypto/rand package to fill the randomBytes slice with random bytes
	// from OS' CSPRNG. This will return an error if it fails.
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// Encode the byte slice to a base32 encoded string and store it in the plaintext field of the token.
	// This will the token string that we send to the user's welcome email.
	// Note: By default base32 string may be padded at the end with '=' character. Use WithPadding(base32.NoPadding) to omit them.
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// Generate a SHA-256 hash of plaintext token string.
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// Check that the plaintext token provided is exactly 52bytes long.
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// TokenModel type.
type TokenModel struct {
	DB *sql.DB
}

// New() method creates a new Token struct then inserts the data in the tokens table.
func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

// Insert() method adds the data for a specific token to the tokens table.
func (m TokenModel) Insert(token *Token) error {
	stmt := `INSERT INTO tokens (hash, user_id, expiry, scope) VALUES($1, $2, $3, $4)`

	args := []interface{}{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	_, err := m.DB.ExecContext(ctx, stmt, args...)
	return err
}

// DeleteAllForUser() deletes all tokens for a specific user and scope.
func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	stmt := `DELETE FROM tokens WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, stmt, scope, userID)
	return err
}
