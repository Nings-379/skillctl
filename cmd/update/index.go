package updatecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

type options struct {
	project    bool
	local      bool
	repo       string
	projectDir string
}

func NewCommand() *cobra.Command {
	opt := &options{}

	cmd := &cobra.Command{
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
			return runUpdate(opt, args[0])
		},
	}

	cmd.Flags().BoolVarP(&opt.project, "project", "p", false, "Update skill in project directory")
	cmd.Flags().BoolVarP(&opt.local, "local", "l", false, "Update skill in local directory")
	cmd.Flags().StringVarP(&opt.repo, "repo", "r", "", "Repository name to search skills from")
	cmd.Flags().StringVarP(&opt.projectDir, "dir", "d", "", "Project directory (default: current directory)")

	return cmd
}

func runUpdate(opt *options, skillName string) error {
	if !opt.project && !opt.local {
		opt.local = true
	}

	fmt.Printf("🔍 Searching for skill '%s'", skillName)
	if opt.repo != "" {
		fmt.Printf(" in repository '%s'...\n", opt.repo)
	} else {
		fmt.Println(" in database...")
	}

	skillFromDB, err := cmdutils.FindSkillInDatabase(skillName, opt.repo)
	if err != nil {
		tip := "💡 Tip: Use 'skillctl search %s' to search for skills, or provide a Git URL"
		if opt.repo != "" {
			tip = "💡 Tip: Use 'skillctl search -r %s %s' to search in this repository"
			return fmt.Errorf("skill '%s' not found in repository '%s': %w\n\n%s", skillName, opt.repo, err, fmt.Sprintf(tip, opt.repo, skillName))
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

	var targetPath string
	if opt.project {
		baseDir, err := cmdutils.ResolveProjectDir(opt.projectDir)
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		targetPath, err = cmdutils.FindSkillInProject(baseDir, skillName)
		if err != nil {
			return err
		}
	} else {
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

	existingSkill, err := storage.GetSkill(skillName)
	if err == nil && existingSkill != nil && !opt.project {
		fmt.Printf("   Installed version: %s\n", existingSkill.Version)
		if existingSkill.InstalledAt != "" {
			installedTime, _ := time.Parse(time.RFC3339, existingSkill.InstalledAt)
			fmt.Printf("   Installed at: %s\n", installedTime.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	gitURL := skillFromDB.RepoUrl.String
	if gitURL == "" {
		return fmt.Errorf("skill '%s' found in database but has no repository URL", skillName)
	}

	fmt.Printf("📥 Downloading latest version from: %s\n", gitURL)
	fmt.Println()

	subdir := filepath.Join("skills", skillName)
	tempDir := targetPath + ".tmp"
	if err := os.MkdirAll(filepath.Dir(tempDir), 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := cmdutils.CloneRepositoryWithSubdir(gitURL, tempDir, subdir); err != nil {
		return fmt.Errorf("failed to download skill: %w", err)
	}
	fmt.Println("✓ Latest version downloaded successfully")
	fmt.Println()

	newMeta, err := cmdutils.ParseSkillMetadata(tempDir, skillName)
	if err != nil {
		fmt.Printf("⚠ Warning: Could not parse new skill metadata: %v\n", err)
		newMeta = storage.SkillMeta{Name: skillName, Description: fmt.Sprintf("Skill updated from %s", gitURL), InstalledAt: time.Now().Format(time.RFC3339)}
	} else {
		fmt.Println("📋 New skill metadata parsed")
		if newMeta.Version != "" {
			fmt.Printf("   New version: %s\n", newMeta.Version)
		}
	}
	fmt.Println()

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

	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to remove old version: %w", err)
	}
	if err := os.Rename(tempDir, targetPath); err != nil {
		return fmt.Errorf("failed to install new version: %w", err)
	}
	fmt.Println("✓ New version installed")
	fmt.Println()

	newMeta.InstalledAt = time.Now().Format(time.RFC3339)
	if opt.local || !opt.project {
		if err := storage.AddSkill(skillName, newMeta); err != nil {
			fmt.Printf("⚠ Warning: Could not update skill metadata: %v\n", err)
		} else {
			fmt.Println("✓ Skill metadata updated")
		}
	}

	displayMeta, err := cmdutils.ParseSkillMetadata(targetPath, skillName)
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
