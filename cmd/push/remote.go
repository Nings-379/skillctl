package pushcmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cmdutils "github.com/seekthought/skill/cmd/utils"
)

var gitCommandRunner = runGitCommandOutputDirect

func validateSkillPackage(path string) error {
	skillMd := filepath.Join(path, "SKILL.md")
	skillYaml := filepath.Join(path, "skill.yaml")
	if _, err := os.Stat(skillMd); os.IsNotExist(err) {
		if _, err := os.Stat(skillYaml); os.IsNotExist(err) {
			return fmt.Errorf("neither SKILL.md nor skill.yaml found in skill package")
		}
	}
	return nil
}

func pushRemote(skillName, skillPath, repoURL string) error {
	host, projectPath, scheme, err := parseRemoteRepoURL(repoURL)
	if err != nil {
		return err
	}
	provider, err := detectRemoteProvider(scheme, host)
	if err != nil {
		return err
	}

	var defaultBranch, webRepoURL, gitlabAPIBase, token, pushBranch, mergeRequestURL string

	switch provider {
	case "gitlab":
		gitlabAPIBase = fmt.Sprintf("%s://%s/api/v4", scheme, host)
		webRepoURL = buildWebRepoURL(repoURL, host, projectPath)
		token = os.Getenv("GITLAB_TOKEN")
		fmt.Printf("📤 Remote repository: %s\n", repoURL)
		fmt.Printf("📤 Remote provider: GitLab\n")
		fmt.Printf("📤 GitLab API: %s\n", gitlabAPIBase)
		fmt.Println()
		if token == "" {
			fmt.Println("⚠️  GITLAB_TOKEN not set. Branch push will use local git credentials, but merge request creation will be manual.")
			fmt.Println("💡 Tip: Set GITLAB_TOKEN if you want skillctl to create the merge request automatically.")
		}
	case "github":
		webRepoURL = buildWebRepoURL(repoURL, host, projectPath)
		fmt.Printf("📤 Remote repository: %s\n", repoURL)
		fmt.Printf("📤 Remote provider: GitHub\n")
		fmt.Printf("📤 Remote transport: local git\n")
		fmt.Println()
	default:
		return fmt.Errorf("unsupported remote provider for URL: %s (currently supports GitHub and GitLab)", repoURL)
	}

	defaultBranch, err = detectDefaultBranchFromGit(repoURL)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not detect default branch from git, using 'main': %v\n", err)
		defaultBranch = "main"
	}

	fmt.Printf("📂 Default branch: %s\n", defaultBranch)
	fmt.Println()
	pushBranch = defaultBranch
	if provider == "gitlab" {
		pushBranch = fmt.Sprintf("skill/%s-%s", sanitizeBranchName(skillName), timeNow())
	}

	if err := pushRemoteWithGit(skillName, skillPath, repoURL, defaultBranch, pushBranch); err != nil {
		return err
	}

	fmt.Printf("Remote repository: %s\n", repoURL)
	if provider == "gitlab" {
		fmt.Println()
		if token != "" {
			fmt.Printf("🔀 Creating merge request: %s -> %s\n", pushBranch, defaultBranch)
			mrTitle := fmt.Sprintf("Add/Update skill: %s", skillName)
			mrDescription := fmt.Sprintf("Automated update from skillctl push for skill '%s'.", skillName)
			mergeRequestURL, err = createGitLabMergeRequest(gitlabAPIBase, projectPath, pushBranch, defaultBranch, mrTitle, mrDescription, token)
			if err != nil {
				return fmt.Errorf("branch pushed but failed to create merge request: %w", err)
			}
			fmt.Println("✓ Merge request created")
			fmt.Printf("MR URL: %s\n", mergeRequestURL)
		} else {
			fmt.Printf("🌿 Branch pushed: %s\n", pushBranch)
			fmt.Printf("💡 Create the merge request manually: %s/-/merge_requests/new?merge_request[source_branch]=%s&merge_request[target_branch]=%s\n", webRepoURL, url.QueryEscape(pushBranch), url.QueryEscape(defaultBranch))
		}
		fmt.Printf("Branch view: %s/-/tree/%s/skills/%s\n", webRepoURL, pushBranch, skillName)
	} else {
		fmt.Printf("View at: %s/tree/%s/skills/%s\n", webRepoURL, defaultBranch, skillName)
	}

	return nil
}

