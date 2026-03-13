package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	removeProject bool // Whether to remove skill from project
)

var removeCmd = &cobra.Command{
	Use:   "remove [skill-name]",
	Short: "Remove a skill from the local repository or project",
	Long:  `Uninstall and remove a skill from your local machine or project directory.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		return doRemove(skillName)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&removeProject, "project", "p", false, "Remove skill from project directory")
}

func doRemove(skillName string) error {
	var skillPath string
	var skillMeta *storage.SkillMeta
	var err error
	var projectDir string

	if removeProject {
		// Remove skill from project directory
		// Determine project directory
		projectDir = projectDir
		if projectDir == "" {
			projectDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
		}

		// Find skill location in project
		projectSkillsDirs := []string{
			filepath.Join(projectDir, ".agents", "skills", skillName),
			filepath.Join(projectDir, ".github", "skills", skillName),
			filepath.Join(projectDir, ".claude", "skills", skillName),
			filepath.Join(projectDir, ".copilot", "skills", skillName),
			filepath.Join(projectDir, "examples", "skills", skillName),
		}

		for _, dir := range projectSkillsDirs {
			if _, err := os.Stat(dir); err == nil {
				skillPath = dir
				break
			}
		}

		if skillPath == "" {
			return fmt.Errorf("skill '%s' not found in project directory: %s", skillName, projectDir)
		}

		// Try to parse skill metadata
		parsedMeta, parseErr := parseCustomSkill(skillPath, skillName)
		if parseErr == nil {
			skillMeta = &storage.SkillMeta{
				Name:        parsedMeta.Name,
				Description: parsedMeta.Description,
				Version:     parsedMeta.Version,
				Tags:        parsedMeta.Tags,
				Author:      parsedMeta.Author,
			}
		}
	} else {
		// Remove skill from local ~/.skills
		skillMeta, err = storage.GetSkill(skillName)
		if err != nil {
			return fmt.Errorf("skill '%s' is not installed", skillName)
		}

		// Get skills directory
		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return fmt.Errorf("failed to get skills directory: %w", err)
		}

		skillPath = filepath.Join(skillsDir, skillName)
	}

	// Display skill information
	fmt.Printf("🗑️  Removing skill: %s\n", skillName)
	fmt.Println()
	if removeProject {
		fmt.Printf("   Type: Project skill\n")
		fmt.Printf("   Project: %s\n", projectDir)
	} else {
		fmt.Printf("   Type: Installed skill\n")
	}

	if skillMeta != nil {
		if skillMeta.Description != "" {
			fmt.Printf("   Description: %s\n", skillMeta.Description)
		}
		if skillMeta.Version != "" {
			fmt.Printf("   Version: %s\n", skillMeta.Version)
		}
		if skillMeta.Author != "" {
			fmt.Printf("   Author: %s\n", skillMeta.Author)
		}
		if skillMeta.InstalledAt != "" {
			fmt.Printf("   Installed: %s\n", skillMeta.InstalledAt)
		}
	}
	fmt.Printf("   Location: %s\n", skillPath)
	fmt.Println()

	// Confirm removal
	fmt.Printf("Are you sure you want to remove '%s'? [y/N]: ", skillName)
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("❌ Removal cancelled.")
		return nil
	}

	fmt.Println()

	// Remove skill directory
	fmt.Println("🗑️  Removing skill files...")
	if err := os.RemoveAll(skillPath); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}
	fmt.Println("✓ Files removed")

	// If not project skill, remove from index
	if !removeProject {
		fmt.Println("📋 Updating skill index...")
		if err := storage.RemoveSkill(skillName); err != nil {
			return fmt.Errorf("failed to remove skill from index: %w", err)
		}
		fmt.Println("✓ Index updated")
	}

	fmt.Println()
	fmt.Printf("✅ Skill '%s' has been successfully removed.\n", skillName)

	return nil
}
