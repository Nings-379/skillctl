package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

// GetDBPath returns the path to the SQLite database file
func GetDBPath() (string, error) {
	var dbDir string
	
	if runtime.GOOS == "windows" {
		// Windows: %APPDATA%\skill
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		dbDir = filepath.Join(appData, "skill")
	} else {
		// Linux/macOS: ~/.cache/skill
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dbDir = filepath.Join(homeDir, ".cache", "skill")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create database directory: %w", err)
	}

	return filepath.Join(dbDir, "skill.db"), nil
}

// OpenDB opens the SQLite database connection
func OpenDB() (*sql.DB, error) {
	dbPath, err := GetDBPath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return db, nil
}

// InitDB initializes the database schema
func InitDB() error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Read and execute migrations
	migrationPath := filepath.Join("migrations", "0001_initial_schema.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if _, err := db.Exec(string(migrationSQL)); err != nil {
		// If error is about tables already existing, that's okay
		// The tables might already exist from a previous run
		if !isTableExistsError(err) {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}

// isTableExistsError checks if the error is due to tables already existing
func isTableExistsError(err error) bool {
	return err != nil && 
		(err.Error() == "table repositories already exists" || 
		 err.Error() == "table skills already exists" ||
		 err.Error() == "table tags already exists" ||
		 err.Error() == "table db_version already exists")
}

// EnsureDBExists ensures the database exists and is properly initialized
func EnsureDBExists() error {
	dbPath, err := GetDBPath()
	if err != nil {
		return err
	}

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Database doesn't exist, initialize it
		return InitDB()
	}

	// Database exists, verify it's accessible
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if tables exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='repositories'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check database structure: %w", err)
	}

	if count == 0 {
		// Tables don't exist, initialize
		return InitDB()
	}

	return nil
}