package statuscmd

import (
	"fmt"
	"time"

	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "List all installed skills",
		Long:  `Display a list of all skills currently installed on your local machine.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}

	cmd.AddCommand(newDownloadsCommand())
	return cmd
}

func runStatus() error {
	fmt.Println("Installed Skills:")
	fmt.Println()

	skills, err := storage.ListSkills()
	if err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills are currently installed.")
		fmt.Println()
		fmt.Println("Use 'skillctl add <skill-name>' to install a skill.")
		return nil
	}

	for i, skill := range skills {
		fmt.Printf("%d. %s\n", i+1, skill.Name)
		fmt.Printf("   Version: %s\n", skill.Version)
		fmt.Printf("   Description: %s\n", skill.Description)
		fmt.Printf("   Author: %s\n", skill.Author)
		fmt.Printf("   Tags: %s\n", cmdutils.FormatTags(skill.Tags))

		if skill.InstalledAt != "" {
			installedTime, err := time.Parse(time.RFC3339, skill.InstalledAt)
			if err == nil {
				fmt.Printf("   Installed: %s\n", installedTime.Format("2006-01-02 15:04:05"))
			}
		}

		fmt.Println()
	}

	fmt.Printf("Total: %d skill(s) installed\n", len(skills))
	return nil
}
