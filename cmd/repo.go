package cmd

import (
	"fmt"
	"os"

	"skillctl/pkg/db"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage remote skill repositories",
	Long:  `Manage remote skill repositories including adding, listing, removing, and indexing repositories.`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}

// initDB initializes the database for repo commands
func initDB() (*db.Manager, error) {
	if err := db.EnsureDBExists(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	manager, err := db.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}

	return manager, nil
}

// closeDB closes the database connection
func closeDB(manager *db.Manager) {
	if manager != nil {
		if err := manager.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close database: %v\n", err)
		}
	}
}
