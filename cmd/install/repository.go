package installcmd

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

	"github.com/seekthought/skill/pkg/db"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

func newSyncRepositoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync-repo [name]",
		Short: "Index and sync skills from a remote repository",
		Long:  `Index and sync skills from a remote repository to the local database. If no repository name is specified, the default repository is used.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoName := ""
			if len(args) > 0 {
				repoName = args[0]
			}
			return runRepositorySync(repoName)
		},
	}
}

func newDefaultRepositoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "default-repo <name>",
		Short: "Set a repository as default",
		Long:  `Set a repository as the default repository. The default repository is used when no specific repository is specified.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefaultRepository(args[0])
		},
	}
}

func runDefaultRepository(name string) error {
	fmt.Printf("⭐ Setting default repository: %s\n", name)
	fmt.Println()

	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()
	repo, err := manager.GetQueries().GetRepositoryByName(ctx, name)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", name)
	}

	fmt.Printf("   URL: %s\n", repo.Url)
	if repo.Description.Valid && repo.Description.String != "" {
		fmt.Printf("   Description: %s\n", repo.Description.String)
	}
	fmt.Println()

	if repo.IsDefault == 1 {
		fmt.Println("ℹ️  This repository is already set as default.")
		return nil
	}

	if err := manager.SetRepositoryAsDefaultByID(ctx, repo.ID); err != nil {
		return fmt.Errorf("failed to set default repository: %w", err)
	}

	fmt.Println("✅ Default repository set successfully!")
	fmt.Printf("   Repository: %s\n", name)
	fmt.Println()
	fmt.Println("💡 This repository will be used by default for operations that don't specify a repository")

	return nil
}

func runRepositorySync(repoName string) error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()

	var repo db.Repository
	if repoName == "" {
		defaultRepo, err := manager.GetQueries().GetDefaultRepository(ctx)
		if err != nil {
			return fmt.Errorf("no default repository found. Please specify a repository name or set a default repository")
		}
		repo = defaultRepo
		fmt.Printf("📥 Using default repository: %s\n", repo.Name)
	} else {
		repo, err = manager.GetQueries().GetRepositoryByName(ctx, repoName)
		if err != nil {
			return fmt.Errorf("repository '%s' not found", repoName)
		}
		fmt.Printf("📥 Indexing repository: %s\n", repo.Name)
	}

	fmt.Printf("   URL: %s\n", repo.Url)
	fmt.Println()

	if err := manager.UpdateRepositorySync(ctx, repo.ID, "syncing", ""); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	fmt.Println("🔄 Fetching skills from repository...")
	skills, err := fetchSkillsFromRepository(repo.Url)
	if err != nil {
		updateErr := manager.UpdateRepositorySync(ctx, repo.ID, "failed", err.Error())
		if updateErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update sync status: %v\n", updateErr)
		}
		return fmt.Errorf("failed to fetch skills: %w", err)
	}

	fmt.Printf("✓ Found %d skill(s)\n", len(skills))
	fmt.Println()
	fmt.Println("💾 Syncing skills to database...")

	syncedCount := 0
	updatedCount := 0
	for _, skill := range skills {
		existing, err := manager.GetQueries().GetSkillByRepositoryAndSlug(ctx, db.GetSkillByRepositoryAndSlugParams{
			RepositoryID: repo.ID,
			Slug:         skill.Slug,
		})

		now := time.Now().Format(time.RFC3339)
		if err == nil {
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

			if err := manager.ReplaceTagsForSkill(ctx, existing.ID, skill.Topics); err != nil {
				fmt.Printf("⚠️  Warning: Failed to update tags for '%s': %v\n", skill.Name, err)
			}

			updatedCount++
			continue
		}

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

		createdSkill, err := manager.GetQueries().GetSkillByRepositoryAndSlug(ctx, db.GetSkillByRepositoryAndSlugParams{
			RepositoryID: repo.ID,
			Slug:         skill.Slug,
		})
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to get created skill '%s': %v\n", skill.Name, err)
			continue
		}

		if err := manager.AddTagsToSkill(ctx, createdSkill.ID, skill.Topics); err != nil {
			fmt.Printf("⚠️  Warning: Failed to add tags for '%s': %v\n", skill.Name, err)
		}

		syncedCount++
	}

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

