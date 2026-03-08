package main

import (
	"os"

	"github.com/hmans/beans/internal/commands"
)

func main() {
	root := commands.NewRootCmd()
	root.Use = "beans-tui"
	commands.RegisterTuiCmd(root)

	// Default to "tui" when no subcommand is given
	if len(os.Args) < 2 || os.Args[1][0] == '-' {
		os.Args = append([]string{os.Args[0], "tui"}, os.Args[1:]...)
	}

	commands.Execute(root)
}
