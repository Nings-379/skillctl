package removecmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seekthought/skill/pkg/db"
	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

type options struct {
	project    bool
	projectDir string
	repo       string
}

func NewCommand() *cobra.Command {
	opt := &options{}

	cmd := &cobra.Command{
		Use:   "remove [skill-name]",
		Short: "Remove a skill from the local repository or project",
		Long:  `Uninstall and remove a skill from your local machine or project directory.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opt.repo != "" {
				return runRepositoryRemove(opt.repo)
			}
			if len(args) != 1 {
				return cobra.ExactArgs(1)(cmd, args)
			}
			return runRemove(opt, args[0])
		},
	}

	cmd.Flags().BoolVarP(&opt.project, "project", "p", false, "Remove skill from project directory")
	cmd.Flags().StringVarP(&opt.projectDir, "dir", "d", "", "Project directory (default: current directory)")
	cmd.Flags().StringVarP(&opt.repo, "repo", "r", "", "Remove a remote repository by name")
	return cmd
}

func runRemove(opt *options, skillName string) error {
	var skillPath string
	var skillMeta *storage.SkillMeta

	if opt.project {
		baseDir, err := cmdutils.ResolveProjectDir(opt.projectDir)
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		skillPath, err = cmdutils.FindSkillInProject(baseDir, skillName)
		if err != nil {
			return err
		}

		parsedMeta, parseErr := cmdutils.ParseCustomSkill(skillPath, skillName)
		if parseErr == nil {
			skillMeta = &storage.SkillMeta{
				Name:        parsedMeta.Name,
				Description: parsedMeta.Description,
				Version:     parsedMeta.Version,
				Tags:        parsedMeta.Tags,
				Author:      parsedMeta.Author,
			}
		}

		fmt.Printf("🗑️  Removing skill: %s\n\n", skillName)
		fmt.Printf("   Type: Project skill\n")
		fmt.Printf("   Project: %s\n", baseDir)
		if err := printRemoveDetails(skillMeta, skillPath); err != nil {
			return err
		}
	} else {
		savedMeta, err := storage.GetSkill(skillName)
		if err != nil {
			return fmt.Errorf("skill '%s' is not installed", skillName)
		}
		skillMeta = savedMeta

		skillsDir, err := storage.GetSkillsDir()
		if err != nil {
			return fmt.Errorf("failed to get skills directory: %w", err)
		}
		skillPath = filepath.Join(skillsDir, skillName)

		fmt.Printf("🗑️  Removing skill: %s\n\n", skillName)
		fmt.Printf("   Type: Installed skill\n")
		if err := printRemoveDetails(skillMeta, skillPath); err != nil {
			return err
		}
	}

	fmt.Printf("Are you sure you want to remove '%s'? [y/N]: ", skillName)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("❌ Removal cancelled.")
		return nil
	}

	fmt.Println()
	fmt.Println("🗑️  Removing skill files...")
	if err := os.RemoveAll(skillPath); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}
	fmt.Println("✓ Files removed")

	if !opt.project {
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

func printRemoveDetails(skillMeta *storage.SkillMeta, skillPath string) error {
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
	fmt.Printf("   Location: %s\n\n", skillPath)
	return nil
}

func runRepositoryRemove(name string) error {
	manager, err := cmdutils.InitDB()
	if err != nil {
		return err
	}
	defer cmdutils.CloseDB(manager)

	ctx := context.Background()
	repo, err := manager.GetQueries().GetRepositoryByName(ctx, name)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", name)
	}

	skills, err := manager.GetQueries().ListSkillsByRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to query repository skills: %w", err)
	}

	if err := manager.Transaction(ctx, func(q *db.Queries) error {
		for _, skill := range skills {
			if err := q.DeleteTagsBySkillID(ctx, skill.ID); err != nil {
				return err
			}
		}
		if err := q.DeleteSkillsByRepository(ctx, repo.ID); err != nil {
			return err
		}
		return q.DeleteRepository(ctx, repo.ID)
	}); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	fmt.Printf("✅ Repository '%s' removed successfully!\n", name)
	if len(skills) > 0 {
		fmt.Printf("   Removed indexed skills: %d\n", len(skills))
	}
	return nil
}
