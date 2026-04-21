package main

import "github.com/devpulse-cli/devpulse/cmd"

func main() {
	cmd.RegisterCustomCommands()
	cmd.Execute()
}
