package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"skillctl/pkg/db"

	"github.com/spf13/cobra"
)

var (
	repoName        string
	repoDescription string
	repoDefault     bool
	repoPriority    int
)

var repoAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a remote repository",
	Long:  `Add a remote skill repository to the database. The repository can then be indexed to sync skills.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		return doRepoAdd(url)
	},
}

func init() {
	repoCmd.AddCommand(repoAddCmd)
	repoAddCmd.Flags().StringVarP(&repoName, "name", "n", "", "Repository name (default: auto-generated)")
	repoAddCmd.Flags().StringVarP(&repoDescription, "description", "d", "", "Repository description")
	repoAddCmd.Flags().BoolVarP(&repoDefault, "default", "D", false, "Set as default repository")
	repoAddCmd.Flags().IntVarP(&repoPriority, "priority", "p", 0, "Repository priority (higher = more preferred)")
}

func doRepoAdd(url string) error {
	fmt.Printf("📦 Adding repository: %s\n", url)
	fmt.Println()

	// Initialize database
	manager, err := initDB()
	if err != nil {
		return err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Check if repository with same URL already exists
	_, err = manager.GetQueries().GetRepositoryByName(ctx, repoName)
	if err == nil {
		return fmt.Errorf("repository with name '%s' already exists", repoName)
	}

	// Generate name if not provided
	name := repoName
	if name == "" {
		name = generateRepoName(url)
	}

	// Check if name is already taken
	_, err = manager.GetQueries().GetRepositoryByName(ctx, name)
	if err == nil {
		return fmt.Errorf("repository with name '%s' already exists", name)
	}

	// Create repository parameters
	now := time.Now().Format(time.RFC3339)
	params := db.CreateRepositoryParams{
		Name:        name,
		Url:         url,
		Description: sql.NullString{String: repoDescription, Valid: repoDescription != ""},
		IsDefault:   0,
		Priority:    int64(repoPriority),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// If setting as default, handle transaction
	if repoDefault {
		err = manager.Transaction(ctx, func(q *db.Queries) error {
			// Unset all default repositories
			if err := q.SetRepositoryAsDefault(ctx); err != nil {
				return err
			}

			// Set this repository as default
			params.IsDefault = 1
			return q.CreateRepository(ctx, params)
		})
	} else {
		err = manager.GetQueries().CreateRepository(ctx, params)
	}

	if err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	fmt.Println("✅ Repository added successfully!")
	fmt.Printf("   Name: %s\n", name)
	fmt.Printf("   URL: %s\n", url)
	if repoDescription != "" {
		fmt.Printf("   Description: %s\n", repoDescription)
	}
	if repoDefault {
		fmt.Println("   Default: ✓")
	}
	if repoPriority > 0 {
		fmt.Printf("   Priority: %d\n", repoPriority)
	}
	fmt.Println()
	fmt.Println("💡 Use 'skillctl repo index <name>' to sync skills from this repository")

	return nil
}

// generateRepoName generates a repository name from URL
func generateRepoName(url string) string {
	// Simple implementation: use the last part of the URL
	// e.g., https://github.com/user/repo -> repo
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove .git suffix if present
		if strings.HasSuffix(lastPart, ".git") {
			lastPart = lastPart[:len(lastPart)-4]
		}
		return lastPart
	}
	return "unknown"
}
