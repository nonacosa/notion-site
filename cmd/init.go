package cmd

import (
	"github.com/pkwenda/notion-site/generator"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init the config",
	RunE: func(cmd *cobra.Command, args []string) error {
		return generator.DefaultConfigInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
