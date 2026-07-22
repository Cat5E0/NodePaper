package main

import (
	"fmt"
	"io"
	"os"

	"nodepaper/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	invocation, err := cli.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "nodepaper: %v\n\n%s", err, cli.Usage)
		return 2
	}

	if invocation.Version {
		fmt.Fprintf(stdout, "nodepaper %s\n", version)
		return 0
	}
	if invocation.Help {
		fmt.Fprint(stdout, cli.Usage)
		return 0
	}

	fmt.Fprintf(stderr, "nodepaper: %s is not available in this development build\n", invocation.Command)
	return 1
}
