package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RegisterCoreCommands adds all core CLI commands to the root command.
func RegisterCoreCommands(root *cobra.Command) {
	RegisterArchiveCmd(root)
	RegisterBlockedCmd(root)
	RegisterCheckCmd(root)
	RegisterCompleteCmd(root)
	RegisterCreateCmd(root)
	RegisterDeleteCmd(root)
	RegisterGraphqlCmd(root)
	RegisterInitCmd(root)
	RegisterListCmd(root)
	RegisterMilestonesCmd(root)
	RegisterNextCmd(root)
	RegisterPrimeCmd(root)
	RegisterProgressCmd(root)
	RegisterReadyCmd(root)
	RegisterRoadmapCmd(root)
	RegisterScrapCmd(root)
	RegisterShowCmd(root)
	RegisterStartCmd(root)
	RegisterUpdateCmd(root)
	RegisterVersionCmd(root)

	// Deprecated placeholders for commands that moved to separate binaries
	registerDeprecatedCmd(root, "serve", "beans-serve")
	registerDeprecatedCmd(root, "tui", "beans-tui")
}

func registerDeprecatedCmd(root *cobra.Command, name, binary string) {
	root.AddCommand(&cobra.Command{
		Use:    name,
		Short:  fmt.Sprintf("(moved to %s)", binary),
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stderr, "The %q command has moved to a separate binary: %s\n", name, binary)
			fmt.Fprintf(os.Stderr, "Please install and use %q instead.\n", binary)
			os.Exit(1)
		},
	})
}
