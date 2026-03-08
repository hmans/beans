package main

import "github.com/hmans/beans/internal/commands"

func main() {
	root := commands.NewRootCmd()
	commands.RegisterServeCmd(root)
	commands.Execute(root)
}
