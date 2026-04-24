package listcmd

import (
	"context"
	"fmt"
	"strings"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

const listRepoAll = "__all_repositories__"

type options struct {
	remoteURL  string
	remote     bool
	project    bool
	scanDir    string
	repo       string
	projectDir string
	versions   bool
}

type customSkillRenderOptions struct {
	PathLabel     string
	BasePath      string
	ShowPath      bool
	ShowVersion   bool
	ShowAuthor    bool
	ShowInstalled bool
	ShowTags      bool
}

func NewCommand() *cobra.Command {
	opt := &options{}

	cmd := &cobra.Command{
		Use:   "list [url]",
		Short: "List all installed skills",
		Long: `Display a list of all skills installed in the local skills directory, current project, or remote repository.
When using --remote, provide a URL to list skills from any remote Git repository.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opt, args)
		},
	}

	cmd.Flags().BoolVarP(&opt.remote, "remote", "R", false, "List skills from remote repository (provide URL as argument)")
	cmd.Flags().BoolVarP(&opt.project, "project", "p", false, "List skills in current project directory")
	cmd.Flags().StringVarP(&opt.repo, "repo", "r", "", "List repositories, or list skills from a specific repository when a name is provided")
	cmd.Flags().StringVarP(&opt.scanDir, "skills-dir", "s", "", "Scan skills from a specific directory")
	cmd.Flags().StringVarP(&opt.projectDir, "dir", "d", "", "Project directory (default: current directory)")
	cmd.Flags().BoolVar(&opt.versions, "versions", false, "List all available versions of a skill (requires skill name as argument)")
	cmd.Flags().Lookup("repo").NoOptDefVal = listRepoAll
	return cmd
}

func runList(opt *options, args []string) error {
	if opt.versions {
		if len(args) == 0 {
			return fmt.Errorf("please provide a skill name when using --versions flag\nUsage: skillctl list --versions <skill-name> -r <repo-name>")
		}
		return listVersions(opt, args[0])
	}

	if opt.repo != "" {
		if opt.repo == listRepoAll && len(args) > 0 {
			opt.repo = args[0]
		}
		return listFromRepoDB(opt)
	}

	if opt.remote {
		if len(args) == 0 {
			return fmt.Errorf("please provide a URL when using --remote flag\nUsage: skillctl list -R <url>")
		}
		return listRemote(opt, args[0])
	}

	if opt.project {
		return listProject(opt)
	}

	if opt.scanDir != "" {
		return listScanDir(opt.scanDir)
	}

	if len(args) > 0 {
		return listRemote(opt, args[0])
	}

	return listLocal(opt)
}

func listFromRepoDB(opt *options) error {
	if opt.repo == "" || opt.repo == listRepoAll {
		return listRepositories()
	}

	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()
	repo, err := manager.GetQueries().GetRepositoryByName(ctx, opt.repo)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", opt.repo)
	}

	skills, err := manager.GetQueries().ListSkillsByRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to list repository skills: %w", err)
	}

	fmt.Printf("📦 Repository: %s\n", repo.Name)
	fmt.Printf("   URL: %s\n", repo.Url)
	if repo.Description.Valid && repo.Description.String != "" {
		fmt.Printf("   Description: %s\n", repo.Description.String)
	}
	fmt.Println()

	if len(skills) == 0 {
		fmt.Println("No indexed skills found in this repository.")
		return nil
	}

	for i, skill := range skills {
		fmt.Printf("%d. %s\n", i+1, skill.Name)
		if skill.Description.Valid && skill.Description.String != "" {
			fmt.Printf("   Description: %s\n", skill.Description.String)
		}
		if skill.Version.Valid && skill.Version.String != "" {
			fmt.Printf("   Version: %s\n", skill.Version.String)
		}
		if skill.Author.Valid && skill.Author.String != "" {
			fmt.Printf("   Author: %s\n", skill.Author.String)
		}
		if skill.Category.Valid && skill.Category.String != "" {
			fmt.Printf("   Category: %s\n", skill.Category.String)
		}
		fmt.Println()
	}

	fmt.Printf("✅ Found %d skill(s) in repository '%s'\n", len(skills), repo.Name)
	return nil
}

func listRepositories() error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	repos, err := manager.GetQueries().ListAllRepositories(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories configured.")
		fmt.Println("💡 Use 'skillctl add -r <repo-url>' to add one.")
		return nil
	}

	fmt.Println("Configured repositories:")
	fmt.Println(strings.Repeat("=", 60))
	for i, repo := range repos {
		fmt.Printf("%d. %s\n", i+1, repo.Name)
		fmt.Printf("   URL: %s\n", repo.Url)
		if repo.Description.Valid && repo.Description.String != "" {
			fmt.Printf("   Description: %s\n", repo.Description.String)
		}
		if repo.IsDefault == 1 {
			fmt.Println("   Default: yes")
		}
		if repo.Priority != 0 {
			fmt.Printf("   Priority: %d\n", repo.Priority)
		}
		fmt.Println()
	}

	return nil
}

func listVersions(opt *options, skillName string) error {
	var gitURL string

	if opt.repo != "" && opt.repo != listRepoAll {
		manager, err := cmdutils.InitDB()
		if err != nil {
			return err
		}
		defer cmdutils.CloseDB(manager)

		ctx := context.Background()
		repo, err := manager.GetQueries().GetRepositoryByName(ctx, opt.repo)
		if err != nil {
			return fmt.Errorf("repository '%s' not found", opt.repo)
		}
		gitURL = repo.Url
	} else {
		skillFromDB, err := cmdutils.FindSkillInDatabase(skillName, "")
		if err != nil {
			return fmt.Errorf("skill '%s' not found in database. Use -r <repo-name> to specify the repository", skillName)
		}
		if !skillFromDB.RepoUrl.Valid || skillFromDB.RepoUrl.String == "" {
			return fmt.Errorf("skill '%s' has no repository URL", skillName)
		}
		gitURL = skillFromDB.RepoUrl.String
	}

	fmt.Printf("🔍 Listing versions for skill '%s'...\n", skillName)
	fmt.Printf("   Repository: %s\n\n", gitURL)

	prefix := skillName + "@"
	tags, err := cmdutils.ListRemoteTags(gitURL, prefix)
	if err != nil {
		return fmt.Errorf("failed to list remote tags: %w", err)
	}

	if len(tags) == 0 {
		fmt.Printf("No version tags found for skill '%s'.\n", skillName)
		fmt.Println("💡 Version tags are created when pushing with skillctl push.")
		fmt.Printf("   Tag format: %s@<version>\n", skillName)
		return nil
	}

	fmt.Printf("Available versions for '%s':\n", skillName)
	fmt.Println(strings.Repeat("=", 40))
	for i, tag := range tags {
		version := strings.TrimPrefix(tag, prefix)
		fmt.Printf("  %d. %s\n", i+1, version)
	}
	fmt.Printf("\nTotal: %d version(s)\n", len(tags))
	fmt.Printf("\n💡 Install a specific version:\n   skillctl install %s -v <version>\n", skillName)

	return nil
}
