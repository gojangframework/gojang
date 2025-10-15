package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gojangframework/gojang/gojang/models"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"           // Postgres driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// NewClient creates a new Ent client from a database URL
func NewClient(databaseURL string) (*models.Client, error) {
	var (
		db         *sql.DB
		err        error
		driverName string
	)

	// Parse database URL to determine driver
	if strings.HasPrefix(databaseURL, "sqlite://") {
		driverName = dialect.SQLite
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		db, err = sql.Open("sqlite3", dbPath+"?_fk=1")
	} else if strings.HasPrefix(databaseURL, "postgres://") {
		driverName = dialect.Postgres
		db, err = sql.Open("postgres", databaseURL)
	} else {
		return nil, fmt.Errorf("unsupported database URL scheme: %s", databaseURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed opening database: %w", err)
	}

	// Create Ent driver
	drv := entsql.OpenDB(driverName, db)
	client := models.NewClient(models.Driver(drv))

	return client, nil
}

// AutoMigrate runs automatic migrations (creates/updates tables)
func AutoMigrate(ctx context.Context, client *models.Client) error {
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("failed creating schema resources: %w", err)
	}
	return nil
}