type remoteSkill struct {
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

func fetchSkillsFromRepository(url string) ([]remoteSkill, error) {
	if cmdutils.IsGitURL(url) {
		return fetchSkillsFromGitRepo(url)
	}

	return fetchSkillsFromAPI(url)
}

func fetchSkillsFromGitRepo(gitURL string) ([]remoteSkill, error) {
	tempDir, err := os.MkdirTemp("", "skill-index-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("   Cloning repository...")
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
			break
		}
		fmt.Printf("   ⚠️  Clone failed with '%s', trying alternative method...\n", cmdArgs[3])
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to clone repository after multiple attempts: %w", lastErr)
	}
	fmt.Println("   ✓ Repository cloned")

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
		return []remoteSkill{}, nil
	}

	fmt.Println("   Scanning skills directory...")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []remoteSkill
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

func parseSkillMetadataFromDir(skillPath, skillName, repoURL string) (remoteSkill, error) {
	var skill remoteSkill
	skill.Name = skillName
	skill.Slug = skillName
	skill.RepoURL = repoURL

	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if data, err := os.ReadFile(skillMdPath); err == nil {
		parseSkillMetadataFromMarkdown(string(data), &skill)
	}

	if skill.Description == "" {
		skillYamlPath := filepath.Join(skillPath, "skill.yaml")
		if data, err := os.ReadFile(skillYamlPath); err == nil {
			parseSkillMetadataFromYAML(string(data), &skill)
		}
	}

	if skill.Description == "" {
		skill.Description = fmt.Sprintf("Skill: %s", skillName)
	}

	skill.CreatedAt = time.Now().Format(time.RFC3339)
	skill.UpdatedAt = time.Now().Format(time.RFC3339)

	return skill, nil
}

func parseSkillMetadataFromMarkdown(markdown string, skill *remoteSkill) {
	lines := strings.Split(markdown, "\n")
	inFrontMatter := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "---") {
			if i == 0 {
				inFrontMatter = true
				continue
			}
			if inFrontMatter {
				break
			}
		}

		if !inFrontMatter {
			continue
		}

		if strings.HasPrefix(trimmedLine, "name:") {
			skill.Name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:")), `"'`)
		}
		if strings.HasPrefix(trimmedLine, "description:") {
			skill.Description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:")), `"'`)
		}
		if strings.HasPrefix(trimmedLine, "version:") {
			skill.Version = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:")), `"'`)
		}
		if strings.HasPrefix(trimmedLine, "author:") {
			skill.Author = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:")), `"'`)
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

func parseSkillMetadataFromYAML(yaml string, skill *remoteSkill) {
	lines := strings.Split(yaml, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "name:") {
			skill.Name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:")), `"'`)
		}
		if strings.HasPrefix(trimmedLine, "description:") {
			skill.Description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:")), `"'`)
		}
		if strings.HasPrefix(trimmedLine, "version:") {
			skill.Version = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:")), `"'`)
		}
		if strings.HasPrefix(trimmedLine, "author:") {
			skill.Author = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:")), `"'`)
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

func fetchSkillsFromAPI(url string) ([]remoteSkill, error) {
	client := &http.Client{Timeout: 30 * time.Second}

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
		Skills []remoteSkill `json:"skills"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	for i := range response.Skills {
		if response.Skills[i].Category.Name != "" {
			response.Skills[i].Topics = append([]string{response.Skills[i].Category.Name}, response.Skills[i].Topics...)
		}
	}

	return response.Skills, nil
}
