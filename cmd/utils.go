package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"skillctl/pkg/storage"
)

// Shared command line parameter variables
var (
	projectDir     string // Project directory
	agentSkillsDir string // Agent skills directory
)

// extractRepoName Extract repository name from Git URL
func extractRepoName(gitURL string) (string, error) {
	gitURL = strings.TrimSuffix(gitURL, ".git")
	parts := strings.Split(gitURL, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid git URL")
	}
	repoName := parts[len(parts)-1]
	if repoName == "" {
		return "", fmt.Errorf("invalid git URL: empty repository name")
	}
	return repoName, nil
}

// cloneRepository Clone Git repository with optimizations for large repositories
func cloneRepository(gitURL, destPath string) error {
	// Check if directory already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination directory already exists: %s", destPath)
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Configure Git for better network stability
	// http.lowSpeedLimit: Disable low speed limit (default is 0, but some systems have issues)
	// http.postBuffer: Increase buffer size for large repositories
	configCmds := [][]string{
		{"git", "config", "--global", "http.lowSpeedLimit", "0"},
		{"git", "config", "--global", "http.postBuffer", "524288000"},
	}
	for _, cfg := range configCmds {
		_ = exec.Command(cfg[0], cfg[1:]...).Run()
	}

	// Use shallow clone with optimizations for large repositories
	// --depth=1: Clone only the latest commit (shallow clone)
	// --single-branch: Clone only the default branch
	// --filter=blob:none: Skip fetching blobs until needed (partial clone)
	cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", "--filter=blob:none", gitURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// cloneRepositoryWithSubdir Clone specified subdirectory of repository using sparse-checkout
func cloneRepositoryWithSubdir(gitURL, destPath, subdir string) error {
	// Check if directory already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination directory already exists: %s", destPath)
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Convert Windows path separators to Unix-style for Git
	gitSubdir := strings.ReplaceAll(subdir, "\\", "/")

	// Create temporary directory for cloning
	tempDir := destPath + ".tmp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository in temp directory
	initCmd := exec.Command("git", "init", tempDir)
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Add remote repository
	remoteCmd := exec.Command("git", "-C", tempDir, "remote", "add", "origin", gitURL)
	remoteCmd.Stdout = os.Stdout
	remoteCmd.Stderr = os.Stderr
	if err := remoteCmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Enable sparse-checkout BEFORE fetch
	sparseCmd := exec.Command("git", "-C", tempDir, "config", "core.sparseCheckout", "true")
	sparseCmd.Stdout = os.Stdout
	sparseCmd.Stderr = os.Stderr
	if err := sparseCmd.Run(); err != nil {
		return fmt.Errorf("failed to enable sparse-checkout: %w", err)
	}

	// Set directory to checkout BEFORE fetch
	infoDir := filepath.Join(tempDir, ".git", "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .git/info directory: %w", err)
	}

	sparseFile := filepath.Join(infoDir, "sparse-checkout")
	sparseContent := gitSubdir + "\n"
	if err := os.WriteFile(sparseFile, []byte(sparseContent), 0644); err != nil {
		return fmt.Errorf("failed to write sparse-checkout file: %w", err)
	}

	// Fetch remote data
	// Use --filter=tree:0 to only fetch the commit and tree, not all blobs
	fetchCmd := exec.Command("git", "-C", tempDir, "fetch", "--depth=1", "--filter=tree:0", "origin")
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch remote data: %w", err)
	}

	// Read the tree to get directory listing
	readTreeCmd := exec.Command("git", "-C", tempDir, "read-tree", "-mu", "origin/HEAD")
	readTreeCmd.Stdout = os.Stdout
	readTreeCmd.Stderr = os.Stderr
	if err := readTreeCmd.Run(); err != nil {
		return fmt.Errorf("failed to read tree: %w", err)
	}

	// Checkout the files
	checkoutCmd := exec.Command("git", "-C", tempDir, "checkout-index", "-a")
	checkoutCmd.Stdout = os.Stdout
	checkoutCmd.Stderr = os.Stderr
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout files: %w", err)
	}

	// Remove .git directory (we don't need git history for installed skills)
	gitDir := filepath.Join(tempDir, ".git")
	os.RemoveAll(gitDir)

	// Move contents from subdir to root
	// After sparse-checkout, the structure is: tempDir/skills/xxx/...
	// We want: tempDir/...
	subdirPath := filepath.Join(tempDir, filepath.FromSlash(gitSubdir))
	if _, err := os.Stat(subdirPath); err == nil {
		// List all files/dirs in subdir
		entries, err := os.ReadDir(subdirPath)
		if err != nil {
			return fmt.Errorf("failed to read subdir: %w", err)
		}

		// Move each entry to tempDir root
		for _, entry := range entries {
			src := filepath.Join(subdirPath, entry.Name())
			dst := filepath.Join(tempDir, entry.Name())
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
			}
		}

		// Remove empty parent directories
		// Remove the subdir and its parent directories until tempDir
		parts := strings.Split(gitSubdir, "/")
		currentPath := tempDir
		for i := len(parts) - 1; i >= 0; i-- {
			currentPath = filepath.Join(tempDir, filepath.Join(parts[:i+1]...))
			if _, err := os.Stat(currentPath); err == nil {
				_ = os.Remove(currentPath)
			}
		}
	}

	// Rename temp directory to destination
	if err := os.Rename(tempDir, destPath); err != nil {
		return fmt.Errorf("failed to move to destination: %w", err)
	}

	return nil
}

