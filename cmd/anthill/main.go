// Command anthill is the entry point for the Anthill backlog and escalation CLI.
package main

import (
	"fmt"
	"os"

	"github.com/Dwight-D/anthill-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
