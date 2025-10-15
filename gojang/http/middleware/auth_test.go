package middleware

import (
	"context"
	"testing"

	"github.com/gojangframework/gojang/gojang/models"
)

func TestGetUser_WithUser(t *testing.T) {
	// Create a test user
	user := &models.User{
		Email:   "test@example.com",
		IsStaff: true,
	}

	// Add user to context
	ctx := context.WithValue(context.Background(), userContextKey, user)

	// Retrieve user
	retrievedUser := GetUser(ctx)

	if retrievedUser == nil {
		t.Fatal("Expected user to be retrieved from context")
	}

	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
	}

	if retrievedUser.IsStaff != user.IsStaff {
		t.Errorf("Expected IsStaff %v, got %v", user.IsStaff, retrievedUser.IsStaff)
	}
}

func TestGetUser_WithoutUser(t *testing.T) {
	ctx := context.Background()

	retrievedUser := GetUser(ctx)

	if retrievedUser != nil {
		t.Error("Expected nil user from empty context")
	}
}

func TestGetUser_WithWrongType(t *testing.T) {
	// Add wrong type to context
	ctx := context.WithValue(context.Background(), userContextKey, "not a user")

	retrievedUser := GetUser(ctx)

	if retrievedUser != nil {
		t.Error("Expected nil user when context contains wrong type")
	}
}
