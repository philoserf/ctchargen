// Command ctchargen generates rules-accurate Classic Traveller characters
// from Book 1 pp. 4-25, as held in the FFE reprint of the (c) 1977 text.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

// errUsage reports a command line the tool cannot act on.
var errUsage = errors.New("usage")

func main() {
	err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctchargen:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: ctchargen new|render|version", errUsage)
	}

	switch args[0] {
	case "new":
		return newCharacter(args[1:], out)
	case "version":
		return version(out)
	default:
		return fmt.Errorf("%w: unknown command %q; want new, render or version", errUsage, args[0])
	}
}

func version(out io.Writer) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("%w: no build information is embedded", errUsage)
	}

	_, err := fmt.Fprintf(out, "%s %s\n", info.Main.Path, buildVersion(info))
	if err != nil {
		return fmt.Errorf("writing the version: %w", err)
	}

	return nil
}

func buildVersion(info *debug.BuildInfo) string {
	if info.Main.Version != "" {
		return info.Main.Version
	}

	return "(devel)"
}
