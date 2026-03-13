package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	targetLocation string // Target location (github, agents, claude, copilot, or custom path)
	fromDir        string // Directory to search for skills
)

var addCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Add a skill from Git repository, local path, or installed skill name",
	Long:  `Download and install a skill from a Git repository, copy from a local path, or add from an installed skill. Can install to ~/.skills or directly to VS Code recognized skills directories for immediate use.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		return doAdd(source)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&targetLocation, "project", "p", "", "Target location: github, agents, claude, copilot, or custom path")
	addCmd.Flags().StringVarP(&projectDir, "dir", "d", "", "Project directory (default: current directory)")
	addCmd.Flags().StringVarP(&agentSkillsDir, "skills-dir", "s", "examples/skills", "Agent skills directory relative to project")

	// Set default fromDir to .skill folder in user's home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		fromDir = filepath.Join(homeDir, ".skill")
	}
	addCmd.Flags().StringVar(&fromDir, "from", fromDir, "Directory to search for skills (default: ~/.skill)")
}

// VSCodeSkillsDirs VS Code recognized skill directories
var VSCodeSkillsDirs = map[string]string{
	"github":  ".github/skills",
	"agents":  ".agents/skills",
	"claude":  ".claude/skills",
	"copilot": ".copilot/skills",
}

func doAdd(source string) error {
	fmt.Printf("📦 Adding skill from: %s\n", source)
	fmt.Println()

	// Determine if it's an installed skill name, Git URL, or local path
	skillName, sourcePath, isInstalledSkill, err := resolveSource(source)
	if err != nil {
		return err
	}

	var repoName string

	if isInstalledSkill {
		// Installed skill
		repoName = skillName
		fmt.Printf("📁 Using installed skill: %s\n", skillName)
		fmt.Printf("   Source path: %s\n", sourcePath)
	} else if sourcePath != "" {
		// Local path
		repoName = filepath.Base(sourcePath)
		fmt.Printf("📁 Using local path: %s\n", sourcePath)
	} else {
		// Git URL
		repoName = skillName
		fmt.Printf("📁 Using Git URL: %s\n", source)
	}

	// Determine installation path
	var installPath string

	if targetLocation != "" {
		// Use specified VS Code recognized directory
		installPath, err = getVSCodeInstallPath(targetLocation, repoName)
		if err != nil {
			return err
		}
		fmt.Printf("📁 Installing to: %s\n", installPath)
	} else {
		// No target location specified, return error
		return fmt.Errorf("please specify a target location using --project/-p")
	}

	// Check if already exists
	if _, err := os.Stat(installPath); err == nil {
		return fmt.Errorf("⚠️ skill already exists at: %s", installPath)
	}

	// Copy or clone skill
	if sourcePath != "" {
		fmt.Println("📥 Copying skill...")
		if err := copyDirectory(sourcePath, installPath); err != nil {
			return fmt.Errorf("failed to copy skill: %w", err)
		}
		fmt.Println("✓ Skill copied successfully")
	} else {
		fmt.Println("📥 Cloning repository...")
		if err := cloneRepository(source, installPath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Println("✓ Repository cloned successfully")
	}

	// Parse skill metadata
	fmt.Println("📋 Parsing skill metadata...")
	skillMeta, err := parseSkillMetadata(installPath, repoName)
	if err != nil {
		fmt.Printf("⚠ Warning: Could not parse skill metadata: %v\n", err)
		sourceDesc := source
		if sourcePath != "" {
			sourceDesc = sourcePath
		}
		skillMeta = storage.SkillMeta{
			Name:        repoName,
			Description: fmt.Sprintf("Skill from %s", sourceDesc),
			InstalledAt: time.Now().Format(time.RFC3339),
		}
	} else {
		fmt.Println("✓ Skill metadata parsed")
	}

	// Update skill index in ~/.skills
	fmt.Println("📋 Updating skill index...")
	if err := storage.AddSkill(repoName, skillMeta); err != nil {
		fmt.Printf("⚠ Warning: Could not update skill index: %v\n", err)
	} else {
		fmt.Println("✓ Skill index updated")
	}

	fmt.Println()
	fmt.Printf("✅ Skill '%s' has been successfully installed!\n", repoName)
	if skillMeta.Version != "" {
		fmt.Printf("   Version: %s\n", skillMeta.Version)
	}
	if skillMeta.Author != "" {
		fmt.Printf("   Author: %s\n", skillMeta.Author)
	}
	if skillMeta.Description != "" {
		fmt.Printf("   Description: %s\n", skillMeta.Description)
	}
	fmt.Printf("   Location: %s\n", installPath)

	fmt.Println()
	fmt.Println("💡 The skill is now available for agent use!")
	fmt.Println("   Agent can now access this skill from:", installPath)

	// Refresh VS Code
	if err := refreshVSCode(); err != nil {
		fmt.Printf("⚠ Warning: Could not refresh VS Code: %v\n", err)
	} else {
		fmt.Println("✓ VS Code has been refreshed")
	}

	return nil
}

// getVSCodeInstallPath Get VS Code recognized skill directory path
func getVSCodeInstallPath(location, repoName string) (string, error) {
	var skillsDir string

	// Check if it's a predefined location
	if dir, ok := VSCodeSkillsDirs[location]; ok {
		skillsDir = dir
	} else {
		// Use custom path
		skillsDir = location
	}

	// Determine base directory
	var baseDir string
	if filepath.IsAbs(skillsDir) {
		// If absolute path, use directly
		baseDir = ""
	} else {
		// If relative path, use project directory
		baseDir = projectDir
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current directory: %w", err)
			}
		}
	}

	// Build complete path
	if baseDir != "" {
		return filepath.Join(baseDir, skillsDir, repoName), nil
	}
	return filepath.Join(skillsDir, repoName), nil
}

// refreshVSCode Refresh VS Code workspace
func refreshVSCode() error {
	// Check if running in VS Code
	if os.Getenv("VSCODE_PID") == "" {
		// Not running in VS Code, try to refresh using code command
		// Get current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try to use code command
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "code", ".", "--goto", cwd)
		} else {
			cmd = exec.Command("code", ".", "--goto", cwd)
		}

		// Run in background, don't block
		if err := cmd.Start(); err != nil {
			// code command may not be available, this is normal
			return nil
		}

		return nil
	}

	// Running in VS Code, send refresh command
	// Use VS Code command to refresh workspace
	fmt.Println("   (Running in VS Code - workspace will refresh automatically)")
	return nil
}

// isLocalPath Determine if it's a local path
func isLocalPath(source string) bool {
	// Check if it contains Git URL characteristics
	if strings.Contains(source, "://") {
		// Check if it's a Git protocol
		if strings.HasPrefix(source, "git://") ||
			strings.HasPrefix(source, "http://") ||
			strings.HasPrefix(source, "https://") ||
			strings.HasPrefix(source, "ssh://") {
			return false
		}
	}

	// Check if it's a Git URL format (git@github.com:user/repo.git)
	if strings.Contains(source, "git@") && strings.Contains(source, ":") {
		return false
	}

	// Check if it contains .git suffix (may be a Git URL)
	if strings.HasSuffix(source, ".git") && !filepath.IsAbs(source) {
		return false
	}

	// Check if it contains / or \ (may be a path)
	if strings.Contains(source, "/") || strings.Contains(source, string(filepath.Separator)) {
		// Check if path exists
		if _, err := os.Stat(source); err == nil {
			return true
		}
	}

	// Check if it's a relative path
	if !strings.Contains(source, "://") && !strings.Contains(source, "@") {
		// Try to check as a path
		if _, err := os.Stat(source); err == nil {
			return true
		}
	}

	return false
}

// copyDirectory Recursively copy directory
func copyDirectory(src, dst string) error {
	// Create target directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Traverse source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy files
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveSource Resolve source, return skill name, source path, whether it's an installed skill
func resolveSource(source string) (skillName string, sourcePath string, isInstalledSkill bool, err error) {
	// 0. If --from parameter is specified, search in that directory first
	if fromDir != "" {
		possiblePath := filepath.Join(fromDir, source)
		if _, err := os.Stat(possiblePath); err == nil {
			// Found skill in specified directory
			return source, possiblePath, true, nil
		}
	}

	// 1. First check if it's an installed skill name (in ~/.skills)
	skillMeta, err := storage.GetSkill(source)
	if err == nil && skillMeta != nil {
		// Is an installed skill
		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return "", "", false, fmt.Errorf("failed to get skills directory: %w", err)
		}

		installedPath := filepath.Join(skillsDir, skillMeta.Name)
		if _, err := os.Stat(installedPath); err != nil {
			return "", "", false, fmt.Errorf("skill '%s' is in index but directory not found: %s", source, installedPath)
		}

		return skillMeta.Name, installedPath, true, nil
	}

	// 2. Check if it's a local path
	if isLocalPath(source) {
		path := filepath.Clean(source)
		if !filepath.IsAbs(path) {
			path, err = filepath.Abs(path)
			if err != nil {
				return "", "", false, fmt.Errorf("failed to get absolute path: %w", err)
			}
		}

		// Check if path exists
		if _, err := os.Stat(path); err != nil {
			return "", "", false, fmt.Errorf("source path does not exist: %s", source)
		}

		return filepath.Base(path), path, false, nil
	}

	// 3. Check if it's an explicit Git URL (contains protocol or git@)
	if isGitURL(source) {
		repoName, err := extractRepoName(source)
		if err != nil {
			return "", "", false, fmt.Errorf("failed to extract repository name: %w", err)
		}
		return repoName, "", false, nil
	}

	// 4. Search in ~/.skills directory (even if not in index)
	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return "", "", false, fmt.Errorf("failed to get skills directory: %w", err)
	}

	possiblePath := filepath.Join(skillsDir, source)
	if _, err := os.Stat(possiblePath); err == nil {
		// Found skill in ~/.skills directory
		return source, possiblePath, true, nil
	}

	// 5. Search in common skill directories in project directory
	projectDirs := []string{
		"examples/skills",
		".github/skills",
		".agents/skills",
		".claude/skills",
		".copilot/skills",
		".skill", // Add .skill directory as default search location
	}

	for _, projectDir := range projectDirs {
		possiblePath := filepath.Join(projectDir, source)
		if _, err := os.Stat(possiblePath); err == nil {
			// Found skill in project directory
			return source, possiblePath, true, nil
		}
	}

	// 6. Finally considered an invalid source
	return "", "", false, fmt.Errorf("skill '%s' not found.\n\nSearched in:\n- ~/.skills (installed skills)\n- %s (--from directory)\n- .skill (default skills directory)\n- examples/skills, .github/skills, .agents/skills, .claude/skills, .copilot/skills\n\nPlease:\n- Use 'skillctl list' to see installed skills\n- Use 'skillctl list --skills-dir .skill' to see skills in .skill directory\n- Use 'skillctl add https://github.com/user/skill -l <location>' for Git URLs\n- Use 'skillctl add ./local-path -l <location>' for local paths", source, fromDir)
}

// isGitURL Determine if it's a Git URL
func isGitURL(source string) bool {
	// Check if it contains Git URL characteristics
	if strings.Contains(source, "://") {
		// Check if it's a Git protocol
		if strings.HasPrefix(source, "git://") ||
			strings.HasPrefix(source, "http://") ||
			strings.HasPrefix(source, "https://") ||
			strings.HasPrefix(source, "ssh://") {
			return true
		}
	}

	// Check if it's a Git URL format (git@github.com:user/repo.git)
	if strings.Contains(source, "git@") && strings.Contains(source, ":") {
		return true
	}

	return false
}
