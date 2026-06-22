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

func TestValidatePasswordComplexity_Valid(t *testing.T) {
	validPasswords := []string{
		"Password123!",
		"MyP@ssw0rd",
		"C0mpl3x!Pass",
		"Str0ng#Password",
		"Test1234!@",
		"A1b2C3d4!@#$",
	}

	for _, password := range validPasswords {
		err := ValidatePasswordComplexity(password)
		if err != nil {
			t.Errorf("Expected password %q to be valid, got error: %v", password, err)
		}
	}
}

func TestValidatePasswordComplexity_TooShort(t *testing.T) {
	shortPasswords := []string{
		"Pass1!",
		"Abc123!",
		"Short1!",
	}

	for _, password := range shortPasswords {
		err := ValidatePasswordComplexity(password)
		if err == nil {
			t.Errorf("Expected password %q to fail (too short), but it passed", password)
		}
		if err.Error() != "password must be at least 10 characters long" {
			t.Errorf("Expected 'too short' error for %q, got: %v", password, err)
		}
	}
}

func TestValidatePasswordComplexity_NoUppercase(t *testing.T) {
	passwords := []string{
		"password123!",
		"myp@ssw0rd",
		"test1234!@",
	}

	for _, password := range passwords {
		err := ValidatePasswordComplexity(password)
		if err == nil {
			t.Errorf("Expected password %q to fail (no uppercase), but it passed", password)
		}
		if err.Error() != "password must contain at least one uppercase letter" {
			t.Errorf("Expected 'no uppercase' error for %q, got: %v", password, err)
		}
	}
}

func TestValidatePasswordComplexity_NoLowercase(t *testing.T) {
	passwords := []string{
		"PASSWORD123!",
		"MYP@SSW0RD",
		"TEST1234!@",
	}

	for _, password := range passwords {
		err := ValidatePasswordComplexity(password)
		if err == nil {
			t.Errorf("Expected password %q to fail (no lowercase), but it passed", password)
		}
		if err.Error() != "password must contain at least one lowercase letter" {
			t.Errorf("Expected 'no lowercase' error for %q, got: %v", password, err)
		}
	}
}

func TestValidatePasswordComplexity_NoSpecialChar(t *testing.T) {
	passwords := []string{
		"Password123",
		"MyPassword0",
		"Test1234Abc",
	}

	for _, password := range passwords {
		err := ValidatePasswordComplexity(password)
		if err == nil {
			t.Errorf("Expected password %q to fail (no special char), but it passed", password)
		}
		if err.Error() != "password must contain at least one special character" {
			t.Errorf("Expected 'no special character' error for %q, got: %v", password, err)
		}
	}
}

func TestValidatePasswordComplexity_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedError string
	}{
		{
			name:          "exactly 10 chars with all requirements",
			password:      "Passw0rd!@",
			expectedError: "",
		},
		{
			name:          "very long password",
			password:      "ThisIsAVeryLongPassword123!@#WithManyCharacters",
			expectedError: "",
		},
		{
			name:          "empty password",
			password:      "",
			expectedError: "password must be at least 10 characters long",
		},
		{
			name:          "multiple special characters",
			password:      "P@ssw0rd!#$%",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordComplexity(tt.password)
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("Expected no error for %q, got: %v", tt.password, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for %q, got none", tt.password)
				} else if err.Error() != tt.expectedError {
					t.Errorf("Expected error %q for %q, got: %v", tt.expectedError, tt.password, err)
				}
			}
		})
	}
}
