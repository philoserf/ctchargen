// Command ctchargen generates rules-accurate Classic Traveller characters
// from Book 1 pp. 4-25, as held in the FFE reprint of the (c) 1977 text.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/philoserf/ctchargen/chargen"
)

// errUsage reports a command line the tool cannot act on.
var errUsage = errors.New("usage")

func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctchargen:", err)
		os.Exit(1)
	}
}

// run takes the two output channels apart. `out` is the data channel - the
// record, the sheet, the transcript - and `asking` is where the tool talks to
// the person driving it. Interactive mode writes its questions to the second,
// so that `ctchargen new --seed 7 | jq` pipes a record and not a conversation.
func run(args []string, in io.Reader, out, asking io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: ctchargen new|batch|render|version", errUsage)
	}

	switch args[0] {
	case "new":
		return newCharacter(args[1:], in, out, asking)
	case "batch":
		return batch(args[1:], out)
	case "render":
		return renderRecord(args[1:], out)
	case "version":
		return version(out)
	default:
		return fmt.Errorf("%w: unknown command %q; want new, render, batch or version", errUsage, args[0])
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

// stamp names the build that wrote a record.
//
// The command is what fills it, and only the command: a record generated
// in-process by a test carries none, because a test binary's build info is
// not the build of a released tool and would make every golden differ by
// machine.
func stamp(character *chargen.Character) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	character.Build = info.Main.Path + " " + buildVersion(info)
}

func buildVersion(info *debug.BuildInfo) string {
	if info.Main.Version != "" {
		return info.Main.Version
	}

	return "(devel)"
}
