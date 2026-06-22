package utils

import (
	"errors"
	"unicode"

	"github.com/alexedwards/argon2id"
)

// DefaultParams provides secure defaults for Argon2id
var DefaultParams = &argon2id.Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// Password complexity requirements
const (
	MinPasswordLength = 10
)

// HashPassword hashes a plaintext password using Argon2id
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, DefaultParams)
}

// CheckPassword verifies a password against an Argon2id hash
func CheckPassword(hash, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

// ValidatePasswordComplexity checks if a password meets complexity requirements:
// - At least 10 characters
// - Contains at least one uppercase letter
// - Contains at least one lowercase letter
// - Contains at least one special character
func ValidatePasswordComplexity(password string) error {
	if len(password) < MinPasswordLength {
		return errors.New("password must be at least 10 characters long")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}
