// Package main is the entry point for the messenger CLI.
package main

import (
	"os"

	"nit/cmd/messenger/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
