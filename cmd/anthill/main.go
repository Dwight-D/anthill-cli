// Command anthill is the entry point for the Anthill backlog and escalation CLI.
package main

import (
	"os"

	"github.com/Dwight-D/anthill-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
