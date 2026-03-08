package main

import "github.com/hmans/beans/internal/commands"

func main() {
	root := commands.NewRootCmd()
	commands.RegisterCoreCommands(root)
	commands.Execute(root)
}
