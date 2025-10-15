package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}
}

func TestCheckPassword_ValidPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	match, err := CheckPassword(hash, password)
	if err != nil {
		t.Fatalf("CheckPassword failed: %v", err)
	}

	if !match {
		t.Error("Expected password to match hash")
	}
}

func TestCheckPassword_InvalidPassword(t *testing.T) {
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	match, err := CheckPassword(hash, wrongPassword)
	if err != nil {
		t.Fatalf("CheckPassword failed: %v", err)
	}

	if match {
		t.Error("Expected password not to match hash")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "testpassword123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("First HashPassword failed: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Second HashPassword failed: %v", err)
	}

	// Due to random salt, hashes should be different
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (different salts)")
	}

	// But both should validate correctly
	match1, _ := CheckPassword(hash1, password)
	match2, _ := CheckPassword(hash2, password)

	if !match1 || !match2 {
		t.Error("Both hashes should validate the password")
	}
}
