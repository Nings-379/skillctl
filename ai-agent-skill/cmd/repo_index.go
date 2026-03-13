package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"skillctl/pkg/db"

	"github.com/spf13/cobra"
)

var repoIndexCmd = &cobra.Command{
	Use:   "index [name]",
	Short: "Index/sync skills from a remote repository",
	Long:  `Index and sync skills from a remote repository to the local database. If no repository name is specified, the default repository will be used.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := ""
		if len(args) > 0 {
			repoName = args[0]
		}
		return doRepoIndex(repoName)
	},
}

func init() {
	repoCmd.AddCommand(repoIndexCmd)
}

func doRepoIndex(repoName string) error {
	// Initialize database
	manager, err := initDB()
	if err != nil {
		return err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Get repository
	var repo db.Repository
	if repoName == "" {
		// Use default repository
		defaultRepo, err := manager.GetQueries().GetDefaultRepository(ctx)
		if err != nil {
			return fmt.Errorf("no default repository found. Please specify a repository name or set a default repository")
		}
		repo = defaultRepo
		fmt.Printf("📥 Using default repository: %s\n", repo.Name)
	} else {
		// Use specified repository
		repo, err = manager.GetQueries().GetRepositoryByName(ctx, repoName)
		if err != nil {
			return fmt.Errorf("repository '%s' not found", repoName)
		}
		fmt.Printf("📥 Indexing repository: %s\n", repo.Name)
	}

	fmt.Printf("   URL: %s\n", repo.Url)
	fmt.Println()

	// Update sync status to syncing
	if err := manager.UpdateRepositorySync(ctx, repo.ID, "syncing", ""); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Fetch skills from repository
	fmt.Println("🔄 Fetching skills from repository...")
	skills, err := fetchSkillsFromRepository(repo.Url)
	if err != nil {
		// Update sync status to failed
		updateErr := manager.UpdateRepositorySync(ctx, repo.ID, "failed", err.Error())
		if updateErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update sync status: %v\n", updateErr)
		}
		return fmt.Errorf("failed to fetch skills: %w", err)
	}

	fmt.Printf("✓ Found %d skill(s)\n", len(skills))
	fmt.Println()

	// Sync skills to database
	fmt.Println("💾 Syncing skills to database...")
	syncedCount := 0
	updatedCount := 0

	for _, skill := range skills {
		// Check if skill already exists
		existing, err := manager.GetQueries().GetSkillByRepositoryAndSlug(ctx, db.GetSkillByRepositoryAndSlugParams{
			RepositoryID: repo.ID,
			Slug:         skill.Slug,
		})

		now := time.Now().Format(time.RFC3339)
		if err == nil {
			// Update existing skill
			updateParams := db.UpdateSkillParams{
				Description:    sql.NullString{String: skill.Description, Valid: skill.Description != ""},
				Version:        sql.NullString{String: skill.Version, Valid: skill.Version != ""},
				Author:         sql.NullString{String: skill.Author, Valid: skill.Author != ""},
				RepoUrl:        sql.NullString{String: skill.RepoURL, Valid: skill.RepoURL != ""},
				RepoOwner:      sql.NullString{String: skill.RepoOwner, Valid: skill.RepoOwner != ""},
				RepoName:       sql.NullString{String: skill.RepoName, Valid: skill.RepoName != ""},
				Stars:          sql.NullInt64{Int64: int64(skill.Stars), Valid: true},
				Forks:          sql.NullInt64{Int64: int64(skill.Forks), Valid: true},
				Downloads:      sql.NullInt64{Int64: int64(skill.Downloads), Valid: true},
				Language:       sql.NullString{String: skill.Language, Valid: skill.Language != ""},
				License:        sql.NullString{String: skill.License, Valid: skill.License != ""},
				Homepage:       sql.NullString{String: skill.Homepage, Valid: skill.Homepage != ""},
				Category:       sql.NullString{String: skill.Category.Name, Valid: skill.Category.Name != ""},
				SecurityStatus: sql.NullString{String: skill.SecurityStatus, Valid: skill.SecurityStatus != ""},
				UpdatedAt:      sql.NullString{String: now, Valid: true},
				ID:             existing.ID,
			}
			if err := manager.GetQueries().UpdateSkill(ctx, updateParams); err != nil {
				fmt.Printf("⚠️  Warning: Failed to update skill '%s': %v\n", skill.Name, err)
				continue
			}

			// Update tags
			if err := manager.ReplaceTagsForSkill(ctx, existing.ID, skill.Topics); err != nil {
				fmt.Printf("⚠️  Warning: Failed to update tags for '%s': %v\n", skill.Name, err)
			}

			updatedCount++
		} else {
			// Create new skill
			createParams := db.CreateSkillParams{
				RepositoryID:   repo.ID,
				Name:           skill.Name,
				Slug:           skill.Slug,
				Description:    sql.NullString{String: skill.Description, Valid: skill.Description != ""},
				Version:        sql.NullString{String: skill.Version, Valid: skill.Version != ""},
				Author:         sql.NullString{String: skill.Author, Valid: skill.Author != ""},
				RepoUrl:        sql.NullString{String: skill.RepoURL, Valid: skill.RepoURL != ""},
				RepoOwner:      sql.NullString{String: skill.RepoOwner, Valid: skill.RepoOwner != ""},
				RepoName:       sql.NullString{String: skill.RepoName, Valid: skill.RepoName != ""},
				Stars:          sql.NullInt64{Int64: int64(skill.Stars), Valid: true},
				Forks:          sql.NullInt64{Int64: int64(skill.Forks), Valid: true},
				Downloads:      sql.NullInt64{Int64: int64(skill.Downloads), Valid: true},
				Language:       sql.NullString{String: skill.Language, Valid: skill.Language != ""},
				License:        sql.NullString{String: skill.License, Valid: skill.License != ""},
				Homepage:       sql.NullString{String: skill.Homepage, Valid: skill.Homepage != ""},
				Category:       sql.NullString{String: skill.Category.Name, Valid: skill.Category.Name != ""},
				SecurityStatus: sql.NullString{String: skill.SecurityStatus, Valid: skill.SecurityStatus != ""},
				CreatedAt:      sql.NullString{String: skill.CreatedAt, Valid: skill.CreatedAt != ""},
				UpdatedAt:      sql.NullString{String: skill.UpdatedAt, Valid: skill.UpdatedAt != ""},
				IndexedAt:      now,
			}

			if err := manager.GetQueries().CreateSkill(ctx, createParams); err != nil {
				fmt.Printf("⚠️  Warning: Failed to create skill '%s': %v\n", skill.Name, err)
				continue
			}

			// Get the created skill ID
			createdSkill, err := manager.GetQueries().GetSkillByRepositoryAndSlug(ctx, db.GetSkillByRepositoryAndSlugParams{
				RepositoryID: repo.ID,
				Slug:         skill.Slug,
			})
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to get created skill '%s': %v\n", skill.Name, err)
				continue
			}

			// Add tags
			if err := manager.AddTagsToSkill(ctx, createdSkill.ID, skill.Topics); err != nil {
				fmt.Printf("⚠️  Warning: Failed to add tags for '%s': %v\n", skill.Name, err)
			}

			syncedCount++
		}
	}

	// Update sync status to success
	if err := manager.UpdateRepositorySync(ctx, repo.ID, "success", ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to update sync status: %v\n", err)
	}

	fmt.Println()
	fmt.Println("✅ Index completed successfully!")
	fmt.Printf("   New skills: %d\n", syncedCount)
	fmt.Printf("   Updated skills: %d\n", updatedCount)
	fmt.Printf("   Total skills: %d\n", syncedCount+updatedCount)
	fmt.Println()
	fmt.Println("💡 Use 'skillctl search <keyword>' to search for skills")

	return nil
}

// RemoteSkill represents a skill from the remote API
type RemoteSkill struct {
	ID             string   `json:"id"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Version        string   `json:"version"`
	Author         string   `json:"author"`
	RepoURL        string   `json:"repoUrl"`
	RepoOwner      string   `json:"repoOwner"`
	RepoName       string   `json:"repoName"`
	Stars          int      `json:"stars"`
	Forks          int      `json:"forks"`
	Topics         []string `json:"topics"`
	Language       string   `json:"language"`
	SecurityStatus string   `json:"securityStatus"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	Downloads      int      `json:"downloads"`
	License        string   `json:"license"`
	Homepage       string   `json:"homepage"`
	Category       struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"category"`
}

