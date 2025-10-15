package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gojangframework/gojang/gojang/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"           // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate <up|down>")
		fmt.Println("  up   - Apply all pending migrations")
		fmt.Println("  down - Rollback the last migration")
		os.Exit(1)
	}

	command := os.Args[1]

	// Load config
	cfg := config.MustLoad()

	// Parse database URL and connect
	var db *sql.DB
	var err error
	var driver string
	var databaseName string

	if strings.HasPrefix(cfg.DatabaseURL, "sqlite://") {
		dbPath := strings.TrimPrefix(cfg.DatabaseURL, "sqlite://")
		db, err = sql.Open("sqlite3", dbPath+"?_fk=1")
		driver = "sqlite3"
		databaseName = "sqlite3"
	} else if strings.HasPrefix(cfg.DatabaseURL, "postgres://") {
		db, err = sql.Open("postgres", cfg.DatabaseURL)
		driver = "postgres"
		databaseName = "postgres"
	} else {
		log.Fatalf("Unsupported database URL scheme: %s", cfg.DatabaseURL)
	}

	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create driver instance based on database type
	var m *migrate.Migrate
	if driver == "sqlite3" {
		driverInstance, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			log.Fatalf("Failed to create SQLite driver instance: %v", err)
		}
		m, err = migrate.NewWithDatabaseInstance(
			"file://gojang/models/migrations",
			databaseName,
			driverInstance,
		)
		if err != nil {
			log.Fatalf("Failed to create migrate instance: %v", err)
		}
	} else if driver == "postgres" {
		driverInstance, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Fatalf("Failed to create PostgreSQL driver instance: %v", err)
		}
		m, err = migrate.NewWithDatabaseInstance(
			"file://gojang/models/migrations",
			databaseName,
			driverInstance,
		)
		if err != nil {
			log.Fatalf("Failed to create migrate instance: %v", err)
		}
	}

	// Execute command
	switch command {
	case "up":
		if err := m.Up(); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("✅ No pending migrations")
			} else {
				log.Fatalf("Failed to run migrations: %v", err)
			}
		} else {
			fmt.Println("✅ All migrations applied successfully")
		}

	case "down":
		if err := m.Steps(-1); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("⚠️  No migrations to rollback")
			} else if strings.Contains(err.Error(), "file does not exist") {
				fmt.Println("⚠️  No migrations to rollback (database is empty)")
			} else {
				log.Fatalf("Failed to rollback migration: %v", err)
			}
		} else {
			fmt.Println("✅ Last migration rolled back successfully")
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: migrate <up|down>")
		os.Exit(1)
	}
}
