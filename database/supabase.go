package database

import (
	"context"
	"fmt"
	"os"
	//"strings"
	"time"
	

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitSupabase() error {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Parse the configuration
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("unable to parse database URL: %v", err)
	}

	// Force SSL mode for Supabase
	config.ConnConfig.RuntimeParams["sslmode"] = "require"

	// Add connection pool settings
	config.MaxConns = 10
	config.MinConns = 2

	// Create connection pool
	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %v", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := DB.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %v", err)
	}

	fmt.Println("✅ Successfully connected to Supabase database")
	return nil
}