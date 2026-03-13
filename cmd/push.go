package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skillctl/pkg/db"
	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

const (
	remoteRepository = "https://github.com/Nings-379/ninskills.git"
	githubAPIBase    = "https://api.github.com/repos/Nings-379/ninskills"
)

var (
	pushRemoteURL   string // Remote repository URL for -R flag
	pushRepoName    string // Repository name for -r flag
)

var pushCmd = &cobra.Command{
	Use:   "push [skill-name]",
	Short: "Push a skill to local .skills or remote repository",
	Long: `Push a skill from examples/skills to local .skills directory (default) or remote GitHub repository.

Usage:
  push <name>              - Push to local .skills directory
  push <name> -R <url>     - Push to remote repository with URL
  push <name> -r <repo>    - Push to remote repository by repo name (from database)

By default, pushes to local ~/.skills directory and updates the index.
Use -R to specify a remote repository URL directly.
Use -r to specify a repository name that exists in the database.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		return doPush(skillName)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVarP(&pushRemoteURL, "remote-url", "R", "", "Push to remote repository with URL")
	pushCmd.Flags().StringVarP(&pushRepoName, "remote-repo", "r", "", "Push to remote repository by name (from database)")
}

func doPush(skillName string) error {
	// Check for conflicting flags
	if pushRemoteURL != "" && pushRepoName != "" {
		return fmt.Errorf("cannot use both -R (remote-url) and -r (remote-repo) flags together")
	}

	// Determine push mode
	var remoteURL string
	var pushMode string

	if pushRemoteURL != "" {
		remoteURL = pushRemoteURL
		pushMode = "remote-url"
	} else if pushRepoName != "" {
		// Fetch repository URL from database
		repo, err := getRepositoryByName(pushRepoName)
		if err != nil {
			return fmt.Errorf("failed to get repository '%s' from database: %w", pushRepoName, err)
		}
		remoteURL = repo.Url
		pushMode = "remote-repo"
		fmt.Printf("📂 Repository '%s' resolved to URL: %s\n", pushRepoName, remoteURL)
	} else {
		pushMode = "local"
	}

	// Define list of directories to search
	searchDirs := []string{
		filepath.Join(".github", "skills"),
		filepath.Join(".agents", "skills"),
		filepath.Join(".copilot", "skills"),
		filepath.Join("examples", "skills"),
	}

	// Search for skill in multiple directories
	var skillPath string
	var foundDir string

	for _, dir := range searchDirs {
		possiblePath := filepath.Join(dir, skillName)
		if _, err := os.Stat(possiblePath); err == nil {
			skillPath = possiblePath
			foundDir = dir
			break
		}
	}

	// If not found, list all available skills
	if skillPath == "" {
		var allAvailableSkills []string

		// Collect skills from all directories
		for _, dir := range searchDirs {
			if entries, err := os.ReadDir(dir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						allAvailableSkills = append(allAvailableSkills, entry.Name())
					}
				}
			}
		}

		fmt.Printf("Skill '%s' does not exist.\n\n", skillName)
		if len(allAvailableSkills) > 0 {
			fmt.Println("Available skills in project:")
			for _, skill := range allAvailableSkills {
				fmt.Printf("  - %s\n", skill)
			}
		} else {
			fmt.Println("No skills found in project directories (.github/skills, .agents/skills, .copilot/skills, examples/skills)")
		}
		return fmt.Errorf("skill not found: %s", skillName)
	}

	fmt.Printf("📦 Pushing skill: %s\n", skillName)
	fmt.Printf("From: %s\n", skillPath)
	fmt.Printf("Source directory: %s\n", foundDir)
	fmt.Printf("Push mode: %s\n", pushMode)
	fmt.Println()

	// Validate skill package structure
	fmt.Println("📤 Validating skill package...")
	if err := validateSkillPackage(skillPath); err != nil {
		return fmt.Errorf("skill validation failed: %w", err)
	}
	fmt.Println("✓ Skill structure is valid")

	// Execute push based on mode
	switch pushMode {
	case "local":
		return doPushLocal(skillName, skillPath)
	case "remote-url", "remote-repo":
		return doPushRemote(skillName, skillPath, remoteURL)
	default:
		return fmt.Errorf("unknown push mode: %s", pushMode)
	}
}

// doPushLocal Push to local .skills folder
func doPushLocal(skillName, skillPath string) error {
	// Get skills directory
	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get skills directory: %w", err)
	}

	// Build .skills directory path
	localSkillDir := filepath.Join(skillsDir, skillName)

	fmt.Printf("📁 Target directory: %s\n", localSkillDir)
	fmt.Println()

	// Check if target directory already exists
	if _, err := os.Stat(localSkillDir); err == nil {
		fmt.Printf("⚠️  Warning: Skill '%s' already exists in ~/.skills\n", skillName)
		fmt.Printf("Overwriting existing skill...\n")
	}

	// Create target directory
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy files
	fmt.Println("📤 Copying files to local .skills folder...")
	fileCount := 0

	err = filepath.Walk(skillPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(skillPath, filePath)
		if err != nil {
			return err
		}

		// Build target path
		targetPath := filepath.Join(localSkillDir, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Copy file
		if err := copyFile(filePath, targetPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", relPath, err)
		}

		fileCount++
		fmt.Printf("  ✓ %s\n", relPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Successfully copied %d file(s)\n", fileCount)

	// Parse skill metadata
	fmt.Println("📋 Parsing skill metadata...")
	skillMeta, err := parseSkillMetadata(localSkillDir, skillName)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not parse skill metadata: %v\n", err)
		// Use default metadata
		skillMeta = storage.SkillMeta{
			Name:        skillName,
			Description: fmt.Sprintf("Skill from examples/skills/%s", skillName),
			InstalledAt: time.Now().Format(time.RFC3339),
		}
	} else {
		fmt.Println("✓ Skill metadata parsed")
	}

	// Update storage index file
	fmt.Println("📋 Updating skill index...")
	if err := storage.AddSkill(skillName, skillMeta); err != nil {
		return fmt.Errorf("failed to add skill to index: %w", err)
	}
	fmt.Println("✓ Index updated")

	fmt.Println()
	fmt.Printf("✅ Skill '%s' has been successfully pushed to local .skills folder!\n", skillName)
	fmt.Printf("Location: %s\n", localSkillDir)
	fmt.Println()
	fmt.Println("💡 The skill is now installed and available for use.")

	return nil
}

// doPushRemote Push to remote GitHub repository
func doPushRemote(skillName, skillPath, repoURL string) error {
	// Parse GitHub URL to get owner and repo
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid GitHub URL: %s", repoURL)
	}
	owner := parts[len(parts)-2]
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")

	// Build GitHub API base URL
	githubAPIBase := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	fmt.Printf("📤 Remote repository: %s\n", repoURL)
	fmt.Printf("📤 GitHub API: %s\n", githubAPIBase)
	fmt.Println()

	// Get GitHub token (from environment variable)
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("⚠️  Warning: GITHUB_TOKEN not set. Using unauthenticated API (may have rate limits).")
		fmt.Println("💡 Tip: Set GITHUB_TOKEN environment variable for higher rate limits.")
	}

	// Detect default branch
	defaultBranch, err := detectDefaultBranch(githubAPIBase, token)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not detect default branch, using 'main': %v\n", err)
		defaultBranch = "main"
	}
	fmt.Printf("📂 Default branch: %s\n", defaultBranch)
	fmt.Println()

	// Walk through all files in skill directory
	fmt.Println("📤 Uploading files to remote repository...")
	fileCount := 0

	err = filepath.Walk(skillPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(skillPath, filePath)
		if err != nil {
			return err
		}

		// Build remote path
		remotePath := fmt.Sprintf("skills/%s/%s", skillName, filepath.ToSlash(relPath))

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", relPath, err)
		}

		// Upload file
		if err := uploadFile(githubAPIBase, remotePath, content, token, defaultBranch); err != nil {
			return fmt.Errorf("failed to upload file %s: %w", relPath, err)
		}

		fileCount++
		fmt.Printf("  ✓ %s\n", remotePath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Successfully uploaded %d file(s)\n", fileCount)
	fmt.Printf("Remote repository: %s\n", repoURL)
	fmt.Printf("View at: %s/tree/%s/skills/%s\n", strings.TrimSuffix(repoURL, ".git"), defaultBranch, skillName)

	return nil
}

// detectDefaultBranch detects the default branch of a GitHub repository
func detectDefaultBranch(githubAPIBase, token string) (string, error) {
	url := fmt.Sprintf("%s", githubAPIBase)

	var resp *http.Response
	var err error

	if token != "" {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
		resp, err = http.DefaultClient.Do(req)
	} else {
		resp, err = http.Get(url)
	}

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get repository info (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.DefaultBranch == "" {
		return "", fmt.Errorf("default branch not found in repository info")
	}

	return result.DefaultBranch, nil
}

// uploadFile Upload file to GitHub repository
func uploadFile(githubAPIBase, path string, content []byte, token string, branch string) error {
	// Check if file already exists
	checkURL := fmt.Sprintf("%s/contents/%s?ref=%s", githubAPIBase, path, branch)

	var resp *http.Response
	var err error

	if token != "" {
		req, _ := http.NewRequest("GET", checkURL, nil)
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
		resp, err = http.DefaultClient.Do(req)
	} else {
		resp, err = http.Get(checkURL)
	}

	if err != nil {
		return err
	}

	// Parse response to get SHA (if file exists)
	var sha string
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			SHA string `json:"sha"`
		}
		if err := json.Unmarshal(body, &result); err == nil {
			sha = result.SHA
		}
	} else {
		resp.Body.Close()
	}

	// Prepare upload data
	encodedContent := base64.StdEncoding.EncodeToString(content)
	uploadData := map[string]interface{}{
		"message": fmt.Sprintf("Update %s", path),
		"content": encodedContent,
		"branch":  branch,
	}

	// Add SHA if file already exists
	if sha != "" {
		uploadData["sha"] = sha
	}

	jsonData, err := json.Marshal(uploadData)
	if err != nil {
		return err
	}

	// Send PUT request
	putURL := fmt.Sprintf("%s/contents/%s", githubAPIBase, path)
	req, err := http.NewRequest("PUT", putURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload file (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// getRepositoryByName fetches repository information from the database by name
func getRepositoryByName(name string) (*db.Repository, error) {
	manager, err := initDB()
	if err != nil {
		return nil, err
	}
	defer closeDB(manager)

	queries := manager.GetQueries()
	repo, err := queries.GetRepositoryByName(context.Background(), name)
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

// validateSkillPackage Validate skill package structure
func validateSkillPackage(path string) error {
	// Check if SKILL.md or skill.yaml exists
	skillMd := filepath.Join(path, "SKILL.md")
	skillYaml := filepath.Join(path, "skill.yaml")

	if _, err := os.Stat(skillMd); os.IsNotExist(err) {
		if _, err := os.Stat(skillYaml); os.IsNotExist(err) {
			return fmt.Errorf("neither SKILL.md nor skill.yaml found in skill package")
		}
	}
	return nil
}
