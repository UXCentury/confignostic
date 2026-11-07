// Package identity provides credential primitives for the E01 authentication layer.
// It does not authenticate HTTP requests or authorize workspace operations.
package identity

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	passwordIterations = 600000
	saltBytes          = 16
	keyBytes           = 32
	tokenBytes         = 32
)

var (
	ErrPassword   = errors.New("password must contain 15 to 1024 UTF-8 bytes")
	ErrCredential = errors.New("invalid credential")
)

// HashPassword uses a fresh salt and a versioned, fixed-cost PBKDF2-SHA256 format.
// Passwords are never normalized or truncated. Callers must not log inputs.
func HashPassword(password string) (string, error) {
	if !validPassword(password) {
		return "", ErrPassword
	}
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, keyBytes)
	if err != nil {
		return "", ErrCredential
	}
	return "pbkdf2-sha256$v1$600000$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

func validPassword(password string) bool {
	return len(password) >= 15 && len(password) <= 1024 && utf8.ValidString(password)
}

// VerifyPassword accepts only the supported fixed-cost format. Untrusted hash
// parameters cannot request unbounded CPU or memory. Sign-in handlers must also
// throttle attempts and perform dummy verification for unknown users.
func VerifyPassword(encoded, password string) error {
	if !validPassword(password) || len(encoded) != 90 {
		return ErrCredential
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2-sha256" || parts[1] != "v1" || parts[2] != "600000" {
		return ErrCredential
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[3])
	if err != nil || len(salt) != saltBytes {
		return ErrCredential
	}
	want, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(want) != keyBytes {
		return ErrCredential
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, keyBytes)
	if err != nil || subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrCredential
	}
	return nil
}

// NewToken returns a 256-bit bearer secret. Persist only TokenDigest's result;
// the plaintext belongs only in the client's secure cookie or operator handoff.
func NewToken() (string, error) {
	data := make([]byte, tokenBytes)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// TokenDigest rejects noncanonical encodings and returns the storage/lookup key.
// Fast SHA-256 is appropriate for random tokens, never for human passwords.
func TokenDigest(token string) (string, error) {
	if len(token) != 43 {
		return "", ErrCredential
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(data) != tokenBytes {
		return "", ErrCredential
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
