package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/gojangframework/gojang/gojang/config"
	"github.com/gojangframework/gojang/gojang/models/db"
	"github.com/gojangframework/gojang/gojang/models/user"
	"github.com/gojangframework/gojang/gojang/utils"

	"golang.org/x/term"
)

func main() {
	cfg := config.MustLoad()

	client, err := db.NewClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Check if superuser already exists
	exists, err := client.User.Query().Where(user.IsSuperuserEQ(true)).Exist(ctx)
	if err != nil {
		log.Fatalf("Failed to query database: %v", err)
	}

	if exists {
		log.Println("⚠️  A superuser already exists. Do you want to create another? (y/N)")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(response)) != "y" {
			log.Println("Aborted.")
			return
		}
	}

	// Display password requirements
	fmt.Println("\n📋 Password Requirements:")
	fmt.Println("   • At least 10 characters")
	fmt.Println("   • At least one uppercase letter")
	fmt.Println("   • At least one lowercase letter")
	fmt.Println("   • At least one special character (!@#$%^&*()_+-=[]{}|;:,.<>?)")
	fmt.Println()

	// Prompt for email
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	if email == "" {
		log.Fatal("Email is required")
	}

	// Check if email exists
	exists, err = client.User.Query().Where(user.EmailEQ(email)).Exist(ctx)
	if err != nil {
		log.Fatalf("Failed to query database: %v", err)
	}
	if exists {
		log.Fatalf("User with email %s already exists", email)
	}

	// Prompt for password
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	fmt.Println()

	password := string(passwordBytes)

	// Validate password complexity
	if err := utils.ValidatePasswordComplexity(password); err != nil {
		log.Fatalf("Password does not meet complexity requirements: %v", err)
	}

	// Hash password
	hash, err := utils.HashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create superuser
	u, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash(hash).
		SetIsActive(true).
		SetIsStaff(true).
		SetIsSuperuser(true).
		Save(ctx)

	if err != nil {
		log.Fatalf("Failed to create superuser: %v", err)
	}

	log.Printf("✅ Superuser created successfully: %s (ID: %d)", u.Email, u.ID)
}
