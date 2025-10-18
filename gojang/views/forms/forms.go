package forms

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Keep time import referenced until forms/models start using it explicitly
// This avoids "imported and not used" compile errors while keeping the import ready.
var _ time.Time

// LoginForm represents login form data
type LoginForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required,min=8"`
	Next     string `form:"next"`
}

// RegisterForm represents registration form data
type RegisterForm struct {
	Email           string `form:"email" validate:"required,email"`
	Password        string `form:"password" validate:"required,min=8"`
	PasswordConfirm string `form:"password_confirm" validate:"required,eqfield=Password"`
}

// UserForm represents user create/update form
type UserForm struct {
	Email       string `form:"email" validate:"required,email"`
	IsActive    bool   `form:"is_active"`
	IsStaff     bool   `form:"is_staff"`
	IsSuperuser bool   `form:"is_superuser"`
	Password    string `form:"password" validate:"omitempty,min=8"`
}

// PostForm represents post create/update form
type PostForm struct {
	Subject string `form:"subject" validate:"required,max=255"`
	Body    string `form:"body" validate:"required"`
}

// ProductForm represents product create/update form
// Uncomment when Product model exists
// type SampleProductForm struct {
// 	Name        string  `form:"name" validate:"required,max=255"`
// 	Description string  `form:"description" validate:"required"`
// 	Price       float64 `form:"price" validate:"required,gt=0"`
// 	Stock       int     `form:"stock" validate:"required,gte=0"`
// }

// Validate validates a form struct
func Validate(form interface{}) map[string]string {
	errors := make(map[string]string)

	err := validate.Struct(form)
	if err == nil {
		return errors
	}

	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		switch err.Tag() {
		case "required":
			errors[field] = "This field is required"
		case "email":
			errors[field] = "Invalid email address"
		case "min":
			errors[field] = "Minimum length is " + err.Param()
		case "eqfield":
			errors[field] = "Must match " + err.Param()
		default:
			errors[field] = "Invalid value"
		}
	}

	return errors
}
