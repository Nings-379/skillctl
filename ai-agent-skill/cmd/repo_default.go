package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var repoDefaultCmd = &cobra.Command{
	Use:   "default <name>",
	Short: "Set a repository as default",
	Long:  `Set a repository as the default repository. The default repository is used when no specific repository is specified.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return doRepoDefault(name)
	},
}

func init() {
	repoCmd.AddCommand(repoDefaultCmd)
}

func doRepoDefault(name string) error {
	fmt.Printf("⭐ Setting default repository: %s\n", name)
	fmt.Println()

	// Initialize database
	manager, err := initDB()
	if err != nil {
		return err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Get repository
	repo, err := manager.GetQueries().GetRepositoryByName(ctx, name)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", name)
	}

	// Display repository information
	fmt.Printf("   URL: %s\n", repo.Url)
	if repo.Description.Valid && repo.Description.String != "" {
		fmt.Printf("   Description: %s\n", repo.Description.String)
	}
	fmt.Println()

	// Check if already default
	if repo.IsDefault == 1 {
		fmt.Println("ℹ️  This repository is already set as default.")
		return nil
	}

	// Set as default
	if err := manager.SetRepositoryAsDefaultByID(ctx, repo.ID); err != nil {
		return fmt.Errorf("failed to set default repository: %w", err)
	}

	fmt.Println("✅ Default repository set successfully!")
	fmt.Printf("   Repository: %s\n", name)
	fmt.Println()
	fmt.Println("💡 This repository will be used by default for operations that don't specify a repository")

	return nil
}
