package pushcmd

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
	remoteURL string
	repoName  string
}

func NewCommand() *cobra.Command {
	opt := &options{}

	cmd := &cobra.Command{
		Use:   "push [skill-name]",
		Short: "Push a skill to local .skills or remote repository",
		Long: `Push a skill from examples/skills to local .skills directory (default) or a remote repository (GitHub/GitLab).

Usage:
  push <name>              - Push to local .skills directory
  push <name> -R <url>     - Push to remote repository with URL
  push <name> -r <repo>    - Push to remote repository by repo name (from database)

By default, pushes to local ~/.skills directory and updates the index.
Use -R to specify a remote repository URL directly.
Use -r to specify a repository name that exists in the database.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(opt, args[0])
		},
	}

	cmd.Flags().StringVarP(&opt.remoteURL, "remote-url", "R", "", "Push to remote repository with URL")
	cmd.Flags().StringVarP(&opt.repoName, "remote-repo", "r", "", "Push to remote repository by name (from database)")

	return cmd
}

func runPush(opt *options, skillName string) error {
	if opt.remoteURL != "" && opt.repoName != "" {
		return fmt.Errorf("cannot use both -R (remote-url) and -r (remote-repo) flags together")
	}

	var remoteURL string
	var pushMode string
	if opt.remoteURL != "" {
		remoteURL = opt.remoteURL
		pushMode = "remote-url"
	} else if opt.repoName != "" {
		repo, err := cmdutils.GetRepositoryByName(opt.repoName)
		if err != nil {
			return fmt.Errorf("failed to get repository '%s' from database: %w", opt.repoName, err)
		}
		remoteURL = repo.Url
		pushMode = "remote-repo"
		fmt.Printf("📂 Repository '%s' resolved to URL: %s\n", opt.repoName, remoteURL)
	} else {
		pushMode = "local"
	}

	searchDirs := []string{filepath.Join(".github", "skills"), filepath.Join(".agents", "skills"), filepath.Join(".copilot", "skills"), filepath.Join("examples", "skills")}
	var skillPath string
	var foundDir string
	for _, dir := range searchDirs {
		possiblePath := filepath.Join(dir, skillName)
		if _, err := os.Stat(possiblePath); err == nil {
			skillPath = possiblePath
			foundDir = dir
			break
		}
	}

	if skillPath == "" {
		var allAvailableSkills []string
		for _, dir := range searchDirs {
			if entries, err := os.ReadDir(dir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						allAvailableSkills = append(allAvailableSkills, entry.Name())
					}
				}
			}
		}
		fmt.Printf("Skill '%s' does not exist.\n\n", skillName)
		if len(allAvailableSkills) > 0 {
			fmt.Println("Available skills in project:")
			for _, skill := range allAvailableSkills {
				fmt.Printf("  - %s\n", skill)
			}
		} else {
			fmt.Println("No skills found in project directories (.github/skills, .agents/skills, .copilot/skills, examples/skills)")
		}
		return fmt.Errorf("skill not found: %s", skillName)
	}

	fmt.Printf("📦 Pushing skill: %s\n", skillName)
	fmt.Printf("From: %s\n", skillPath)
	fmt.Printf("Source directory: %s\n", foundDir)
	fmt.Printf("Push mode: %s\n", pushMode)
	fmt.Println()

	fmt.Println("📤 Validating skill package...")
	if err := validateSkillPackage(skillPath); err != nil {
		return fmt.Errorf("skill validation failed: %w", err)
	}
	fmt.Println("✓ Skill structure is valid")

	switch pushMode {
	case "local":
		return pushLocal(skillName, skillPath)
	case "remote-url", "remote-repo":
		return pushRemote(skillName, skillPath, remoteURL)
	default:
		return fmt.Errorf("unknown push mode: %s", pushMode)
	}
}

func pushLocal(skillName, skillPath string) error {
	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get skills directory: %w", err)
	}
	localSkillDir := filepath.Join(skillsDir, skillName)
	fmt.Printf("📁 Target directory: %s\n", localSkillDir)
	fmt.Println()
	if _, err := os.Stat(localSkillDir); err == nil {
		fmt.Printf("⚠️  Warning: Skill '%s' already exists in ~/.skills\n", skillName)
		fmt.Printf("Overwriting existing skill...\n")
	}
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	fmt.Println("📤 Copying files to local .skills folder...")
	fileCount := 0
	err = filepath.Walk(skillPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(skillPath, filePath)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(localSkillDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		if err := cmdutils.CopyFile(filePath, targetPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", relPath, err)
		}
		fileCount++
		fmt.Printf("  ✓ %s\n", relPath)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Successfully copied %d file(s)\n", fileCount)
	fmt.Println("📋 Parsing skill metadata...")
	skillMeta, err := cmdutils.ParseSkillMetadata(localSkillDir, skillName)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not parse skill metadata: %v\n", err)
		skillMeta = storage.SkillMeta{Name: skillName, Description: fmt.Sprintf("Skill from examples/skills/%s", skillName), InstalledAt: time.Now().Format(time.RFC3339)}
	} else {
		fmt.Println("✓ Skill metadata parsed")
	}
	fmt.Println("📋 Updating skill index...")
	if err := storage.AddSkill(skillName, skillMeta); err != nil {
		return fmt.Errorf("failed to add skill to index: %w", err)
	}
	fmt.Println("✓ Index updated")
	fmt.Println()
	fmt.Printf("✅ Skill '%s' has been successfully pushed to local .skills folder!\n", skillName)
	fmt.Printf("Location: %s\n", localSkillDir)
	fmt.Println()
	fmt.Println("💡 The skill is now installed and available for use.")
	return nil
}