func pushRemoteWithGit(skillName, skillPath, repoURL, defaultBranch, pushBranch string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	workDir, err := os.MkdirTemp("", "skillctl-push-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	cloneDir := filepath.Join(workDir, "repo")
	fmt.Printf("📥 Cloning remote branch '%s'...\n", defaultBranch)
	if err := runGitCommand("", "clone", "--depth=1", "--branch", defaultBranch, repoURL, cloneDir); err != nil {
		return fmt.Errorf("failed to clone remote repository: %w", err)
	}
	fmt.Println("✓ Repository cloned")

	if pushBranch != defaultBranch {
		fmt.Printf("🌿 Creating local branch: %s\n", pushBranch)
		if err := runGitCommand(cloneDir, "checkout", "-b", pushBranch); err != nil {
			return fmt.Errorf("failed to create local branch: %w", err)
		}
		fmt.Println("✓ Branch created")
	}

	targetSkillDir := filepath.Join(cloneDir, "skills", skillName)
	if err := os.RemoveAll(targetSkillDir); err != nil {
		return fmt.Errorf("failed to remove existing remote skill directory: %w", err)
	}
	if err := copySkillDirectory(skillPath, targetSkillDir); err != nil {
		return err
	}

	if err := runGitCommand(cloneDir, "add", filepath.ToSlash(filepath.Join("skills", skillName))); err != nil {
		return fmt.Errorf("failed to stage skill files: %w", err)
	}

	hasChanges, err := gitHasChanges(cloneDir)
	if err != nil {
		return err
	}

	skillMeta, _ := cmdutils.ParseSkillMetadata(skillPath, skillName)

	if hasChanges {
		commitMessage := fmt.Sprintf("Add/Update skill: %s", skillName)
		fmt.Printf("📝 Creating commit: %s\n", commitMessage)
		if err := runGitCommand(cloneDir, "commit", "-m", commitMessage); err != nil {
			return fmt.Errorf("failed to create commit: %w", err)
		}
		fmt.Println("✓ Commit created")

		fmt.Printf("🚀 Pushing branch '%s'...\n", pushBranch)
		if err := runGitCommand(cloneDir, "push", "-u", "origin", pushBranch); err != nil {
			return fmt.Errorf("failed to push branch: %w", err)
		}
		fmt.Println("✓ Branch pushed")
	} else {
		fmt.Println("ℹ️  No file changes detected; skipping commit.")
	}

	if skillMeta.Version != "" {
		tagName := fmt.Sprintf("%s@%s", skillName, skillMeta.Version)
		fmt.Printf("🏷️  Creating version tag: %s\n", tagName)
		_ = runGitCommand(cloneDir, "tag", "-d", tagName)
		if err := runGitCommand(cloneDir, "tag", "-a", tagName, "-m", fmt.Sprintf("Release %s version %s", skillName, skillMeta.Version)); err != nil {
			fmt.Printf("⚠️  Warning: Could not create tag: %v\n", err)
		} else {
			fmt.Println("✓ Tag created")
			fmt.Printf("🚀 Pushing tag '%s'...\n", tagName)
			if err := runGitCommand(cloneDir, "push", "origin", tagName, "--force"); err != nil {
				fmt.Printf("⚠️  Warning: Could not push tag: %v\n", err)
			} else {
				fmt.Println("✓ Tag pushed")
			}
		}
	}

	return nil
}

func copySkillDirectory(sourcePath, targetPath string) error {
	fileCount := 0
	err := filepath.Walk(sourcePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourcePath, filePath)
		if err != nil {
			return err
		}
		destination := filepath.Join(targetPath, relPath)
		if info.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destination, content, info.Mode()); err != nil {
			return err
		}
		fileCount++
		fmt.Printf("  ✓ %s\n", filepath.ToSlash(filepath.Join("skills", skillNameFromTarget(targetPath), relPath)))
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to copy files into git worktree: %w", err)
	}
	fmt.Println()
	fmt.Printf("✓ Prepared %d file(s) for push\n", fileCount)
	return nil
}

func skillNameFromTarget(targetPath string) string {
	return filepath.Base(targetPath)
}