// parseSkillMetadata Parse metadata from skill directory
func parseSkillMetadata(skillPath, skillName string) (storage.SkillMeta, error) {
	var meta storage.SkillMeta
	meta.Name = skillName
	meta.InstalledAt = "" // Set by caller

	// Try to read SKILL.md first
	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	if data, err := os.ReadFile(skillMdPath); err == nil {
		if err := parseSkillMetaFromMarkdown(&meta, string(data)); err == nil {
			return meta, nil
		}
	}

	// Try skill.yaml
	skillYamlPath := filepath.Join(skillPath, "skill.yaml")
	if data, err := os.ReadFile(skillYamlPath); err == nil {
		if err := parseSkillFromYaml(&meta, string(data)); err == nil {
			return meta, nil
		}
	}

	// Try to get information from Git remote
	if gitRemote, err := getGitRemote(skillPath); err == nil {
		if readme, err := fetchGitHubReadme(gitRemote); err == nil {
			meta.Description = extractDescriptionFromReadme(readme)
		}
	}

	return meta, nil
}

// parseSkillMetaFromMarkdown Parse metadata from SKILL.md
func parseSkillMetaFromMarkdown(meta *storage.SkillMeta, markdown string) error {
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
				meta.Name = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:"))
				meta.Name = strings.Trim(meta.Name, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "description:") {
				meta.Description = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:"))
				meta.Description = strings.Trim(meta.Description, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "version:") {
				meta.Version = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:"))
				meta.Version = strings.Trim(meta.Version, `"'`)
			}

			if strings.HasPrefix(trimmedLine, "tags:") {
				tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
				if tagsStr != "" && tagsStr != "[]" {
					tagsStr = strings.Trim(tagsStr, "[]")
					tags := strings.Split(tagsStr, ",")
					for i, tag := range tags {
						tags[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
					}
					meta.Tags = tags
				}
			}

			if strings.HasPrefix(trimmedLine, "author:") {
				meta.Author = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:"))
				meta.Author = strings.Trim(meta.Author, `"'`)
			}
		}
	}

	return nil
}

// parseSkillFromYaml Parse metadata from skill.yaml
func parseSkillFromYaml(meta *storage.SkillMeta, yamlContent string) error {
	lines := strings.Split(yamlContent, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "name:") {
			meta.Name = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:"))
			meta.Name = strings.Trim(meta.Name, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "description:") {
			meta.Description = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:"))
			meta.Description = strings.Trim(meta.Description, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "version:") {
			meta.Version = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:"))
			meta.Version = strings.Trim(meta.Version, `"'`)
		}

		if strings.HasPrefix(trimmedLine, "tags:") {
			tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
			if tagsStr != "" && tagsStr != "[]" {
				tagsStr = strings.Trim(tagsStr, "[]")
				tags := strings.Split(tagsStr, ",")
				for i, tag := range tags {
					tags[i] = strings.TrimSpace(strings.Trim(tag, `"'`))
				}
				meta.Tags = tags
			}
		}

		if strings.HasPrefix(trimmedLine, "author:") {
			meta.Author = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:"))
			meta.Author = strings.Trim(meta.Author, `"'`)
		}
	}

	return nil
}

// getGitRemote Get Git remote repository URL
func getGitRemote(skillPath string) (string, error) {
	cmd := exec.Command("git", "-C", skillPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// fetchGitHubReadme Fetch README from GitHub
func fetchGitHubReadme(gitURL string) (string, error) {
	if !strings.Contains(gitURL, "github.com") {
		return "", fmt.Errorf("not a GitHub repository")
	}

	repoURL := strings.TrimSuffix(gitURL, ".git")
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub URL")
	}

	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	readmeURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/README.md", owner, repo)
	resp, err := http.Get(readmeURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		readmeURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/README.md", owner, repo)
		resp, err = http.Get(readmeURL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("README not found")
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// extractDescriptionFromReadme Extract description from README
func extractDescriptionFromReadme(readme string) string {
	lines := strings.Split(readme, "\n")

	startIdx := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			startIdx = i + 1
		} else {
			break
		}
	}

	var description []string
	for i := startIdx; i < len(lines) && i < startIdx+5; i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "!") {
			description = append(description, line)
		}
	}

	if len(description) > 0 {
		desc := strings.Join(description, " ")
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		return desc
	}

	return ""
}

// copyFile Copy file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
