package main

import "github.com/hmans/beans/internal/commands"

func main() {
	root := commands.NewRootCmd()
	commands.RegisterTuiCmd(root)
	commands.Execute(root)
}
