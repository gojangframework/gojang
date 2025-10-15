package middleware

import (
	"context"
	"net/http"

	"github.com/gojangframework/gojang/gojang/models"

	"github.com/alexedwards/scs/v2"
)

type contextKey string

const userContextKey contextKey = "user"

// RequireAuth middleware ensures user is authenticated
func RequireAuth(sm *scs.SessionManager, client *models.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sm.GetInt(r.Context(), "user_id")
			if userID == 0 {
				// Check if htmx request
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/login?next="+r.URL.Path, http.StatusSeeOther)
				return
			}

			// Load user and add to context
			user, err := client.User.Get(r.Context(), userID)
			if err != nil || !user.IsActive {
				sm.Destroy(r.Context())
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireStaff middleware ensures user is staff
func RequireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil || !user.IsStaff {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoadUser middleware loads the user from session if available (doesn't require auth)
func LoadUser(sm *scs.SessionManager, client *models.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sm.GetInt(r.Context(), "user_id")
			if userID != 0 {
				// Load user and add to context
				user, err := client.User.Get(r.Context(), userID)
				if err == nil && user.IsActive {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					r = r.WithContext(ctx)
				} else {
					// Invalid session, destroy it
					sm.Destroy(r.Context())
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUser retrieves the authenticated user from context
func GetUser(ctx context.Context) *models.User {
	user, _ := ctx.Value(userContextKey).(*models.User)
	return user
}
