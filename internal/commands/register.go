package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RegisterCoreCommands adds all core CLI commands to the root command.
func RegisterCoreCommands(root *cobra.Command) {
	RegisterArchiveCmd(root)
	RegisterCheckCmd(root)
	RegisterCreateCmd(root)
	RegisterDeleteCmd(root)
	RegisterGraphqlCmd(root)
	RegisterInitCmd(root)
	RegisterListCmd(root)
	RegisterPrimeCmd(root)
	RegisterRoadmapCmd(root)
	RegisterShowCmd(root)
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
