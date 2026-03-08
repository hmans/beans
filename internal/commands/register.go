package commands

import "github.com/spf13/cobra"

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
}
