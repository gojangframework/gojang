package utils

import (
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

// HashPassword hashes a plaintext password using Argon2id
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, DefaultParams)
}

// CheckPassword verifies a password against an Argon2id hash
func CheckPassword(hash, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
