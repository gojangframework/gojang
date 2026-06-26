package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gojangframework/gojang/app/gojang/models"
	"github.com/gojangframework/gojang/app/gojang/models/enttest"
	"github.com/gojangframework/gojang/app/gojang/utils"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
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

func TestRequireAuth_UnverifiedUserRedirectsToVerify(t *testing.T) {
	name := "middleware_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", name))
	t.Cleanup(func() { client.Close() })

	hash, err := utils.HashPassword("Password123!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	u, err := client.User.Create().
		SetID(uuid.New()).
		SetEmail("unverified@example.com").
		SetPasswordHash(hash).
		SetIsActive(true).
		SetIsEmailVerified(false).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create user error = %v", err)
	}

	sessions := scs.New()
	sessions.Lifetime = time.Hour
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	ctx, err := sessions.Load(req.Context(), "")
	if err != nil {
		t.Fatalf("session load error = %v", err)
	}
	sessions.Put(ctx, "user_id", u.ID.String())
	req = req.WithContext(ctx)

	protected := RequireAuth(sessions, client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/register-verify-email" {
		t.Fatalf("Location = %q, want /register-verify-email", got)
	}
}
