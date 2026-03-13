package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"time"

	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	updateProject bool   // Update skill in project directory
	updateLocal   bool   // Update skill in local directory
	updateRepo    string // Repository name for database search
)

var updateCmd = &cobra.Command{
	Use:   "update <skill-name>",
	Short: "Update a skill from remote repository",
	Long: `Update an installed skill by comparing with the remote repository and downloading the latest version.

You can update skills from database or Git URLs:
  - skillctl update <skill-name> - Update skill from database (compare version)
  - skillctl update <skill-name> -l - Update skill in local directory
  - skillctl update <skill-name> -p - Update skill in project directory
  - skillctl update <skill-name> -r <repo-name> - Update skill from specific repository`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		return doUpdate(skillName)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVarP(&updateProject, "project", "p", false, "Update skill in project directory")
	updateCmd.Flags().BoolVarP(&updateLocal, "local", "l", false, "Update skill in local directory")
	updateCmd.Flags().StringVarP(&updateRepo, "repo", "r", "", "Repository name to search skills from")
}

func doUpdate(skillName string) error {
	// Determine update target
	if !updateProject && !updateLocal {
		updateLocal = true // Default to local directory
	}

	// Search for skill in database
	fmt.Printf("🔍 Searching for skill '%s'", skillName)
	if updateRepo != "" {
		fmt.Printf(" in repository '%s'...\n", updateRepo)
	} else {
		fmt.Println(" in database...")
	}

	skillFromDB, err := findSkillInDatabase(skillName, updateRepo)
	if err != nil {
		tip := "💡 Tip: Use 'skillctl search %s' to search for skills, or provide a Git URL"
		if updateRepo != "" {
			tip = "💡 Tip: Use 'skillctl search -r %s %s' to search in this repository"
			return fmt.Errorf("skill '%s' not found in repository '%s': %w\n\n%s", skillName, updateRepo, err, fmt.Sprintf(tip, updateRepo, skillName))
		}
		return fmt.Errorf("skill '%s' not found in database: %w\n\n%s", skillName, err, fmt.Sprintf(tip, skillName))
	}

	fmt.Printf("📦 Found skill in database: %s\n", skillFromDB.Name)
	if skillFromDB.Description.Valid && skillFromDB.Description.String != "" {
		fmt.Printf("   📝 %s\n", skillFromDB.Description.String)
	}
	if skillFromDB.Version.Valid && skillFromDB.Version.String != "" {
		fmt.Printf("   📌 Current version: %s\n", skillFromDB.Version.String)
	}
	if skillFromDB.RepoUrl.Valid && skillFromDB.RepoUrl.String != "" {
		fmt.Printf("   🔗 %s\n", skillFromDB.RepoUrl.String)
	}
	fmt.Println()

	// Determine target path
	var targetPath string
	if updateProject {
		// Update in project directory
		baseDir := projectDir
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
		}

		// Search in project directories
		projectSkillsDirs := []string{
			filepath.Join(baseDir, ".agents", "skills"),
			filepath.Join(baseDir, ".github", "skills"),
			filepath.Join(baseDir, ".claude", "skills"),
			filepath.Join(baseDir, ".copilot", "skills"),
			filepath.Join(baseDir, "examples", "skills"),
		}

		for _, dir := range projectSkillsDirs {
			if _, err := os.Stat(filepath.Join(dir, skillName)); err == nil {
				targetPath = filepath.Join(dir, skillName)
				break
			}
		}

		if targetPath == "" {
			return fmt.Errorf("skill '%s' not found in project directory", skillName)
		}
	} else {
		// Update in local directory
		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return fmt.Errorf("failed to get skills directory: %w", err)
		}

		targetPath = filepath.Join(skillsDir, skillName)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return fmt.Errorf("skill '%s' is not installed", skillName)
		}
	}

	fmt.Printf("📍 Target location: %s\n", targetPath)
	fmt.Println()

	// Check if skill is already installed locally
	existingSkill, err := storage.GetSkill(skillName)
	if err == nil && existingSkill != nil && !updateProject {
		fmt.Printf("   Installed version: %s\n", existingSkill.Version)
		if existingSkill.InstalledAt != "" {
			installedTime, _ := time.Parse(time.RFC3339, existingSkill.InstalledAt)
			fmt.Printf("   Installed at: %s\n", installedTime.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	// Download from remote repository
	gitURL := skillFromDB.RepoUrl.String
	if gitURL == "" {
		return fmt.Errorf("skill '%s' found in database but has no repository URL", skillName)
	}

	fmt.Printf("📥 Downloading latest version from: %s\n", gitURL)
	fmt.Println()

	// Use sparse-checkout to download only the specific skill
	subdir := filepath.Join("skills", skillName)

	// Create temporary directory for download
	tempDir := targetPath + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tempDir), 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := cloneRepositoryWithSubdir(gitURL, tempDir, subdir); err != nil {
		return fmt.Errorf("failed to download skill: %w", err)
	}
	fmt.Println("✓ Latest version downloaded successfully")
	fmt.Println()

	// Parse metadata from downloaded version
	newMeta, err := parseSkillMetadata(tempDir, skillName)
	if err != nil {
		fmt.Printf("⚠ Warning: Could not parse new skill metadata: %v\n", err)
		newMeta = storage.SkillMeta{
			Name:        skillName,
			Description: fmt.Sprintf("Skill updated from %s", gitURL),
			InstalledAt: time.Now().Format(time.RFC3339),
		}
	} else {
		fmt.Println("📋 New skill metadata parsed")
		if newMeta.Version != "" {
			fmt.Printf("   New version: %s\n", newMeta.Version)
		}
	}
	fmt.Println()

	// Check for version change
	if existingSkill != nil && existingSkill.Version != "" && newMeta.Version != "" {
		if existingSkill.Version == newMeta.Version {
			fmt.Printf("✅ Skill '%s' is already up to date (version %s)\n", skillName, existingSkill.Version)
			return nil
		}
		fmt.Printf("📤 Updating from version %s to %s\n", existingSkill.Version, newMeta.Version)
	} else {
		fmt.Printf("📤 Installing new version\n")
	}
	fmt.Println()

	// Remove old version
	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to remove old version: %w", err)
	}

	// Move new version to target
	if err := os.Rename(tempDir, targetPath); err != nil {
		return fmt.Errorf("failed to install new version: %w", err)
	}
	fmt.Println("✓ New version installed")
	fmt.Println()

	// Update metadata
	newMeta.InstalledAt = time.Now().Format(time.RFC3339)
	if updateLocal || !updateProject {
		if err := storage.AddSkill(skillName, newMeta); err != nil {
			fmt.Printf("⚠ Warning: Could not update skill metadata: %v\n", err)
		} else {
			fmt.Println("✓ Skill metadata updated")
		}
	}

	// Parse metadata for display
	displayMeta, err := parseSkillMetadata(targetPath, skillName)
	if err == nil {
		fmt.Println()
		fmt.Printf("✅ Skill '%s' has been successfully updated!\n", skillName)
		if displayMeta.Version != "" {
			fmt.Printf("   Version: %s\n", displayMeta.Version)
		}
		if displayMeta.Author != "" {
			fmt.Printf("   Author: %s\n", displayMeta.Author)
		}
		if displayMeta.Description != "" {
			fmt.Printf("   Description: %s\n", displayMeta.Description)
		}
	} else {
		fmt.Println()
		fmt.Printf("✅ Skill '%s' has been successfully updated!\n", skillName)
	}
	fmt.Printf("   Location: %s\n", targetPath)

	return nil
}
