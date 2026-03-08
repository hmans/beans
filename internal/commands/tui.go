package commands

import (
	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive TUI",
	Long:  `Opens an interactive terminal user interface for browsing and managing beans.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(core, cfg)
	},
}

func RegisterTuiCmd(root *cobra.Command) {
	root.AddCommand(tuiCmd)
}
