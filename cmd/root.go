package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "skill",
	Short: "Agent Skill Management Tool",
	Long: `Skill is a CLI tool for managing AI agent skills,
similar to git but for skill packages.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// GetRootCommand returns the root command for testing
func GetRootCommand() *cobra.Command {
	return rootCmd
}
