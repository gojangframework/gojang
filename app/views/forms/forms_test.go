package forms

import (
	"testing"
)

func TestValidate_RegisterForm_ValidPassword(t *testing.T) {
	form := RegisterForm{
		Email:           "test@example.com",
		Password:        "Password123!",
		PasswordConfirm: "Password123!",
	}

	errors := Validate(form)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid form, got: %v", errors)
	}
}

func TestValidate_RegisterForm_WeakPassword(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedError string
	}{
		{
			name:          "too short",
			password:      "Pass1!",
			expectedError: "password must be at least 10 characters long",
		},
		{
			name:          "no uppercase",
			password:      "password123!",
			expectedError: "password must contain at least one uppercase letter",
		},
		{
			name:          "no lowercase",
			password:      "PASSWORD123!",
			expectedError: "password must contain at least one lowercase letter",
		},
		{
			name:          "no special char",
			password:      "Password123",
			expectedError: "password must contain at least one special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := RegisterForm{
				Email:           "test@example.com",
				Password:        tt.password,
				PasswordConfirm: tt.password,
			}

			errors := Validate(form)
			if len(errors) == 0 {
				t.Errorf("Expected validation error for %s, got none", tt.name)
				return
			}

			if errors["Password"] != tt.expectedError {
				t.Errorf("Expected error %q, got %q", tt.expectedError, errors["Password"])
			}
		})
	}
}

func TestValidate_RegisterForm_PasswordMismatch(t *testing.T) {
	form := RegisterForm{
		Email:           "test@example.com",
		Password:        "Password123!",
		PasswordConfirm: "DifferentPass123!",
	}

	errors := Validate(form)
	if len(errors) == 0 {
		t.Error("Expected validation error for mismatched passwords")
		return
	}

	if _, exists := errors["PasswordConfirm"]; !exists {
		t.Errorf("Expected PasswordConfirm error, got errors: %v", errors)
	}
}

func TestValidate_UserForm_ValidPassword(t *testing.T) {
	form := UserForm{
		Email:    "test@example.com",
		Password: "Password123!",
	}

	errors := Validate(form)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid form, got: %v", errors)
	}
}

func TestValidate_UserForm_EmptyPasswordOK(t *testing.T) {
	// Empty password should be OK for UserForm (update scenario)
	form := UserForm{
		Email:    "test@example.com",
		Password: "",
	}

	errors := Validate(form)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for empty password in UserForm, got: %v", errors)
	}
}

func TestValidate_UserForm_WeakPasswordWhenProvided(t *testing.T) {
	form := UserForm{
		Email:    "test@example.com",
		Password: "weak",
	}

	errors := Validate(form)
	if len(errors) == 0 {
		t.Error("Expected validation error for weak password")
		return
	}

	if _, exists := errors["Password"]; !exists {
		t.Errorf("Expected Password error, got errors: %v", errors)
	}
}

func TestValidate_LoginForm_NoComplexityCheck(t *testing.T) {
	// LoginForm should not validate password complexity (only during registration)
	form := LoginForm{
		Email:    "test@example.com",
		Password: "anypassword",
	}

	errors := Validate(form)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for LoginForm, got: %v", errors)
	}
}
