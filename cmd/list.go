package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	listRemote  bool
	listProject bool
	scanDir     string // Scan specified skill directory
	listRepo    string // Search in repository database
)

var listCmd = &cobra.Command{
	Use:   "list [url]",
	Short: "List all installed skills",
	Long: `Display a list of all skills installed in the local skills directory, current project, or remote repository.
When using --remote, provide a URL to list skills from any remote Git repository.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doList(args)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listRemote, "remote", "R", false, "List skills from remote repository (provide URL as argument)")
	listCmd.Flags().BoolVarP(&listProject, "project", "p", false, "List skills in current project directory")
	listCmd.Flags().StringVarP(&scanDir, "skills-dir", "s", "", "Scan skills from a specific directory")
	listCmd.Flags().StringVarP(&listRepo, "repo", "r", "", "List skills from repository database")
}

// CustomSkill represents a custom skill
type CustomSkill struct {
	Name        string
	Description string
	Version     string
	Tags        []string
	Author      string
	Path        string
	InstalledAt string
}

// doList Execute list command, scan and display local, project or remote skills
func doList(args []string) error {
	if listRepo != "" {
		return doListFromRepoDB()
	}
	if listRemote {
		if len(args) == 0 {
			return fmt.Errorf("please provide a URL when using --remote flag\nUsage: skillctl list -R <url>")
		}
		return doListRemote(args[0])
	}
	if listProject {
		return doListProject()
	}
	if scanDir != "" {
		return doListScanDir(scanDir)
	}
	if len(args) > 0 {
		// If URL is provided without -R flag, treat as remote URL
		return doListRemote(args[0])
	}
	return doListLocal()
}

// doListLocal List local skills
func doListLocal() error {
	fmt.Println("Installed Skills:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Get installed skills from storage index
	installedSkills, err := storage.ListSkills()
	if err != nil {
		return fmt.Errorf("failed to list installed skills: %w", err)
	}

	// Get skills directory
	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get skills directory: %w", err)
	}

	// Convert to CustomSkill format
	var skills []CustomSkill
	for _, skillMeta := range installedSkills {
		skillPath := filepath.Join(skillsDir, skillMeta.Name)
		skill := CustomSkill{
			Name:        skillMeta.Name,
			Description: skillMeta.Description,
			Version:     skillMeta.Version,
			Tags:        skillMeta.Tags,
			Author:      skillMeta.Author,
			Path:        skillPath,
			InstalledAt: skillMeta.InstalledAt,
		}
		skills = append(skills, skill)
	}

	// If no skills in index, scan ~/.skills directory
	if len(skills) == 0 {
		skills, err = scanCustomSkills(skillsDir)
		if err != nil {
			return fmt.Errorf("failed to scan skills directory: %w", err)
		}
	}

	if len(skills) == 0 {
		fmt.Println("No skills installed.")
		fmt.Println()
		fmt.Println("💡 Tip: Install a skill using: skillctl install <git-url>")
		fmt.Println("💡 Tip: Add a skill from local path: skillctl add <local-path>")
		fmt.Println("💡 Tip: List remote skills using: skillctl list -R <url>")
		return nil
	}

	// Display found skills (list format)
	for i, skill := range skills {
		fmt.Printf("\n")
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description != "" {
			fmt.Printf("   Description: %s\n", skill.Description)
		}

		if skill.Path != "" {
			fmt.Printf("   Path: %s\n", skill.Path)
		}

		if skill.InstalledAt != "" {
			fmt.Printf("   Installed: %s\n", skill.InstalledAt)
		}

		if len(skill.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d skill(s) installed\n", len(skills))
	fmt.Println()
	fmt.Println("💡 Use 'skillctl add <skill-name> -p <location>' to add a skill to your project")

	return nil
}

// doListProject List skills in project directory
func doListProject() error {
	fmt.Println("Project Skills:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Determine project directory
	baseDir := projectDir
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Get all VS Code recognized skill directories
	projectSkillsDirs := []string{
		filepath.Join(baseDir, ".github", "skills"),
		filepath.Join(baseDir, ".agents", "skills"),
		filepath.Join(baseDir, ".claude", "skills"),
		filepath.Join(baseDir, ".copilot", "skills"),
		filepath.Join(baseDir, "examples", "skills"),
	}

	// Scan all directories
	var allSkills []CustomSkill
	for _, dir := range projectSkillsDirs {
		skills, err := scanCustomSkills(dir)
		if err != nil {
			// Directory may not exist, skip
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) == 0 {
		fmt.Println("No skills found in project directory.")
		fmt.Println()
		fmt.Printf("Project directory: %s\n", baseDir)
		fmt.Println()
		fmt.Println("💡 Tip: Add a skill to project using: skillctl add <skill-name> -p <location>")
		fmt.Println("💡 Tip: List installed skills using: skillctl list")
		fmt.Println("💡 Tip: List remote skills using: skillctl list -R <url>")
		return nil
	}

	// Display found skills (list format)
	for i, skill := range allSkills {
		fmt.Printf("\n")
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description != "" {
			fmt.Printf("   Description: %s\n", skill.Description)
		}

		if skill.Path != "" {
			// Display path relative to project directory
			relPath, err := filepath.Rel(baseDir, skill.Path)
			if err == nil {
				fmt.Printf("   Location: %s\n", relPath)
			} else {
				fmt.Printf("   Location: %s\n", skill.Path)
			}
		}

		if skill.Version != "" {
			fmt.Printf("   Version: %s\n", skill.Version)
		}

		if skill.Author != "" {
			fmt.Printf("   Author: %s\n", skill.Author)
		}

		if skill.InstalledAt != "" {
			fmt.Printf("   Installed: %s\n", skill.InstalledAt)
		}

		if len(skill.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d skill(s) in project\n", len(allSkills))
	fmt.Printf("Project directory: %s\n", baseDir)
	fmt.Println()
	fmt.Println("💡 Use 'skillctl add <skill-name> -p <location>' to add more skills to your project")

	return nil
}

// doListScanDir Scan skills in specified directory
func doListScanDir(dirPath string) error {
	// Get absolute path
	absPath := dirPath
	if !filepath.IsAbs(dirPath) {
		var err error
		absPath, err = filepath.Abs(dirPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	fmt.Println("Skills in Directory:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Directory: %s\n\n", absPath)

	// Scan specified directory
	skills, err := scanCustomSkills(absPath)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in this directory.")
		fmt.Println()
		fmt.Println("💡 Tip: Make sure the directory contains skill subdirectories with SKILL.md or skill.yaml files")
		return nil
	}

	// Display found skills (list format)
	for i, skill := range skills {
		fmt.Printf("\n")
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description != "" {
			fmt.Printf("   Description: %s\n", skill.Description)
		}

		if skill.Path != "" {
			// Display path relative to scan directory
			relPath, err := filepath.Rel(absPath, skill.Path)
			if err == nil {
				fmt.Printf("   Location: %s\n", relPath)
			} else {
				fmt.Printf("   Location: %s\n", skill.Path)
			}
		}

		if skill.Version != "" {
			fmt.Printf("   Version: %s\n", skill.Version)
		}

		if skill.Author != "" {
			fmt.Printf("   Author: %s\n", skill.Author)
		}

		if skill.InstalledAt != "" {
			fmt.Printf("   Installed: %s\n", skill.InstalledAt)
		}

		if len(skill.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d skill(s) found\n", len(skills))
	fmt.Println()

	return nil
}

// doListRemote List skills from remote repository
func doListRemote(remoteURL string) error {
	fmt.Println("Remote Skills:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	if remoteURL == "" {
		return fmt.Errorf("no URL provided. Please provide a URL: skillctl list -R <url>")
	}

	fmt.Printf("📦 URL: %s\n", remoteURL)
	fmt.Println()

	// Check if it's a GitHub repository URL
	if strings.Contains(remoteURL, "github.com") {
		return doListGitRepo(remoteURL)
	}

	// Check if it's a Git repository URL
	if strings.HasPrefix(remoteURL, "http://") || strings.HasPrefix(remoteURL, "https://") {
		return doListGitRepo(remoteURL)
	}

	// Otherwise, treat as REST API
	return doListAPI(remoteURL)
}

// doListFromRepoDB List skills from repository database
func doListFromRepoDB() error {
	fmt.Println("Repository Skills:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Initialize database
	manager, err := initDB()
	if err != nil {
		return err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Get repository
	repo, err := manager.GetQueries().GetRepositoryByName(ctx, listRepo)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", listRepo)
	}

	fmt.Printf("📦 Repository: %s\n", repo.Name)
	fmt.Printf("   URL: %s\n", repo.Url)
	if repo.Description.Valid && repo.Description.String != "" {
		fmt.Printf("   Description: %s\n", repo.Description.String)
	}
	fmt.Println()

	// Get skills from this repository
	skills, err := manager.GetQueries().ListSkillsByRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in this repository.")
		return nil
	}

	// Display skills
	for i, skill := range skills {
		fmt.Printf("\n")
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description.Valid && skill.Description.String != "" {
			fmt.Printf("   📝 Description: %s\n", skill.Description.String)
		}

		if skill.Version.Valid && skill.Version.String != "" {
			fmt.Printf("   📌 Version: %s\n", skill.Version.String)
		}

		if skill.Author.Valid && skill.Author.String != "" {
			fmt.Printf("   👤 Author: %s\n", skill.Author.String)
		}

		if skill.Category.Valid && skill.Category.String != "" {
			fmt.Printf("   📂 Category: %s\n", skill.Category.String)
		}

		if skill.RepoUrl.Valid && skill.RepoUrl.String != "" {
			fmt.Printf("   🔗 URL: %s\n", skill.RepoUrl.String)
		}

		if skill.Stars.Valid && skill.Stars.Int64 > 0 {
			fmt.Printf("   ⭐ Stars: %d\n", skill.Stars.Int64)
		}

		// Get tags
		tags, err := manager.GetQueries().GetTagsBySkillID(ctx, skill.ID)
		if err == nil && len(tags) > 0 {
			tagNames := make([]string, len(tags))
			for j, tag := range tags {
				tagNames[j] = tag.Tag
			}
			fmt.Printf("   🏷️  Tags: %s\n", strings.Join(tagNames, ", "))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d skill(s)\n", len(skills))
	fmt.Println()
	fmt.Printf("💡 Use 'skillctl search <keyword> -r %s' to search skills in this repository\n", listRepo)

	return nil
}

// doListAPI List skills from REST API
func doListAPI(url string) error {
	fmt.Printf("Fetching from: %s\n\n", url)

	// Get GitHub API response
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch remote skills: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch remote skills: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse remote API response
	var skillsData struct {
		Skills []struct {
			ID             string   `json:"id"`
			Slug           string   `json:"slug"`
			Name           string   `json:"name"`
			Description    string   `json:"description"`
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
			Category       struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"category"`
		} `json:"skills"`
	}

	if err := json.Unmarshal(body, &skillsData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Debug: Print response body length and skill count
	fmt.Fprintf(os.Stderr, "Debug: API response size: %d bytes, skills count: %d\n", len(body), len(skillsData.Skills))

	// Convert to CustomSkill format
	var skills []CustomSkill
	for _, item := range skillsData.Skills {
		skill := CustomSkill{
			Name:        item.Name,
			Description: item.Description,
			Tags:        item.Topics,
			Author:      item.RepoOwner,
			Path:        item.RepoURL,
		}

		// Add category information to tags
		if item.Category.Name != "" {
			skill.Tags = append([]string{item.Category.Name}, skill.Tags...)
		}

		skills = append(skills, skill)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in remote repository.")
		return nil
	}

	// Display remote skills (list format)
	for i, skill := range skills {
		fmt.Printf("\n")
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description != "" {
			fmt.Printf("   Description: %s\n", skill.Description)
		}

		if skill.Path != "" {
			fmt.Printf("   URL: %s\n", skill.Path)
		}

		if skill.Version != "" {
			fmt.Printf("   Version: %s\n", skill.Version)
		}

		if skill.Author != "" {
			fmt.Printf("   Author: %s\n", skill.Author)
		}

		if len(skill.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d skill(s) in remote repository\n", len(skills))
	fmt.Println()
	fmt.Println("💡 Tip: Install a skill using: skillctl install <git-url>")
	fmt.Println("💡 Tip: List local skills using: skillctl list")

	return nil
}

// doListGitRepo List skills in skills subdirectory from Git repository URL
func doListGitRepo(gitURL string) error {
	fmt.Printf("Repository: %s\n", gitURL)
	fmt.Println()

	// Check if it's a GitHub repository
	if strings.Contains(gitURL, "github.com") {
		// Try to use GitHub API (faster, no need to clone)
		skills, err := fetchSkillsFromGitHubAPI(gitURL)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch via GitHub API: %v\n", err)
			fmt.Println("Falling back to git clone...")
			fmt.Println()

			// Fallback: Clone repository
			return doListGitRepoViaClone(gitURL)
		}

		if len(skills) == 0 {
			fmt.Println("No skills found in the repository.")
			return nil
		}

		// Display found skills
		for i, skill := range skills {
			fmt.Printf("\n")
			fmt.Printf("%d. %s\n", i+1, skill.Name)

			if skill.Description != "" {
				fmt.Printf("   Description: %s\n", skill.Description)
			}

			if len(skill.Tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
			}
		}

		fmt.Println()
		fmt.Printf("Total: %d skill(s) found in repository\n", len(skills))
		fmt.Println()
		fmt.Printf("💡 To install a specific skill, use: skillctl install %s <skill-name>\n", gitURL)

		return nil
	}

	// For non-GitHub repositories, use git clone directly
	return doListGitRepoViaClone(gitURL)
}

// fetchSkillsFromGitHubAPI Fetch skills directory content using GitHub API
func fetchSkillsFromGitHubAPI(gitURL string) ([]CustomSkill, error) {
	// Parse GitHub URL to get owner and repo
	parts := strings.Split(gitURL, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL")
	}
	owner := parts[len(parts)-2]
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")

	// Try to get from skills directory
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills", owner, repo)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try .skills directory
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/.skills", owner, repo)
		resp, err = http.Get(apiURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("skills directory not found")
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse GitHub API response
	var items []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}

	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	// Filter out directories and fetch their descriptions
	var skills []CustomSkill
	for _, item := range items {
		if item.Type == "dir" {
			skill := CustomSkill{
				Name: item.Name,
				Path: fmt.Sprintf("%s/%s", gitURL, item.Name),
			}

			// Try to fetch description from SKILL.md or skill.yaml
			description, tags, err := fetchSkillDescriptionFromGitHubAPI(owner, repo, item.Name)
			if err == nil {
				skill.Description = description
				skill.Tags = tags
			}

			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// fetchSkillDescriptionFromGitHubAPI Fetch skill description from SKILL.md or skill.yaml using GitHub API
func fetchSkillDescriptionFromGitHubAPI(owner, repo, skillName string) (string, []string, error) {
	// Try to fetch SKILL.md first
	skillMdURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills/%s/SKILL.md", owner, repo, skillName)
	description, tags, err := fetchGitHubFileContent(skillMdURL)
	if err == nil && description != "" {
		return description, tags, nil
	}

	// Try skill.yaml
	skillYamlURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills/%s/skill.yaml", owner, repo, skillName)
	description, tags, err = fetchGitHubFileContent(skillYamlURL)
	if err == nil && description != "" {
		return description, tags, nil
	}

	// Try README.md as fallback
	readmeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills/%s/README.md", owner, repo, skillName)
	description, tags, err = fetchGitHubFileContent(readmeURL)
	if err == nil {
		return description, tags, nil
	}

	return "", nil, fmt.Errorf("no description file found")
}

// fetchGitHubFileContent Fetch file content from GitHub API
func fetchGitHubFileContent(url string) (string, []string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("file not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	// Parse GitHub API response
	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.Unmarshal(body, &fileContent); err != nil {
		return "", nil, err
	}

	// Decode base64 content
	if fileContent.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
		if err != nil {
			return "", nil, err
		}
		content := string(decoded)

		// Parse metadata from the content
		return parseSkillMetadataFromContent(content)
	}

	return "", nil, fmt.Errorf("unsupported encoding")
}

// parseSkillMetadataFromContent Parse skill metadata from file content string
func parseSkillMetadataFromContent(content string) (string, []string, error) {
	lines := strings.Split(content, "\n")

	var description string
	var tags []string
	inFrontMatter := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for YAML front matter
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
			if strings.HasPrefix(trimmedLine, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:"))
				description = strings.Trim(description, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "tags:") {
				tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
				if tagsStr != "" && tagsStr != "[]" {
					tagsStr = strings.Trim(tagsStr, "[]")
					tagList := strings.Split(tagsStr, ",")
					for i, tag := range tagList {
						tagList[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
					}
					tags = tagList
				}
			}
		}
	}

	// If no description found in front matter, try to extract from the first paragraph
	if description == "" {
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			// Skip headers, empty lines, and front matter markers
			if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") || strings.HasPrefix(trimmedLine, "---") {
				continue
			}
			// Take the first non-empty, non-header line as description
			if trimmedLine != "" {
				description = trimmedLine
				// Limit description length
				if len(description) > 200 {
					description = description[:200] + "..."
				}
				break
			}
		}
	}

	return description, tags, nil
}

// doListGitRepoViaClone List skills by cloning repository
func doListGitRepoViaClone(gitURL string) error {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "skill-list-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository (shallow clone) - suppress output
	fmt.Println("📥 Fetching repository information...")
	cmd := exec.Command("git", "clone", "--depth=1", "--quiet", gitURL, tempDir)
	// Suppress git output by redirecting to nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	fmt.Println("✓ Repository fetched successfully")
	fmt.Println()

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
		fmt.Println("No 'skills' or '.skills' directory found in the repository.")
		fmt.Println()
		fmt.Println("💡 Tip: The repository should contain a 'skills' or '.skills' directory with skill subdirectories.")
		return nil
	}

	// Scan skills directory
	skills, err := scanCustomSkills(skillsDir)
	if err != nil {
		return fmt.Errorf("failed to scan skills directory: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in the skills directory.")
		return nil
	}

	// Display found skills
	for i, skill := range skills {
		fmt.Printf("\n")
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description != "" {
			fmt.Printf("   Description: %s\n", skill.Description)
		}

		if len(skill.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d skill(s) found in repository\n", len(skills))
	fmt.Println()
	fmt.Printf("💡 To install a specific skill, use: skillctl install %s <skill-name>\n", gitURL)

	return nil
}

// scanCustomSkills Scan custom skills in specified folder
func scanCustomSkills(skillsDir string) ([]CustomSkill, error) {
	var skills []CustomSkill

	// Check if directory exists
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return skills, nil // Directory doesn't exist, return empty list
	}

	// Read directory
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())

		// Try to parse skill
		skill, err := parseCustomSkill(skillPath, entry.Name())
		if err != nil {
			// Parse failed, skip this skill
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// parseCustomSkill Parse custom skill metadata from skill directory
func parseCustomSkill(skillPath, skillName string) (CustomSkill, error) {
	var skill CustomSkill
	skill.Name = skillName
	skill.Path = skillPath

	// Try to read SKILL.md first (YAML front matter format)
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if data, err := os.ReadFile(skillMdPath); err == nil {
		markdown := string(data)
		lines := strings.Split(markdown, "\n")

		// Check if it's YAML front matter format (starts with ---)
		inFrontMatter := false
		for i, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			if strings.HasPrefix(trimmedLine, "---") {
				if i == 0 {
					// Start front matter
					inFrontMatter = true
					continue
				} else if inFrontMatter {
					// End front matter
					inFrontMatter = false
					break
				}
			}

			if inFrontMatter {
				// Parse fields in YAML front matter
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

				if strings.HasPrefix(trimmedLine, "tags:") {
					// Handle tags array
					tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
					if tagsStr != "" && tagsStr != "[]" {
						tagsStr = strings.Trim(tagsStr, "[]")
						tags := strings.Split(tagsStr, ",")
						for i, tag := range tags {
							tags[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
						}
						skill.Tags = tags
					}
				}

				if strings.HasPrefix(trimmedLine, "author:") {
					skill.Author = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:"))
					skill.Author = strings.Trim(skill.Author, `"'`)
				}
			}
		}
	}

	// If SKILL.md has no information, try skill.yaml
	if skill.Description == "" {
		skillYamlPath := filepath.Join(skillPath, "skill.yaml")
		if data, err := os.ReadFile(skillYamlPath); err == nil {
			lines := strings.Split(string(data), "\n")

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

				if strings.HasPrefix(trimmedLine, "tags:") {
					tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
					if tagsStr != "" && tagsStr != "[]" {
						tagsStr = strings.Trim(tagsStr, "[]")
						tags := strings.Split(tagsStr, ",")
						for i, tag := range tags {
							tags[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
						}
						skill.Tags = tags
					}
				}

				if strings.HasPrefix(trimmedLine, "author:") {
					skill.Author = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:"))
					skill.Author = strings.Trim(skill.Author, `"'`)
				}
			}
		}
	}

	// If no description found, use directory name
	if skill.Description == "" {
		skill.Description = fmt.Sprintf("Skill: %s", skillName)
	}

	return skill, nil
}
