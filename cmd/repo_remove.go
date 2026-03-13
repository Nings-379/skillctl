package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var repoRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote repository",
	Long:  `Remove a remote repository from the database. This will also delete all skills indexed from this repository.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return doRepoRemove(name)
	},
}

func init() {
	repoCmd.AddCommand(repoRemoveCmd)
}

func doRepoRemove(name string) error {
	fmt.Printf("🗑️  Removing repository: %s\n", name)
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

	// Count skills in this repository
	skillCount, err := manager.GetQueries().CountSkillsByRepository(ctx, repo.ID)
	if err == nil {
		fmt.Printf("   Skills indexed: %d\n", skillCount)
	}

	fmt.Println()

	// Confirm removal
	fmt.Printf("Are you sure you want to remove '%s'? [y/N]: ", name)
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("❌ Removal cancelled.")
		return nil
	}

	fmt.Println()

	// Delete repository (this will cascade delete skills due to FOREIGN KEY)
	if err := manager.GetQueries().DeleteRepository(ctx, repo.ID); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	fmt.Println("✅ Repository removed successfully!")
	fmt.Printf("   Repository: %s\n", name)
	if skillCount > 0 {
		fmt.Printf("   Skills deleted: %d\n", skillCount)
	}

	return nil
}
