package cmd

import (
	addcmd "github.com/seekthought/skill/cmd/add"
	installcmd "github.com/seekthought/skill/cmd/install"
	listcmd "github.com/seekthought/skill/cmd/list"
	pushcmd "github.com/seekthought/skill/cmd/push"
	removecmd "github.com/seekthought/skill/cmd/remove"
	searchcmd "github.com/seekthought/skill/cmd/search"
	statuscmd "github.com/seekthought/skill/cmd/status"
	updatecmd "github.com/seekthought/skill/cmd/update"

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
	rootCmd.AddCommand(addcmd.NewCommand())
	rootCmd.AddCommand(installcmd.NewCommand())
	rootCmd.AddCommand(listcmd.NewCommand())
	rootCmd.AddCommand(pushcmd.NewCommand())
	rootCmd.AddCommand(removecmd.NewCommand())
	rootCmd.AddCommand(searchcmd.NewCommand())
	rootCmd.AddCommand(statuscmd.NewCommand())
	rootCmd.AddCommand(updatecmd.NewCommand())
}

// GetRootCommand returns the root command for testing
func GetRootCommand() *cobra.Command {
	return rootCmd
}
