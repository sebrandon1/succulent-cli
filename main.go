package main

import "github.com/sebrandon1/succulent-cli/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	_ = cmd.Execute()
}
