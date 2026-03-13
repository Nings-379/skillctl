package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all remote repositories",
	Long:  `List all configured remote skill repositories.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return doRepoList()
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
}

func doRepoList() error {
	fmt.Println("📋 Remote Repositories:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Initialize database
	manager, err := initDB()
	if err != nil {
		return err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Get all repositories
	repos, err := manager.GetQueries().ListAllRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories configured.")
		fmt.Println()
		fmt.Println("💡 Use 'skillctl repo add <url>' to add a repository")
		return nil
	}

	// Display repositories
	for i, repo := range repos {
		fmt.Printf("%d. %s\n", i+1, repo.Name)
		fmt.Printf("   URL: %s\n", repo.Url)

		if repo.Description.Valid && repo.Description.String != "" {
			fmt.Printf("   Description: %s\n", repo.Description.String)
		}

		fmt.Printf("   Default: %s", map[bool]string{true: "✓", false: "✗"}[repo.IsDefault == 1])
		fmt.Printf("   Priority: %d\n", repo.Priority)

		if repo.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, repo.CreatedAt); err == nil {
				fmt.Printf("   Added: %s\n", t.Format("2006-01-02 15:04:05"))
			}
		}

		if repo.LastSyncedAt.Valid && repo.LastSyncedAt.String != "" {
			if t, err := time.Parse(time.RFC3339, repo.LastSyncedAt.String); err == nil {
				fmt.Printf("   Last synced: %s\n", t.Format("2006-01-02 15:04:05"))
			}
		}

		if repo.SyncStatus.Valid && repo.SyncStatus.String != "" {
			status := repo.SyncStatus.String
			statusIcon := "⏳"
			switch status {
			case "success":
				statusIcon = "✅"
			case "failed":
				statusIcon = "❌"
			case "syncing":
				statusIcon = "🔄"
			}
			fmt.Printf("   Sync status: %s %s\n", statusIcon, status)
		}

		if repo.SyncError.Valid && repo.SyncError.String != "" {
			fmt.Printf("   Sync error: %s\n", repo.SyncError.String)
		}

		fmt.Println()
	}

	fmt.Printf("Total: %d repository(ies)\n", len(repos))
	fmt.Println()
	fmt.Println("💡 Use 'skillctl repo index <name>' to sync skills from a repository")

	return nil
}
