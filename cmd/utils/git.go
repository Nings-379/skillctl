package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func IsGitURL(source string) bool {
	if strings.Contains(source, "://") {
		if strings.HasPrefix(source, "git://") ||
			strings.HasPrefix(source, "http://") ||
			strings.HasPrefix(source, "https://") ||
			strings.HasPrefix(source, "ssh://") {
			return true
		}
	}

	return strings.Contains(source, "git@") && strings.Contains(source, ":")
}

func ExtractRepoName(gitURL string) (string, error) {
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

func CloneRepository(gitURL, destPath string) error {
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination directory already exists: %s", destPath)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	configCmds := [][]string{
		{"git", "config", "--global", "http.lowSpeedLimit", "0"},
		{"git", "config", "--global", "http.postBuffer", "524288000"},
	}
	for _, cfg := range configCmds {
		_ = exec.Command(cfg[0], cfg[1:]...).Run()
	}

	cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", "--filter=blob:none", gitURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CloneRepositoryWithSubdir(gitURL, destPath, subdir string) error {
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination directory already exists: %s", destPath)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	gitSubdir := strings.ReplaceAll(subdir, "\\", "/")
	tempDir := destPath + ".tmp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := exec.Command("git", "init", tempDir).Run(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "remote", "add", "origin", gitURL).Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "config", "core.sparseCheckout", "true").Run(); err != nil {
		return fmt.Errorf("failed to enable sparse-checkout: %w", err)
	}

	infoDir := filepath.Join(tempDir, ".git", "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .git/info directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(infoDir, "sparse-checkout"), []byte(gitSubdir+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write sparse-checkout file: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "fetch", "--depth=1", "--filter=tree:0", "origin").Run(); err != nil {
		return fmt.Errorf("failed to fetch remote data: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "read-tree", "-mu", "origin/HEAD").Run(); err != nil {
		return fmt.Errorf("failed to read tree: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "checkout-index", "-a").Run(); err != nil {
		return fmt.Errorf("failed to checkout files: %w", err)
	}

	_ = os.RemoveAll(filepath.Join(tempDir, ".git"))
	subdirPath := filepath.Join(tempDir, filepath.FromSlash(gitSubdir))
	if _, err := os.Stat(subdirPath); err == nil {
		entries, err := os.ReadDir(subdirPath)
		if err != nil {
			return fmt.Errorf("failed to read subdir: %w", err)
		}
		for _, entry := range entries {
			src := filepath.Join(subdirPath, entry.Name())
			dst := filepath.Join(tempDir, entry.Name())
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
			}
		}

		parts := strings.Split(gitSubdir, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			currentPath := filepath.Join(tempDir, filepath.Join(parts[:i+1]...))
			if _, err := os.Stat(currentPath); err == nil {
				_ = os.Remove(currentPath)
			}
		}
	}

	if err := os.Rename(tempDir, destPath); err != nil {
		return fmt.Errorf("failed to move to destination: %w", err)
	}

	return nil
}

func CloneRepositoryAtTag(gitURL, destPath, tag string) error {
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination directory already exists: %s", destPath)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth=1", "--branch", tag, "--single-branch", gitURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CloneRepositoryWithSubdirAtTag(gitURL, destPath, subdir, tag string) error {
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination directory already exists: %s", destPath)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	gitSubdir := strings.ReplaceAll(subdir, "\\", "/")
	tempDir := destPath + ".tmp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := exec.Command("git", "init", tempDir).Run(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "remote", "add", "origin", gitURL).Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "config", "core.sparseCheckout", "true").Run(); err != nil {
		return fmt.Errorf("failed to enable sparse-checkout: %w", err)
	}

	infoDir := filepath.Join(tempDir, ".git", "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .git/info directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(infoDir, "sparse-checkout"), []byte(gitSubdir+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write sparse-checkout file: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "fetch", "--depth=1", "--filter=tree:0", "origin", "refs/tags/"+tag).Run(); err != nil {
		return fmt.Errorf("failed to fetch tag '%s': %w", tag, err)
	}
	if err := exec.Command("git", "-C", tempDir, "read-tree", "-mu", "FETCH_HEAD").Run(); err != nil {
		return fmt.Errorf("failed to read tree: %w", err)
	}
	if err := exec.Command("git", "-C", tempDir, "checkout-index", "-a").Run(); err != nil {
		return fmt.Errorf("failed to checkout files: %w", err)
	}

	_ = os.RemoveAll(filepath.Join(tempDir, ".git"))
	subdirPath := filepath.Join(tempDir, filepath.FromSlash(gitSubdir))
	if _, err := os.Stat(subdirPath); err == nil {
		entries, err := os.ReadDir(subdirPath)
		if err != nil {
			return fmt.Errorf("failed to read subdir: %w", err)
		}
		for _, entry := range entries {
			src := filepath.Join(subdirPath, entry.Name())
			dst := filepath.Join(tempDir, entry.Name())
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("failed to move %s: %w", entry.Name(), err)
			}
		}

		parts := strings.Split(gitSubdir, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			currentPath := filepath.Join(tempDir, filepath.Join(parts[:i+1]...))
			if _, err := os.Stat(currentPath); err == nil {
				_ = os.Remove(currentPath)
			}
		}
	}

	if err := os.Rename(tempDir, destPath); err != nil {
		return fmt.Errorf("failed to move to destination: %w", err)
	}

	return nil
}

func ListRemoteTags(gitURL, prefix string) ([]string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git is not installed or not in PATH")
	}

	cmd := exec.Command("git", "ls-remote", "--tags", "--refs", gitURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list remote tags: %w", err)
	}

	var tags []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		tag := strings.TrimPrefix(ref, "refs/tags/")
		if prefix == "" || strings.HasPrefix(tag, prefix) {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func GetGitRemote(skillPath string) (string, error) {
	cmd := exec.Command("git", "-C", skillPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func FetchGitHubReadme(gitURL string) (string, error) {
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
	for _, readmeURL := range []string{
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/README.md", owner, repo),
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/README.md", owner, repo),
	} {
		resp, err := http.Get(readmeURL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	return "", fmt.Errorf("README not found")
}

func ExtractDescriptionFromReadme(readme string) string {
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

	if len(description) == 0 {
		return ""
	}

	desc := strings.Join(description, " ")
	if len(desc) > 200 {
		return desc[:200] + "..."
	}

	return desc
}