func gitHasChanges(repoDir string) (bool, error) {
	output, err := runGitCommandOutput(repoDir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func detectDefaultBranchFromGit(repoURL string) (string, error) {
	output, err := runGitCommandOutput("", "ls-remote", "--symref", repoURL, "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "ref: refs/heads/") {
			branch := strings.TrimPrefix(line, "ref: refs/heads/")
			branch = strings.SplitN(branch, "\t", 2)[0]
			if branch != "" {
				return branch, nil
			}
		}
	}
	return "", fmt.Errorf("could not parse default branch from git ls-remote output")
}

func runGitCommand(repoDir string, args ...string) error {
	_, err := runGitCommandOutput(repoDir, args...)
	return err
}

func runGitCommandOutput(repoDir string, args ...string) (string, error) {
	return gitCommandRunner(repoDir, args...)
}

func runGitCommandOutputDirect(repoDir string, args ...string) (string, error) {
	cmdArgs := args
	if repoDir != "" {
		cmdArgs = append([]string{"-C", repoDir}, args...)
	}
	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, trimmed)
	}
	return string(output), nil
}

func timeNow() string {
	return time.Now().Format("20060102150405")
}

func detectDefaultBranchGitHub(githubAPIBase, token string) (string, error) {
	return detectDefaultBranch(githubAPIBase, token, "Authorization", func(t string) string { return fmt.Sprintf("token %s", t) })
}

func detectDefaultBranchGitLab(gitlabAPIBase, projectPath, token string) (string, error) {
	projectID := url.PathEscape(projectPath)
	repoURL := fmt.Sprintf("%s/projects/%s", gitlabAPIBase, projectID)
	return detectDefaultBranch(repoURL, token, "PRIVATE-TOKEN", func(t string) string { return t })
}

