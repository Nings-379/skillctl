package listcmd

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

	cmdutils "github.com/seekthought/skill/cmd/utils"
)

func listRemote(_ *options, remoteURL string) error {
	printSectionHeader("Remote Skills")

	if remoteURL == "" {
		return fmt.Errorf("no URL provided. Please provide a URL: skillctl list -R <url>")
	}

	fmt.Printf("📦 URL: %s\n", remoteURL)
	fmt.Println()

	if strings.Contains(remoteURL, "github.com") {
		return listGitRepo(remoteURL)
	}
	if strings.HasPrefix(remoteURL, "http://") || strings.HasPrefix(remoteURL, "https://") {
		return listGitRepo(remoteURL)
	}
	return listAPI(remoteURL)
}

func listAPI(url string) error {
	fmt.Printf("Fetching from: %s\n\n", url)

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

	var skillsData struct {
		Skills []struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			RepoURL     string   `json:"repoUrl"`
			RepoOwner   string   `json:"repoOwner"`
			Topics      []string `json:"topics"`
			Category    struct {
				Name string `json:"name"`
			} `json:"category"`
		} `json:"skills"`
	}

	if err := json.Unmarshal(body, &skillsData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Debug: API response size: %d bytes, skills count: %d\n", len(body), len(skillsData.Skills))

	var skills []cmdutils.CustomSkill
	for _, item := range skillsData.Skills {
		skill := cmdutils.CustomSkill{
			Name:        item.Name,
			Description: item.Description,
			Tags:        item.Topics,
			Author:      item.RepoOwner,
			Path:        item.RepoURL,
		}
		if item.Category.Name != "" {
			skill.Tags = append([]string{item.Category.Name}, skill.Tags...)
		}
		skills = append(skills, skill)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in remote repository.")
		return nil
	}

	renderCustomSkills(skills, customSkillRenderOptions{PathLabel: "URL", ShowPath: true, ShowAuthor: true, ShowVersion: true, ShowTags: true})

	fmt.Println()
	fmt.Printf("Total: %d skill(s) in remote repository\n", len(skills))
	fmt.Println()
	printTips("Install a skill using: skillctl install <git-url>", "List local skills using: skillctl list")

	return nil
}

func listGitRepo(gitURL string) error {
	fmt.Printf("Repository: %s\n", gitURL)
	fmt.Println()

	if strings.Contains(gitURL, "github.com") {
		skills, err := fetchSkillsFromGitHubAPI(gitURL)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch via GitHub API: %v\n", err)
			fmt.Println("Falling back to git clone...")
			fmt.Println()
			return listGitRepoViaClone(gitURL)
		}

		if len(skills) == 0 {
			fmt.Println("No skills found in the repository.")
			return nil
		}

		renderCustomSkills(skills, customSkillRenderOptions{ShowTags: true})
		fmt.Println()
		fmt.Printf("Total: %d skill(s) found in repository\n", len(skills))
		fmt.Println()
		fmt.Printf("To install a specific skill, use: skillctl install %s <skill-name>\n", gitURL)
		return nil
	}

	return listGitRepoViaClone(gitURL)
}

func fetchSkillsFromGitHubAPI(gitURL string) ([]cmdutils.CustomSkill, error) {
	parts := strings.Split(gitURL, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL")
	}
	owner := parts[len(parts)-2]
	repo := strings.TrimSuffix(parts[len(parts)-1], ".git")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills", owner, repo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

	var items []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	var skills []cmdutils.CustomSkill
	for _, item := range items {
		if item.Type != "dir" {
			continue
		}

		skill := cmdutils.CustomSkill{Name: item.Name, Path: fmt.Sprintf("%s/%s", gitURL, item.Name)}
		description, tags, err := fetchSkillDescriptionFromGitHubAPI(owner, repo, item.Name)
		if err == nil {
			skill.Description = description
			skill.Tags = tags
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func fetchSkillDescriptionFromGitHubAPI(owner, repo, skillName string) (string, []string, error) {
	urls := []string{
		fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills/%s/SKILL.md", owner, repo, skillName),
		fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills/%s/skill.yaml", owner, repo, skillName),
		fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/skills/%s/README.md", owner, repo, skillName),
	}

	for _, url := range urls {
		description, tags, err := fetchGitHubFileContent(url)
		if err == nil && description != "" {
			return description, tags, nil
		}
	}

	return "", nil, fmt.Errorf("no description file found")
}

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

	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(body, &fileContent); err != nil {
		return "", nil, err
	}

	if fileContent.Encoding != "base64" {
		return "", nil, fmt.Errorf("unsupported encoding")
	}

	decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
	if err != nil {
		return "", nil, err
	}

	return cmdutils.ParseSkillContentMetadata(string(decoded))
}

func listGitRepoViaClone(gitURL string) error {
	tempDir, err := os.MkdirTemp("", "skill-list-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("📥 Fetching repository information...")
	cmd := exec.Command("git", "clone", "--depth=1", "--quiet", gitURL, tempDir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	fmt.Println("✓ Repository fetched successfully")
	fmt.Println()

	var skillsDir string
	for _, dir := range []string{filepath.Join(tempDir, "skills"), filepath.Join(tempDir, ".skills")} {
		if _, err := os.Stat(dir); err == nil {
			skillsDir = dir
			break
		}
	}

	if skillsDir == "" {
		fmt.Println("No 'skills' or '.skills' directory found in the repository.")
		fmt.Println()
		printTips("Tip: The repository should contain a 'skills' or '.skills' directory with skill subdirectories.")
		return nil
	}

	skills, err := cmdutils.ScanCustomSkills(skillsDir)
	if err != nil {
		return fmt.Errorf("failed to scan skills directory: %w", err)
	}
	if len(skills) == 0 {
		fmt.Println("No skills found in the skills directory.")
		return nil
	}

	renderCustomSkills(skills, customSkillRenderOptions{ShowTags: true})
	fmt.Println()
	fmt.Printf("Total: %d skill(s) found in repository\n", len(skills))
	fmt.Println()
	fmt.Printf("To install a specific skill, use: skillctl install %s <skill-name>\n", gitURL)

	return nil
}

var _ = context.Background
