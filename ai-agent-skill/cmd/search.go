package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skillctl/pkg/db"
	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var (
	searchLocal   bool   // Search local root directory
	searchProject bool   // Search project directory
	searchRepo    string // Search in repository database
)

var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search for skills",
	Long: `Search for available skills by keyword or tag.
Search locations (in order of priority):
- Use -l or --local to search in ~/.skills (installed skills)
- Use -p or --project to search in project directory (.agents/skills, etc.)
- Use -r or --repo to search in repository database
- By default, search in ./skills directory`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		return doSearch(keyword)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().BoolVarP(&searchLocal, "local", "l", false, "Search in local installed skills (~/.skills)")
	searchCmd.Flags().BoolVarP(&searchProject, "project", "p", false, "Search in project directory")
	searchCmd.Flags().StringVarP(&searchRepo, "repo", "r", "", "Search in repository database")
}

func doSearch(keyword string) error {
	fmt.Printf("🔍 Searching for skills matching: %s\n", keyword)
	fmt.Println()

	// Check if searching in repository database
	if searchRepo != "" {
		return doSearchInRepoDB(keyword)
	}

	var searchDirs []string
	var searchType string

	// Determine search location
	if searchLocal {
		// Search local root directory ~/.skills
		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return fmt.Errorf("failed to get skills directory: %w", err)
		}
		searchDirs = []string{skillsDir}
		searchType = "Local Installed Skills"
		fmt.Printf("📍 Location: %s\n", skillsDir)
		fmt.Println()
	} else if searchProject {
		// Search project directory
		baseDir := projectDir
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
		}
		searchDirs = []string{
			filepath.Join(baseDir, ".agents", "skills"),
			filepath.Join(baseDir, ".github", "skills"),
			filepath.Join(baseDir, ".claude", "skills"),
			filepath.Join(baseDir, ".copilot", "skills"),
			filepath.Join(baseDir, "examples", "skills"),
		}
		searchType = "Project Skills"
		fmt.Printf("📍 Location: %s\n", baseDir)
		fmt.Println()
	} else {
		// Default search local installed skills ~/.skills
		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return fmt.Errorf("failed to get skills directory: %w", err)
		}
		searchDirs = []string{skillsDir}
		searchType = "Local Installed Skills"
		fmt.Printf("📍 Location: %s\n", skillsDir)
		fmt.Println()
	}

	// Scan all specified directories
	var allSkills []CustomSkill
	for _, dir := range searchDirs {
		skills, err := scanCustomSkills(dir)
		if err != nil {
			// Directory may not exist, skip
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	// If searching local installed skills, also try to get from index
	if searchLocal {
		installedSkills, err := storage.ListSkills()
		if err == nil {
			skillsDir, _ := storage.GetSkillsDir()
			for _, skillMeta := range installedSkills {
				skillPath := filepath.Join(skillsDir, skillMeta.Name)
				skill := CustomSkill{
					Name:        skillMeta.Name,
					Description: skillMeta.Description,
					Version:     skillMeta.Version,
					Tags:        skillMeta.Tags,
					Author:      skillMeta.Author,
					Path:        skillPath,
					InstalledAt: skillMeta.InstalledAt,
				}
				// Avoid duplicates
				found := false
				for _, existing := range allSkills {
					if existing.Name == skill.Name {
						found = true
						break
					}
				}
				if !found {
					allSkills = append(allSkills, skill)
				}
			}
		}
	}

	// Search for matching skills
	var results []CustomSkill
	keywordLower := strings.ToLower(keyword)

	for _, skill := range allSkills {
		// Only search in names
		if strings.Contains(strings.ToLower(skill.Name), keywordLower) {
			results = append(results, skill)
		}
	}

	if len(results) == 0 {
		fmt.Printf("❌ No skills found matching '%s' in %s.\n", keyword, searchType)
		fmt.Println()
		fmt.Println("💡 Tips:")
		fmt.Println("   - Try different keywords")
		fmt.Println("   - Use -l to search in installed skills (~/.skills)")
		fmt.Println("   - Use -p to search in project directory")
		fmt.Println("   - Use -r to search in repository database")
		fmt.Println("   - Use fewer keywords for broader results")
		return nil
	}

	// Display search results
	fmt.Printf("📊 Search Results (%s):\n", searchType)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	for i, skill := range results {
		fmt.Printf("%d. %s\n", i+1, skill.Name)
		if skill.Description != "" {
			fmt.Printf("   📝 Description: %s\n", skill.Description)
		}
		if skill.Path != "" {
			fmt.Printf("   📍 Path: %s\n", skill.Path)
		}
		if skill.Version != "" {
			fmt.Printf("   📌 Version: %s\n", skill.Version)
		}
		if skill.Author != "" {
			fmt.Printf("   👤 Author: %s\n", skill.Author)
		}
		if skill.InstalledAt != "" {
			fmt.Printf("   📅 Installed: %s\n", skill.InstalledAt)
		}
		if len(skill.Tags) > 0 {
			fmt.Printf("   🏷️  Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
		fmt.Println()
	}

	fmt.Printf("✅ Found %d skill(s) matching '%s'\n", len(results), keyword)
	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   - Use 'skillctl install <name>' to install a skill")
	fmt.Println("   - Use 'skillctl remove <name>' to remove a skill")
	fmt.Println("   - Use 'skillctl list' to see all available skills")
	fmt.Println()

	return nil
}

// doSearchInRepoDB searches for skills in the repository database
func doSearchInRepoDB(keyword string) error {
	// Initialize database
	manager, err := initDB()
	if err != nil {
		return err
	}
	defer closeDB(manager)

	ctx := context.Background()

	// Get repository
	var repoID int64
	if searchRepo != "" {
		// Search in specific repository
		repo, err := manager.GetQueries().GetRepositoryByName(ctx, searchRepo)
		if err != nil {
			return fmt.Errorf("repository '%s' not found", searchRepo)
		}
		repoID = repo.ID
		fmt.Printf("📦 Repository: %s\n", searchRepo)
	} else {
		// Search in all repositories
		fmt.Printf("📦 All repositories\n")
	}

	fmt.Printf("   Keyword: %s\n", keyword)
	fmt.Println()

	// Get skills
	var skills []db.Skill
	if searchRepo != "" {
		skills, err = manager.GetQueries().ListSkillsByRepository(ctx, repoID)
		if err != nil {
			return fmt.Errorf("failed to list skills: %w", err)
		}
	} else {
		skills, err = manager.GetQueries().ListAllSkills(ctx)
		if err != nil {
			return fmt.Errorf("failed to list skills: %w", err)
		}
	}

	// Filter skills by keyword
	keywordLower := strings.ToLower(keyword)
	var results []db.Skill

	for _, skill := range skills {
		if containsKeyword(skill, keywordLower) {
			results = append(results, skill)
		}
	}

	if len(results) == 0 {
		fmt.Println("❌ No skills found matching your criteria.")
		return nil
	}

	// Display results
	fmt.Printf("📊 Search Results:\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	for i, skill := range results {
		fmt.Printf("%d. %s\n", i+1, skill.Name)

		if skill.Description.Valid && skill.Description.String != "" {
			fmt.Printf("   📝 Description: %s\n", skill.Description.String)
		}

		if skill.Version.Valid && skill.Version.String != "" {
			fmt.Printf("   📌 Version: %s\n", skill.Version.String)
		}

		if skill.Author.Valid && skill.Author.String != "" {
			fmt.Printf("   👤 Author: %s\n", skill.Author.String)
		}

		if skill.Category.Valid && skill.Category.String != "" {
			fmt.Printf("   📂 Category: %s\n", skill.Category.String)
		}

		if skill.RepoUrl.Valid && skill.RepoUrl.String != "" {
			fmt.Printf("   🔗 URL: %s\n", skill.RepoUrl.String)
		}

		// Get tags
		tags, err := manager.GetQueries().GetTagsBySkillID(ctx, skill.ID)
		if err == nil && len(tags) > 0 {
			tagNames := make([]string, len(tags))
			for j, tag := range tags {
				tagNames[j] = tag.Tag
			}
			fmt.Printf("   🏷️  Tags: %s\n", strings.Join(tagNames, ", "))
		}

		fmt.Println()
	}

	fmt.Printf("✅ Found %d skill(s) matching '%s'\n", len(results), keyword)
	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   - Use 'skillctl add <name> -p <location>' to add a skill to your project")

	return nil
}

// containsKeyword checks if skill contains the keyword
func containsKeyword(skill db.Skill, keywordLower string) bool {
	if strings.Contains(strings.ToLower(skill.Name), keywordLower) {
		return true
	}
	if skill.Description.Valid && strings.Contains(strings.ToLower(skill.Description.String), keywordLower) {
		return true
	}
	return false
}