func detectDefaultBranch(urlValue, token, authHeader string, authValue func(string) string) (string, error) {
	var resp *http.Response
	var err error
	if token != "" {
		req, _ := http.NewRequest("GET", urlValue, nil)
		req.Header.Set(authHeader, authValue(token))
		resp, err = http.DefaultClient.Do(req)
	} else {
		resp, err = http.Get(urlValue)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get repository info (HTTP %d): %s", resp.StatusCode, string(body))
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

func uploadFileGitHub(githubAPIBase, path string, content []byte, token string, branch string) error {
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

	encodedContent := base64.StdEncoding.EncodeToString(content)
	uploadData := map[string]interface{}{"message": fmt.Sprintf("Update %s", path), "content": encodedContent, "branch": branch}
	if sha != "" {
		uploadData["sha"] = sha
	}
	jsonData, err := json.Marshal(uploadData)
	if err != nil {
		return err
	}

	putURL := fmt.Sprintf("%s/contents/%s", githubAPIBase, path)
	req, err := http.NewRequest("PUT", putURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	resp, err = (&http.Client{}).Do(req)
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

func uploadFileGitLab(gitlabAPIBase, projectPath, path string, content []byte, token string, branch string) error {
	projectID := url.PathEscape(projectPath)
	encodedPath := url.PathEscape(path)
	fileURL := fmt.Sprintf("%s/projects/%s/repository/files/%s", gitlabAPIBase, projectID, encodedPath)
	uploadData := map[string]interface{}{"branch": branch, "content": base64.StdEncoding.EncodeToString(content), "commit_message": fmt.Sprintf("Update %s", path), "encoding": "base64"}
	jsonData, err := json.Marshal(uploadData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fileURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(string(body)), "does not exist") {
		createReq, err := http.NewRequest("POST", fileURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		createReq.Header.Set("Content-Type", "application/json")
		if token != "" {
			createReq.Header.Set("PRIVATE-TOKEN", token)
		}

		createResp, err := client.Do(createReq)
		if err != nil {
			return err
		}
		defer createResp.Body.Close()
		if createResp.StatusCode != http.StatusCreated {
			createBody, _ := io.ReadAll(createResp.Body)
			return fmt.Errorf("failed to create file (HTTP %d): %s", createResp.StatusCode, string(createBody))
		}
		return nil
	}

	return fmt.Errorf("failed to upload file (HTTP %d): %s", resp.StatusCode, string(body))
}

func createGitLabBranch(gitlabAPIBase, projectPath, branchName, ref, token string) error {
	projectID := url.PathEscape(projectPath)
	branchURL := fmt.Sprintf("%s/projects/%s/repository/branches", gitlabAPIBase, projectID)
	jsonData, err := json.Marshal(map[string]string{"branch": branchName, "ref": ref})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", branchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create branch (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func createGitLabMergeRequest(gitlabAPIBase, projectPath, sourceBranch, targetBranch, title, description, token string) (string, error) {
	projectID := url.PathEscape(projectPath)
	mrURL := fmt.Sprintf("%s/projects/%s/merge_requests", gitlabAPIBase, projectID)
	jsonData, err := json.Marshal(map[string]interface{}{"source_branch": sourceBranch, "target_branch": targetBranch, "title": title, "description": description, "remove_source_branch": true})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", mrURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create merge request (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		WebURL string `json:"web_url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.WebURL == "" {
		return "", fmt.Errorf("merge request created but web_url not found in response")
	}

	return result.WebURL, nil
}

func sanitizeBranchName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-", ":", "-", "@", "-")
	result := replacer.Replace(lower)
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	result = strings.Trim(result, "-")
	if result == "" {
		return "skill"
	}
	return result
}

func detectRemoteProvider(scheme, host string) (string, error) {
	h := strings.ToLower(host)
	if strings.Contains(h, "gitlab") {
		return "gitlab", nil
	}
	if strings.Contains(h, "github") {
		return "github", nil
	}

	provider, err := detectRemoteProviderByAPI(scheme, host)
	if err == nil {
		return provider, nil
	}

	return "", fmt.Errorf("could not determine remote provider for host %s: %w", host, err)
}

var remoteProviderProbe = probeRemoteProvider

func detectRemoteProviderByAPI(scheme, host string) (string, error) {
	return remoteProviderProbe(scheme, host)
}

func probeRemoteProvider(scheme, host string) (string, error) {
	gitlabURL := fmt.Sprintf("%s://%s/api/v4/version", scheme, host)
	if ok, err := isGitLabEndpoint(gitlabURL); ok {
		return "gitlab", nil
	} else if err == nil {
		githubURL := fmt.Sprintf("%s://%s/api/v3", scheme, host)
		if ok, githubErr := isGitHubEndpoint(githubURL); ok {
			return "github", nil
		} else if githubErr != nil {
			return "", githubErr
		}
	} else {
		githubURL := fmt.Sprintf("%s://%s/api/v3", scheme, host)
		if ok, githubErr := isGitHubEndpoint(githubURL); ok {
			return "github", nil
		} else if githubErr != nil {
			return "", fmt.Errorf("gitlab probe failed: %v; github probe failed: %w", err, githubErr)
		}
	}

	return "", fmt.Errorf("host %s did not match GitLab v4 or GitHub v3 API signatures", host)
}

func isGitLabEndpoint(endpoint string) (bool, error) {
	body, statusCode, err := fetchProviderProbe(endpoint)
	if err != nil {
		return false, err
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return true, nil
	}
	if statusCode != http.StatusOK {
		return false, nil
	}
	text := strings.ToLower(string(body))
	return strings.Contains(text, "version") && strings.Contains(text, "revision"), nil
}

func isGitHubEndpoint(endpoint string) (bool, error) {
	body, statusCode, err := fetchProviderProbe(endpoint)
	if err != nil {
		return false, err
	}
	if statusCode != http.StatusOK {
		return false, nil
	}
	text := strings.ToLower(string(body))
	return strings.Contains(text, "current_user_url") || strings.Contains(text, "repository_url"), nil
}

func fetchProviderProbe(endpoint string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "skillctl")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func parseRemoteRepoURL(repoURL string) (host string, projectPath string, scheme string, err error) {
	scheme = "https"
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		parsed, parseErr := url.Parse(repoURL)
		if parseErr != nil {
			return "", "", "", fmt.Errorf("invalid remote URL: %w", parseErr)
		}
		host = parsed.Host
		scheme = parsed.Scheme
		projectPath = strings.Trim(strings.TrimSuffix(parsed.Path, ".git"), "/")
		if host == "" || projectPath == "" {
			return "", "", "", fmt.Errorf("invalid remote URL: %s", repoURL)
		}
		return host, projectPath, scheme, nil
	}
	if strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") {
		parts := strings.SplitN(repoURL, ":", 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid remote URL: %s", repoURL)
		}
		atParts := strings.SplitN(parts[0], "@", 2)
		if len(atParts) != 2 {
			return "", "", "", fmt.Errorf("invalid remote URL: %s", repoURL)
		}
		host = atParts[1]
		projectPath = strings.Trim(strings.TrimSuffix(parts[1], ".git"), "/")
		if host == "" || projectPath == "" {
			return "", "", "", fmt.Errorf("invalid remote URL: %s", repoURL)
		}
		return host, projectPath, scheme, nil
	}
	return "", "", "", fmt.Errorf("invalid remote URL format: %s", repoURL)
}

func buildWebRepoURL(repoURL, host, projectPath string) string {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		return strings.TrimSuffix(repoURL, ".git")
	}
	return fmt.Sprintf("https://%s/%s", host, projectPath)
}
