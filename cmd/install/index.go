package installcmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/seekthought/skill/pkg/db"
	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

type options struct {
	repo    string
	version string
}

func NewCommand() *cobra.Command {
	opt := &options{}

	cmd := &cobra.Command{
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
			return runInstall(opt, args)
		},
	}

	cmd.Flags().StringVarP(&opt.repo, "repo", "r", "", "Repository name to search skills from")
	cmd.Flags().StringVarP(&opt.version, "version", "v", "", "Install a specific version (by git tag <skill>@<version>)")
	cmd.AddCommand(newSyncRepositoryCommand())
	cmd.AddCommand(newDefaultRepositoryCommand())
	return cmd
}

func runInstall(opt *options, args []string) error {
	arg0 := args[0]
	skillName := ""
	if len(args) > 1 {
		skillName = args[1]
	}

	var gitURL string
	var dbSkill *db.Skill

	if cmdutils.IsGitURL(arg0) {
		gitURL = arg0
		fmt.Printf("📦 Installing skill from: %s\n", gitURL)
	} else {
		fmt.Printf("🔍 Searching for skill '%s'", arg0)
		if opt.repo != "" {
			fmt.Printf(" in repository '%s'...\n", opt.repo)
		} else {
			fmt.Println(" in database...")
		}

		skillFromDB, err := cmdutils.FindSkillInDatabase(arg0, opt.repo)
		if err != nil {
			tip := "💡 Tip: Use 'skillctl search %s' to search for skills, or provide a Git URL"
			if opt.repo != "" {
				tip = "💡 Tip: Use 'skillctl search -r %s %s' to search in this repository, or provide a Git URL"
				return fmt.Errorf("skill '%s' not found in repository '%s': %w\n\n%s", arg0, opt.repo, err, fmt.Sprintf(tip, opt.repo, arg0))
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

	repoName, err := cmdutils.ExtractRepoName(gitURL)
	if err != nil {
		return fmt.Errorf("failed to extract repository name: %w", err)
	}

	targetSkillName := skillName
	if targetSkillName == "" {
		if dbSkill != nil {
			targetSkillName = dbSkill.Name
		} else {
			targetSkillName = repoName
		}
	}

	existing, err := storage.GetSkill(targetSkillName)
	if err == nil && existing != nil {
		return fmt.Errorf("skill '%s' is already installed (version %s)", targetSkillName, existing.Version)
	}

	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get skills directory: %w", err)
	}
	if err := storage.EnsureSkillsDir(); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	installPath := filepath.Join(skillsDir, targetSkillName)

	useSparseCheckout := dbSkill != nil || skillName != ""
	var targetSubdir string
	if dbSkill != nil {
		targetSubdir = filepath.Join("skills", targetSkillName)
	} else if skillName != "" {
		targetSubdir = filepath.Join("skills", skillName)
	}

	if opt.version != "" {
		tag := fmt.Sprintf("%s@%s", targetSkillName, opt.version)
		fmt.Printf("📥 Installing version %s (tag: %s) to: %s\n", opt.version, tag, installPath)
		if useSparseCheckout {
			if err := cmdutils.CloneRepositoryWithSubdirAtTag(gitURL, installPath, targetSubdir, tag); err != nil {
				return fmt.Errorf("failed to clone version %s: %w", opt.version, err)
			}
		} else {
			if err := cmdutils.CloneRepositoryAtTag(gitURL, installPath, tag); err != nil {
				return fmt.Errorf("failed to clone version %s: %w", opt.version, err)
			}
		}
	} else {
		fmt.Printf("📥 Cloning repository to: %s\n", installPath)
		if useSparseCheckout {
			if err := cmdutils.CloneRepositoryWithSubdir(gitURL, installPath, targetSubdir); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}
		} else {
			if err := cmdutils.CloneRepository(gitURL, installPath); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}
		}
	}
	fmt.Println("✓ Repository cloned successfully")

	fmt.Println("📋 Parsing skill metadata...")
	skillMeta, err := cmdutils.ParseSkillMetadata(installPath, targetSkillName)
	if err != nil {
		fmt.Printf("⚠ Warning: Could not parse skill metadata: %v\n", err)
		skillMeta = storage.SkillMeta{
			Name:        targetSkillName,
			Description: fmt.Sprintf("Skill installed from %s", gitURL),
			InstalledAt: time.Now().Format(time.RFC3339),
		}
	} else {
		fmt.Println("✓ Skill metadata parsed")
	}

	if err := storage.AddSkill(targetSkillName, skillMeta); err != nil {
		return fmt.Errorf("failed to add skill to index: %w", err)
	}

	if err := recordInstallDownload(targetSkillName, skillMeta, opt.repo, gitURL, dbSkill); err != nil {
		fmt.Printf("⚠ Warning: Could not record install history: %v\n", err)
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

func recordInstallDownload(installedAs string, skillMeta storage.SkillMeta, repoFilter, gitURL string, dbSkill *db.Skill) error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	downloader := currentDownloader()
	hostname, _ := os.Hostname()

	sourceType := "git-url"
	sourceName := toNullString("")
	sourceURL := toNullString(gitURL)
	if dbSkill != nil {
		sourceType = "indexed-database"
		if repoFilter != "" {
			sourceType = "indexed-repository"
			sourceName = toNullString(repoFilter)
		} else if dbSkill.RepoName.Valid && dbSkill.RepoName.String != "" {
			sourceName = toNullString(dbSkill.RepoName.String)
		}
		if dbSkill.RepoUrl.Valid && dbSkill.RepoUrl.String != "" {
			sourceURL = toNullString(dbSkill.RepoUrl.String)
		}
	}

	return manager.RecordSkillDownload(context.Background(), db.CreateSkillDownloadParams{
		SkillName:      skillMeta.Name,
		InstalledAs:    installedAs,
		SkillVersion:   toNullString(skillMeta.Version),
		SourceType:     sourceType,
		SourceName:     sourceName,
		SourceUrl:      sourceURL,
		Downloader:     downloader,
		DownloaderHost: toNullString(hostname),
		DownloadedAt:   time.Now().Format(time.RFC3339),
	})
}

func currentDownloader() string {
	if account, err := user.Current(); err == nil && account.Username != "" {
		return account.Username
	}
	if username := os.Getenv("USERNAME"); username != "" {
		return username
	}
	if username := os.Getenv("USER"); username != "" {
		return username
	}
	return "unknown"
}

func toNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
