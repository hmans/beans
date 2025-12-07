package cmd

import (
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive TUI",
	Long:  `Opens an interactive terminal user interface for browsing and managing beans.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(store, cfg)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
