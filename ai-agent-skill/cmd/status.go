package cmd

import (
	"fmt"
	"time"

	"skillctl/pkg/storage"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List all installed skills",
	Long:  `Display a list of all skills currently installed on your local machine.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return doStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func doStatus() error {
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

	// Display information for each skill
	for i, skill := range skills {
		fmt.Printf("%d. %s\n", i+1, skill.Name)
		fmt.Printf("   Version: %s\n", skill.Version)
		fmt.Printf("   Description: %s\n", skill.Description)
		fmt.Printf("   Author: %s\n", skill.Author)
		fmt.Printf("   Tags: %s\n", formatTags(skill.Tags))

		// Parse and display installation time
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

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "none"
	}
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ", "
		}
		result += tag
	}
	return result
}
