package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"skillctl/pkg/db"
	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	repoFlag string // Repository name flag
)

var installCmd = &cobra.Command{
	Use:   "install <git-url-or-skill-name> [skill-name]",
	Short: "Install a skill from a Git repository or database",
	Long: `Clone a skill repository and install it to the local skills directory.
If skill-name is provided, only install that specific skill from the repository's skills subdirectory.

You can also install skills by name from the indexed repository database:
  - skillctl install <skill-name> - Search and install from database
  - skillctl install <git-url> [skill-name] - Install from Git URL
  - skillctl install -r <repo-name> <skill-name> - Install skill from specific repository`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doInstall(args)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVarP(&repoFlag, "repo", "r", "", "Repository name to search skills from")
}

func doInstall(args []string) error {
	arg0 := args[0]
	skillName := ""
	if len(args) > 1 {
		skillName = args[1]
	}

	// Check if first argument is a Git URL
	var gitURL string
	var dbSkill *db.Skill

	if isGitURL(arg0) {
		// Direct Git URL
		gitURL = arg0
		fmt.Printf("📦 Installing skill from: %s\n", gitURL)
	} else {
		// Try to find in database
		fmt.Printf("🔍 Searching for skill '%s'", arg0)
		if repoFlag != "" {
			fmt.Printf(" in repository '%s'...\n", repoFlag)
		} else {
			fmt.Println(" in database...")
		}

		skillFromDB, err := findSkillInDatabase(arg0, repoFlag)
		if err != nil {
			tip := "💡 Tip: Use 'skillctl search %s' to search for skills, or provide a Git URL"
			if repoFlag != "" {
				tip = "💡 Tip: Use 'skillctl search -r %s %s' to search in this repository, or provide a Git URL"
				return fmt.Errorf("skill '%s' not found in repository '%s': %w\n\n%s", arg0, repoFlag, err, fmt.Sprintf(tip, repoFlag, arg0))
			}
			return fmt.Errorf("skill '%s' not found in database: %w\n\n%s", arg0, err, fmt.Sprintf(tip, arg0))
		}
		dbSkill = &skillFromDB

		if dbSkill.RepoUrl.Valid && dbSkill.RepoUrl.String != "" {
			gitURL = dbSkill.RepoUrl.String
			fmt.Printf("📦 Found skill in database: %s\n", dbSkill.Name)
			if dbSkill.Description.Valid && dbSkill.Description.String != "" {
				fmt.Printf("   📝 %s\n", dbSkill.Description.String)
			}
			fmt.Printf("   🔗 %s\n", gitURL)
		} else {
			return fmt.Errorf("skill '%s' found in database but has no repository URL", arg0)
		}
	}

	if skillName != "" {
		fmt.Printf("🎯 Target skill: %s\n", skillName)
	}
	fmt.Println()

	// Extract repository name
	repoName, err := extractRepoName(gitURL)
	if err != nil {
		return fmt.Errorf("failed to extract repository name: %w", err)
	}

	// Determine skill name (prioritize parameter, otherwise use repository name or database skill name)
	targetSkillName := skillName
	if targetSkillName == "" {
		if dbSkill != nil {
			targetSkillName = dbSkill.Name
		} else {
			targetSkillName = repoName
		}
	}

	// Check if already installed
	existing, err := storage.GetSkill(targetSkillName)
	if err == nil && existing != nil {
		return fmt.Errorf("skill '%s' is already installed (version %s)", targetSkillName, existing.Version)
	}

	// Ensure skills directory exists
	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get skills directory: %w", err)
	}

	if err := storage.EnsureSkillsDir(); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Clone repository
	installPath := filepath.Join(skillsDir, targetSkillName)
	fmt.Printf("📥 Cloning repository to: %s\n", installPath)

	// Determine if we should use sparse-checkout
	// Use sparse-checkout when:
	// 1. Installing from database (dbSkill != nil) - clone only the specific skill folder
	// 2. skillName parameter is provided - clone only the specified subdirectory
	useSparseCheckout := (dbSkill != nil) || (skillName != "")

	var targetSubdir string
	if dbSkill != nil {
		// From database: clone only the specific skill folder from skills/ directory
		targetSubdir = filepath.Join("skills", targetSkillName)
	} else if skillName != "" {
		// From Git URL with skill name parameter
		targetSubdir = filepath.Join("skills", skillName)
	}

	if useSparseCheckout {
		// Use sparse-checkout to clone only the specified subdirectory
		if err := cloneRepositoryWithSubdir(gitURL, installPath, targetSubdir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Clone entire repository
		if err := cloneRepository(gitURL, installPath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}
	fmt.Println("✓ Repository cloned successfully")

	// Parse skill metadata
	fmt.Println("📋 Parsing skill metadata...")
	skillMeta, err := parseSkillMetadata(installPath, targetSkillName)
	if err != nil {
		fmt.Printf("⚠ Warning: Could not parse skill metadata: %v\n", err)
		// Use default metadata
		skillMeta = storage.SkillMeta{
			Name:        targetSkillName,
			Description: fmt.Sprintf("Skill installed from %s", gitURL),
			InstalledAt: time.Now().Format(time.RFC3339),
		}
	} else {
		fmt.Println("✓ Skill metadata parsed")
	}

	// Add to index
	if err := storage.AddSkill(targetSkillName, skillMeta); err != nil {
		return fmt.Errorf("failed to add skill to index: %w", err)
	}

	fmt.Println()
	fmt.Printf("✅ Skill '%s' has been successfully installed!\n", targetSkillName)
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

	return nil
}

// findSkillInDatabase searches for a skill by name in the database
// If repoName is provided, only searches within that specific repository
func findSkillInDatabase(name string, repoName string) (db.Skill, error) {
	manager, err := initDB()
	if err != nil {
		return db.Skill{}, err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Get all skills
	skills, err := manager.GetQueries().ListAllSkills(ctx)
	if err != nil {
		return db.Skill{}, fmt.Errorf("failed to list skills: %w", err)
	}

	// Get all repositories for name mapping
	var repoMap = make(map[int64]string)
	if repoName != "" {
		repos, err := manager.GetQueries().ListAllRepositories(ctx)
		if err != nil {
			return db.Skill{}, fmt.Errorf("failed to list repositories: %w", err)
		}
		var targetRepoID int64 = -1
		for _, repo := range repos {
			repoMap[repo.ID] = repo.Name
			if strings.EqualFold(repo.Name, repoName) {
				targetRepoID = repo.ID
			}
		}
		if targetRepoID == -1 {
			return db.Skill{}, fmt.Errorf("repository '%s' not found", repoName)
		}
		// Filter skills by repository
		var filteredSkills []db.Skill
		for _, skill := range skills {
			if skill.RepositoryID == targetRepoID {
				filteredSkills = append(filteredSkills, skill)
			}
		}
		skills = filteredSkills
	}

	// First try exact match
	for _, skill := range skills {
		if strings.EqualFold(skill.Name, name) {
			return skill, nil
		}
	}

	// Then try slug match
	for _, skill := range skills {
		if strings.EqualFold(skill.Slug, name) {
			return skill, nil
		}
	}

	// If no exact match, return error with suggestions
	var suggestions []string
	for _, skill := range skills {
		if strings.Contains(strings.ToLower(skill.Name), strings.ToLower(name)) {
			suggestion := skill.Name
			if repoName != "" && len(repoMap) > 0 {
				if repoNameDisplay, ok := repoMap[skill.RepositoryID]; ok {
					suggestion = fmt.Sprintf("%s (from repo: %s)", skill.Name, repoNameDisplay)
				}
			}
			suggestions = append(suggestions, suggestion)
		}
		if len(suggestions) >= 3 {
			break
		}
	}

	errMsg := fmt.Sprintf("skill '%s' not found", name)
	if repoName != "" {
		errMsg += fmt.Sprintf(" in repository '%s'", repoName)
	}
	if len(suggestions) > 0 {
		errMsg += fmt.Sprintf("\n\nDid you mean:\n  - %s", strings.Join(suggestions, "\n  - "))
	}

	return db.Skill{}, fmt.Errorf(errMsg)
}
