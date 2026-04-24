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

	return applyMigrations(db)
}

// EnsureDBExists ensures the database exists and is properly initialized
func EnsureDBExists() error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return applyMigrations(db)
}

func applyMigrations(db *sql.DB) error {
	migrationsDir, err := getMigrationsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		migrationPath := filepath.Join(migrationsDir, entry.Name())
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		if _, err := db.Exec(string(migrationSQL)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func getMigrationsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to determine database source path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")), nil
}
