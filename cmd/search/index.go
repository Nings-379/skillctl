package searchcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/seekthought/skill/pkg/db"
	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

type options struct {
	local      bool
	project    bool
	global     bool
	repo       string
	limit      int
	projectDir string
}

func NewCommand() *cobra.Command {
	opt := &options{}

	cmd := &cobra.Command{
		Use:   "search [keyword]",
		Short: "Search for skills",
		Long: `Search for available skills by keyword or tag.
Search locations (in order of priority):
- Use -g or --global to search GitHub globally
- Use -l or --local to search in ~/.skills (installed skills)
- Use -p or --project to search in project directory (.agents/skills, etc.)
- Use -r or --repo to search in repository database
- By default, search in ~/.skills directory`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(opt, args[0])
		},
	}

	cmd.Flags().BoolVarP(&opt.global, "global", "g", false, "Search GitHub globally for skill repositories")
	cmd.Flags().BoolVarP(&opt.local, "local", "l", false, "Search in local installed skills (~/.skills)")
	cmd.Flags().BoolVarP(&opt.project, "project", "p", false, "Search in project directory")
	cmd.Flags().StringVarP(&opt.repo, "repo", "r", "", "Search in repository database")
	cmd.Flags().IntVar(&opt.limit, "limit", 10, "Maximum number of search results for --global")
	cmd.Flags().StringVarP(&opt.projectDir, "dir", "d", "", "Project directory (default: current directory)")

	return cmd
}

func runSearch(opt *options, keyword string) error {
	fmt.Printf("🔍 Searching for skills matching: %s\n\n", keyword)

	if opt.global {
		if opt.repo != "" || opt.local || opt.project {
			return fmt.Errorf("--global cannot be combined with --repo, --local, or --project")
		}
		return searchGitHubGlobal(keyword, opt.limit)
	}

	if opt.repo != "" {
		return searchInRepoDB(opt.repo, keyword)
	}

	var searchDirs []string
	searchType := "Local Installed Skills"

	if opt.project {
		baseDir, err := cmdutils.ResolveProjectDir(opt.projectDir)
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		searchDirs = cmdutils.ProjectSkillDirs(baseDir)
		searchType = "Project Skills"
		fmt.Printf("📍 Location: %s\n\n", baseDir)
	} else {
		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return fmt.Errorf("failed to get skills directory: %w", err)
		}
		searchDirs = []string{skillsDir}
		fmt.Printf("📍 Location: %s\n\n", skillsDir)
	}

	var allSkills []cmdutils.CustomSkill
	for _, dir := range searchDirs {
		skills, err := cmdutils.ScanCustomSkills(dir)
		if err != nil {
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if opt.local || !opt.project {
		installedSkills, err := storage.ListSkills()
		if err == nil {
			skillsDir, _ := storage.GetSkillsDir()
			for _, skillMeta := range installedSkills {
				skill := cmdutils.SkillMetaToCustomSkill(skillMeta, filepath.Join(skillsDir, skillMeta.Name))
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

	keywordLower := strings.ToLower(keyword)
	var results []cmdutils.CustomSkill
	for _, skill := range allSkills {
		if strings.Contains(strings.ToLower(skill.Name), keywordLower) {
			results = append(results, skill)
		}
	}

	if len(results) == 0 {
		fmt.Printf("❌ No skills found matching '%s' in %s.\n\n", keyword, searchType)
		fmt.Println("💡 Tips:")
		fmt.Println("   - Use -g to search GitHub globally")
		fmt.Println("   - Try different keywords")
		fmt.Println("   - Use -l to search in installed skills (~/.skills)")
		fmt.Println("   - Use -p to search in project directory")
		fmt.Println("   - Use -r to search in repository database")
		fmt.Println("   - Use fewer keywords for broader results")
		return nil
	}

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

	fmt.Printf("✅ Found %d skill(s) matching '%s'\n\n", len(results), keyword)
	fmt.Println("💡 Tips:")
	fmt.Println("   - Use 'skillctl install <name>' to install a skill")
	fmt.Println("   - Use 'skillctl remove <name>' to remove a skill")
	fmt.Println("   - Use 'skillctl list' to see all available skills")
	return nil
}

func searchInRepoDB(repoName, keyword string) error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()
	repo, err := manager.GetQueries().GetRepositoryByName(ctx, repoName)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", repoName)
	}

	fmt.Printf("📦 Repository: %s\n", repoName)
	fmt.Printf("   Keyword: %s\n\n", keyword)

	skills, err := manager.GetQueries().ListSkillsByRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

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

	fmt.Printf("✅ Found %d skill(s) matching '%s'\n\n", len(results), keyword)
	fmt.Println("💡 Tips:")
	fmt.Println("   - Use 'skillctl add <name> -p <location>' to add a skill to your project")
	return nil
}

func containsKeyword(skill db.Skill, keywordLower string) bool {
	if strings.Contains(strings.ToLower(skill.Name), keywordLower) {
		return true
	}
	if skill.Description.Valid && strings.Contains(strings.ToLower(skill.Description.String), keywordLower) {
		return true
	}
	return false
}
