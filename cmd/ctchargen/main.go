// Command ctchargen generates Classic Traveller characters (Books 1-3,
// (c) 1977 text). See docs/PRD.md for the v1 contract.
//
// Implemented subcommands: none yet.
package main

import (
	"fmt"
	"io"
	"os"
)

// Exit codes: 0 success, 1 operational error, 2 usage error (the flag
// package's own convention). exitOK arrives with the first implemented
// subcommand.
const (
	exitError = 1
	exitUsage = 2
)

const usage = `usage:
  ctchargen new [--seed N] [--auto] [--service navy] [--name X] [-o file]
  ctchargen batch --count 20 --auto [--service ...] [-o dir|file.jsonl]
  ctchargen render [--history] character.json
  ctchargen replay [--ignore-provenance] character.json
  ctchargen version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)

		return exitUsage
	}

	switch args[0] {
	case "new", "batch", "render", "replay", "version":
		fmt.Fprintf(stderr, "ctchargen %s: not yet implemented (see docs/PRD.md)\n", args[0])

		return exitError
	default:
		fmt.Fprintf(stderr, "ctchargen: unknown command %q\n", args[0])
		fmt.Fprint(stderr, usage)

		return exitUsage
	}
}
