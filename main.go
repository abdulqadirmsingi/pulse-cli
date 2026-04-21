package main

import "github.com/abdulqadirmsingi/pulse-cli/cmd"

func main() {
	cmd.RegisterCustomCommands()
	cmd.Execute()
}
