package addcmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seekthought/skill/pkg/db"
	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

type options struct {
	targetLocation string
	projectDir     string
	agentSkillsDir string
	fromDir        string
	repoURL        string
	repoName       string
	repoDesc       string
	repoDefault    bool
	repoPriority   int
}

type repositoryOptions struct {
	name        string
	description string
	isDefault   bool
	priority    int
}

func NewCommand() *cobra.Command {
	opt := &options{}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		opt.fromDir = filepath.Join(homeDir, ".skill")
	}

	cmd := &cobra.Command{
		Use:   "add [source]",
		Short: "Add a skill from Git repository, local path, or installed skill name",
		Long: `Download and install a skill from a Git repository, copy from a local path, or add from an installed skill. Can install to ~/.skills or directly to VS Code recognized skills directories for immediate use.

You can also add a remote repository directly:
  - skillctl add -r <repo-url>
  - skillctl add --repo <repo-url> --repo-name <name>`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opt.repoURL != "" {
				repoOpt := &repositoryOptions{
					name:        opt.repoName,
					description: opt.repoDesc,
					isDefault:   opt.repoDefault,
					priority:    opt.repoPriority,
				}
				return runRepositoryAdd(repoOpt, opt.repoURL)
			}
			if len(args) != 1 {
				return cobra.ExactArgs(1)(cmd, args)
			}
			return runAdd(opt, args[0])
		},
	}

	cmd.Flags().StringVarP(&opt.targetLocation, "project", "p", "", "Target location: github, agents, claude, copilot, or custom path")
	cmd.Flags().StringVarP(&opt.projectDir, "dir", "d", "", "Project directory (default: current directory)")
	cmd.Flags().StringVarP(&opt.agentSkillsDir, "skills-dir", "s", "examples/skills", "Agent skills directory relative to project")
	cmd.Flags().StringVar(&opt.fromDir, "from", opt.fromDir, "Directory to search for skills (default: ~/.skill)")
	cmd.Flags().StringVarP(&opt.repoURL, "repo", "r", "", "Add a remote repository by URL")
	cmd.Flags().StringVar(&opt.repoName, "repo-name", "", "Repository name when using --repo")
	cmd.Flags().StringVar(&opt.repoDesc, "repo-description", "", "Repository description when using --repo")
	cmd.Flags().BoolVar(&opt.repoDefault, "repo-default", false, "Set repository as default when using --repo")
	cmd.Flags().IntVar(&opt.repoPriority, "repo-priority", 0, "Repository priority when using --repo")
	return cmd
}

