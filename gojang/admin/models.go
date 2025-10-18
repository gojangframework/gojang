package admin

import (
	"context"
	"fmt"

	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/utils"
)

// BeforeSaveHook is called before saving a record (create or update)
type BeforeSaveHook func(ctx context.Context, data map[string]interface{}) error

// AfterLoadHook is called after loading records (for eager loading relations, etc.)
type AfterLoadHook func(ctx context.Context, query interface{}) interface{}

// ModelRegistration defines a simple model registration with hooks
type ModelRegistration struct {
	ModelType      interface{} // e.g., &models.User{}
	Icon           string
	NamePlural     string
	ListFields     []string
	HiddenFields   []string
	ReadonlyFields []string
	OptionalFields []string
	BeforeSave     BeforeSaveHook // Hook to transform data before save
	QueryModifier  AfterLoadHook  // Hook to modify query (e.g., eager load relations)
}

// RegisterModels registers all models with the admin registry
// This is the main entry point for setting up the admin panel
func RegisterModels(registry *Registry) {
	// Register User model - simple configuration!
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.User{},
		Icon:           "👤",
		NamePlural:     "Users",
		ListFields:     []string{"ID", "Email", "IsActive", "IsStaff", "CreatedAt"},
		HiddenFields:   []string{"PasswordHash"},
		ReadonlyFields: []string{"ID", "CreatedAt", "UpdatedAt", "LastLogin"},

		// Only special logic: hash passwords before saving
		BeforeSave: func(ctx context.Context, data map[string]interface{}) error {
			if password, ok := data["Password"].(string); ok && password != "" {
				hashedPassword, err := utils.HashPassword(password)
				if err != nil {
					return fmt.Errorf("hashing password: %w", err)
				}
				data["PasswordHash"] = hashedPassword
				delete(data, "Password") // Remove plain password
			}
			return nil
		},
	})

	// Register Post model - with author assignment!
	registry.RegisterModel(ModelRegistration{
		ModelType:      &models.Post{},
		Icon:           "📝",
		NamePlural:     "Posts",
		ListFields:     []string{"ID", "Subject", "Author", "CreatedAt"},
		ReadonlyFields: []string{"ID", "CreatedAt", "UpdatedAt"},

		// Set the author to the current user
		BeforeSave: func(ctx context.Context, data map[string]interface{}) error {
			// Get the current user from context
			user := middleware.GetUser(ctx)
			if user == nil {
				return fmt.Errorf("no authenticated user found")
			}
			// Set the author ID
			data["AuthorID"] = user.ID
			return nil
		},

		// Eager load author relationship
		QueryModifier: func(ctx context.Context, query interface{}) interface{} {
			if q, ok := query.(*models.PostQuery); ok {
				return q.WithAuthor()
			}
			return query
		},
	})

	// Register SampleProduct model - example for demonstration
	// Uncomment when SampleProduct model exists
	// registry.RegisterSampleModel(ModelRegistration{
	// 	ModelType:      &models.SampleProduct{},
	// 	Icon:           "📦",
	// 	NamePlural:     "SampleProducts",
	// 	ListFields:     []string{"ID", "Name", "Price", "Stock", "Description"},
	// 	ReadonlyFields: []string{"ID", "CreatedAt", "UpdatedAt"},
	// })

}
