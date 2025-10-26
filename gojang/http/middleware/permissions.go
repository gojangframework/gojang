package middleware

import (
	"net/http"

	"github.com/gojangframework/gojang/gojang/models"
	"github.com/google/uuid"
)

// Permission constants
const (
	PermissionViewAnyPost   = "view_any_post"
	PermissionEditAnyPost   = "edit_any_post"
	PermissionDeleteAnyPost = "delete_any_post"
	PermissionManageUsers   = "manage_users"
)

// CanUser checks if the authenticated user can perform an action
func CanUser(r *http.Request, permission string) bool {
	user := GetUser(r.Context())
	if user == nil {
		return false
	}

	// Staff can do everything
	if user.IsStaff {
		return true
	}

	// Add more granular permission logic here
	// For now, regular users have limited permissions
	switch permission {
	case PermissionViewAnyPost:
		return true // Anyone can view posts
	case PermissionEditAnyPost, PermissionDeleteAnyPost, PermissionManageUsers:
		return false // Only staff
	default:
		return false
	}
}

// OwnsResource checks if user owns a resource (e.g., their own post)
func OwnsResource(r *http.Request, resourceUserID uuid.UUID) bool {
	user := GetUser(r.Context())
	if user == nil {
		return false
	}

	return user.ID == resourceUserID || user.IsStaff
}

// GetUserFromRequest retrieves the authenticated user from the request context
// This is a convenience wrapper around GetUser
func GetUserFromRequest(r *http.Request) *models.User {
	return GetUser(r.Context())
}