// fetchSkillsFromRepository fetches skills from a remote repository
func fetchSkillsFromRepository(url string) ([]RemoteSkill, error) {
	// Check if it's a Git repository URL
	if isGitURL(url) {
		return fetchSkillsFromGitRepo(url)
	}

	// Otherwise, treat it as a REST API
	return fetchSkillsFromAPI(url)
}

// fetchSkillsFromGitRepo fetches skills from a Git repository
func fetchSkillsFromGitRepo(gitURL string) ([]RemoteSkill, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "skill-index-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository (shallow clone for speed)
	fmt.Println("   Cloning repository...")
	// Try different clone strategies
	cloneCommands := [][]string{
		{"git", "clone", "--depth=1", gitURL, tempDir},
		{"git", "clone", gitURL, tempDir},
	}

	var lastErr error
	for _, cmdArgs := range cloneCommands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		lastErr = cmd.Run()
		if lastErr == nil {
			break // Success
		}
		fmt.Printf("   ⚠️  Clone failed with '%s', trying alternative method...\n", cmdArgs[3])
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to clone repository after multiple attempts: %w", lastErr)
	}
	fmt.Println("   ✓ Repository cloned")

	// Find skills directory
	skillsDirs := []string{
		filepath.Join(tempDir, "skills"),
		filepath.Join(tempDir, ".skills"),
	}

	var skillsDir string
	for _, dir := range skillsDirs {
		if _, err := os.Stat(dir); err == nil {
			skillsDir = dir
			break
		}
	}

	if skillsDir == "" {
		return []RemoteSkill{}, nil // No skills directory found, return empty list
	}

	// Scan skills directory and parse skill metadata
	fmt.Println("   Scanning skills directory...")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []RemoteSkill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())
		skill, err := parseSkillMetadataFromDir(skillPath, entry.Name(), gitURL)
		if err != nil {
			fmt.Printf("   ⚠️  Warning: Failed to parse skill '%s': %v\n", entry.Name(), err)
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// parseSkillMetadataFromDir parses skill metadata from directory
func parseSkillMetadataFromDir(skillPath, skillName, repoURL string) (RemoteSkill, error) {
	var skill RemoteSkill
	skill.Name = skillName
	skill.Slug = skillName
	skill.RepoURL = repoURL

	// Try to parse SKILL.md first
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if data, err := os.ReadFile(skillMdPath); err == nil {
		parseSkillMetadataFromMarkdown(string(data), &skill)
	}

	// If SKILL.md doesn't have description, try skill.yaml
	if skill.Description == "" {
		skillYamlPath := filepath.Join(skillPath, "skill.yaml")
		if data, err := os.ReadFile(skillYamlPath); err == nil {
			parseSkillMetadataFromYAML(string(data), &skill)
		}
	}

	// Set default values if needed
	if skill.Description == "" {
		skill.Description = fmt.Sprintf("Skill: %s", skillName)
	}

	// Set timestamps
	skill.CreatedAt = time.Now().Format(time.RFC3339)
	skill.UpdatedAt = time.Now().Format(time.RFC3339)

	return skill, nil
}

// parseSkillMetadataFromMarkdown parses skill metadata from markdown content
func parseSkillMetadataFromMarkdown(markdown string, skill *RemoteSkill) {
	lines := strings.Split(markdown, "\n")

	inFrontMatter := false
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "---") {
			if i == 0 {
				inFrontMatter = true
				continue
			} else if inFrontMatter {
				inFrontMatter = false
				break
			}
		}

		if inFrontMatter {
			if strings.HasPrefix(trimmedLine, "name:") {
				skill.Name = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:"))
				skill.Name = strings.Trim(skill.Name, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "description:") {
				skill.Description = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:"))
				skill.Description = strings.Trim(skill.Description, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "version:") {
				skill.Version = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:"))
				skill.Version = strings.Trim(skill.Version, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "author:") {
				skill.Author = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:"))
				skill.Author = strings.Trim(skill.Author, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "tags:") {
				tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
				if tagsStr != "" && tagsStr != "[]" {
					tagsStr = strings.Trim(tagsStr, "[]")
					tags := strings.Split(tagsStr, ",")
					for i, tag := range tags {
						tags[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
					}
					skill.Topics = tags
				}
			}
		}
	}
}

// parseSkillMetadataFromYAML parses skill metadata from YAML content
func parseSkillMetadataFromYAML(yaml string, skill *RemoteSkill) {
	lines := strings.Split(yaml, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "name:") {
			skill.Name = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:"))
			skill.Name = strings.Trim(skill.Name, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "description:") {
			skill.Description = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:"))
			skill.Description = strings.Trim(skill.Description, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "version:") {
			skill.Version = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:"))
			skill.Version = strings.Trim(skill.Version, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "author:") {
			skill.Author = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:"))
			skill.Author = strings.Trim(skill.Author, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "tags:") {
			tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
			if tagsStr != "" && tagsStr != "[]" {
				tagsStr = strings.Trim(tagsStr, "[]")
				tags := strings.Split(tagsStr, ",")
				for i, tag := range tags {
					tags[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
				}
				skill.Topics = tags
			}
		}
	}
}

// fetchSkillsFromAPI fetches skills from a REST API
func fetchSkillsFromAPI(url string) ([]RemoteSkill, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response struct {
		Skills []RemoteSkill `json:"skills"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Add category to tags for easier searching
	for i := range response.Skills {
		if response.Skills[i].Category.Name != "" {
			response.Skills[i].Topics = append([]string{response.Skills[i].Category.Name}, response.Skills[i].Topics...)
		}
	}

	return response.Skills, nil
}