func runAdd(opt *options, source string) error {
	fmt.Printf("📦 Adding skill from: %s\n", source)
	fmt.Println()

	skillName, sourcePath, isInstalledSkill, err := resolveSource(opt, source)
	if err != nil {
		return err
	}

	var repoName string
	if isInstalledSkill {
		repoName = skillName
		fmt.Printf("📁 Using installed skill: %s\n", skillName)
		fmt.Printf("   Source path: %s\n", sourcePath)
	} else if sourcePath != "" {
		repoName = filepath.Base(sourcePath)
		fmt.Printf("📁 Using local path: %s\n", sourcePath)
	} else {
		repoName = skillName
		fmt.Printf("📁 Using Git URL: %s\n", source)
	}

	if opt.targetLocation == "" {
		return fmt.Errorf("please specify a target location using --project/-p")
	}

	installPath, err := cmdutils.GetVSCodeInstallPath(opt.targetLocation, opt.projectDir, repoName)
	if err != nil {
		return err
	}
	fmt.Printf("📁 Installing to: %s\n", installPath)

	if _, err := os.Stat(installPath); err == nil {
		return fmt.Errorf("⚠️ skill already exists at: %s", installPath)
	}

	if sourcePath != "" {
		fmt.Println("📥 Copying skill...")
		if err := cmdutils.CopyDirectory(sourcePath, installPath); err != nil {
			return fmt.Errorf("failed to copy skill: %w", err)
		}
		fmt.Println("✓ Skill copied successfully")
	} else {
		fmt.Println("📥 Cloning repository...")
		if err := cmdutils.CloneRepository(source, installPath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Println("✓ Repository cloned successfully")
	}

	fmt.Println("📋 Parsing skill metadata...")
	skillMeta, err := cmdutils.ParseSkillMetadata(installPath, repoName)
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

	if err := cmdutils.RefreshVSCode(); err != nil {
		fmt.Printf("⚠ Warning: Could not refresh VS Code: %v\n", err)
	} else {
		fmt.Println("✓ VS Code has been refreshed")
	}

	return nil
}

func resolveSource(opt *options, source string) (skillName string, sourcePath string, isInstalledSkill bool, err error) {
	if opt.fromDir != "" {
		possiblePath := filepath.Join(opt.fromDir, source)
		if _, err := os.Stat(possiblePath); err == nil {
			return source, possiblePath, true, nil
		}
	}

	skillMeta, err := storage.GetSkill(source)
	if err == nil && skillMeta != nil {
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

	if cmdutils.IsLocalPath(source) {
		path := filepath.Clean(source)
		if !filepath.IsAbs(path) {
			path, err = filepath.Abs(path)
			if err != nil {
				return "", "", false, fmt.Errorf("failed to get absolute path: %w", err)
			}
		}
		if _, err := os.Stat(path); err != nil {
			return "", "", false, fmt.Errorf("source path does not exist: %s", source)
		}
		return filepath.Base(path), path, false, nil
	}

	if cmdutils.IsGitURL(source) {
		repoName, err := cmdutils.ExtractRepoName(source)
		if err != nil {
			return "", "", false, fmt.Errorf("failed to extract repository name: %w", err)
		}
		return repoName, "", false, nil
	}

	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return "", "", false, fmt.Errorf("failed to get skills directory: %w", err)
	}
	possiblePath := filepath.Join(skillsDir, source)
	if _, err := os.Stat(possiblePath); err == nil {
		return source, possiblePath, true, nil
	}

	projectDirs := append([]string{".skill"}, cmdutils.ProjectSkillSubdirs...)
	for _, projectDir := range projectDirs {
		possiblePath := filepath.Join(projectDir, source)
		if _, err := os.Stat(possiblePath); err == nil {
			return source, possiblePath, true, nil
		}
	}

	return "", "", false, fmt.Errorf("skill '%s' not found.\n\nSearched in:\n- ~/.skills (installed skills)\n- %s (--from directory)\n- .skill (default skills directory)\n- examples/skills, .github/skills, .agents/skills, .claude/skills, .copilot/skills\n\nPlease:\n- Use 'skillctl list' to see installed skills\n- Use 'skillctl list --skills-dir .skill' to see skills in .skill directory\n- Use 'skillctl add https://github.com/user/skill -l <location>' for Git URLs\n- Use 'skillctl add ./local-path -l <location>' for local paths", source, opt.fromDir)
}

func runRepositoryAdd(opt *repositoryOptions, repoURL string) error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()
	repoName := opt.name
	if repoName == "" {
		repoName, err = cmdutils.ExtractRepoName(repoURL)
		if err != nil {
			return fmt.Errorf("failed to infer repository name: %w", err)
		}
	}

	if existing, err := manager.GetQueries().GetRepositoryByName(ctx, repoName); err == nil {
		return fmt.Errorf("repository '%s' already exists: %s", existing.Name, existing.Url)
	}

	params := db.CreateRepositoryParams{
		Name:        repoName,
		Url:         repoURL,
		Description: sql.NullString{String: opt.description, Valid: opt.description != ""},
		IsDefault:   0,
		Priority:    int64(opt.priority),
	}
	if opt.isDefault {
		params.IsDefault = 1
	}

	if err := manager.CreateRepositoryWithDefaults(ctx, params); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	if opt.isDefault {
		created, err := manager.GetQueries().GetRepositoryByName(ctx, repoName)
		if err != nil {
			return fmt.Errorf("repository created but failed to load for default update: %w", err)
		}
		if err := manager.SetRepositoryAsDefaultByID(ctx, created.ID); err != nil {
			return fmt.Errorf("repository created but failed to mark as default: %w", err)
		}
	}

	fmt.Printf("✅ Repository '%s' added successfully!\n", repoName)
	fmt.Printf("   URL: %s\n", repoURL)
	if opt.description != "" {
		fmt.Printf("   Description: %s\n", opt.description)
	}
	if opt.isDefault {
		fmt.Println("   Default: yes")
	}
	if opt.priority != 0 {
		fmt.Printf("   Priority: %d\n", opt.priority)
	}

	return nil
}
