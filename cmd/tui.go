package cmd

import (
	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/tui"
)

var tuiExcludeArchived bool

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive TUI",
	Long:  `Opens an interactive terminal user interface for browsing and managing beans.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Override config if flag was explicitly set
		if cmd.Flags().Changed("exclude-archived") {
			cfg.TUI.ExcludeArchived = tuiExcludeArchived
		}
		return tui.Run(core, cfg)
	},
}

func init() {
	tuiCmd.Flags().BoolVarP(&tuiExcludeArchived, "exclude-archived", "e", false, "Exclude beans with archive statuses (completed, scrapped)")
	rootCmd.AddCommand(tuiCmd)
}
