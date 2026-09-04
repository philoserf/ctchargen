package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// commandList is the subcommands in the order run's switch takes them. Both
// messages that name them read it here, so the two cannot come to disagree
// about the order again.
const commandList = "new|batch|render|version"

// The usage line of each subcommand: what it takes that is not a flag.
//
// None of them enumerates its flags. `new` has eleven, and an enumeration
// typed beside a flag set is a line that goes stale the first time a flag is
// added; the list writeFlags prints from the set itself is what carries the
// detail, and it cannot go stale.
// Each is the shape alone: errUsage prints the word "usage" ahead of an
// error, and writeHelp prints "usage: ctchargen " ahead of the help.
const (
	newUsage    = "new [flags]"
	batchUsage  = "batch --auto --count N [flags]"
	renderUsage = "render [flags] <character.json>"
)

// topLevelUsage answers `ctchargen --help`: what the commands are, and where
// to ask about one of them.
const topLevelUsage = "usage: ctchargen <command> [flags]\n" +
	"\n" +
	"  new      generate one character, asking at every choice unless --auto\n" +
	"  batch    generate many characters from one base seed, under --auto\n" +
	"  render   write a record saved earlier as a sheet or as a transcript\n" +
	"  version  write the build\n" +
	"\n" +
	"Run `ctchargen <command> --help` for that command's flags.\n"

// The three strategy flags, named once. A flag's name on the command line is
// also its key in chargen.Strategies, and that is not a coincidence to be
// retyped in four places.
const (
	careerFlag = "career"
	skillsFlag = "skills"
	musterFlag = "muster"
)

// seedFlag is named once too, because `new` and `batch` each define it and
// then ask the set whether it was given.
const seedFlag = "seed"

// shortFlag is the length of a flag name written with one dash. There is one
// of them, `-o`, and it is written that way everywhere else.
const shortFlag = 1

// helpWidth is the column the descriptions start in, wide enough for the
// longest flag this command line has: `  --service string`.
const helpWidth = 19

// writeTopLevelHelp answers --help before a subcommand is named.
func writeTopLevelHelp(out io.Writer) error {
	_, err := io.WriteString(out, topLevelUsage)
	if err != nil {
		return fmt.Errorf("writing the usage: %w", err)
	}

	return nil
}

// writeHelp answers --help on a subcommand: its usage line, then its flags.
//
// It writes to out and returns nil, so that help asked for leaves by the
// same door as the record and exits 0 - not to `asking`, which two of the
// three subcommands are never handed, and which would put `new --help | less`
// on the wrong stream. It cannot reach a `-o` file: parsing happens before
// openDestination, and a run that comes here never opens one.
func writeHelp(out io.Writer, line string, flags *flag.FlagSet) error {
	_, err := fmt.Fprintf(out, "usage: ctchargen %s\n\n", line)
	if err != nil {
		return fmt.Errorf("writing the usage: %w", err)
	}

	return writeFlags(out, flags)
}

// writeFlags lists a command's flags, with their descriptions and defaults.
//
// It exists rather than flag.PrintDefaults because PrintDefaults writes one
// dash and everything else here writes two: the README, the error messages,
// and the referee's own transcript. The defaults are printed because a
// strategy flag left alone still chooses, and the reader who could not see
// which one it chose is the one who reported this.
func writeFlags(out io.Writer, flags *flag.FlagSet) error {
	var failed error

	flags.VisitAll(func(each *flag.Flag) {
		if failed != nil {
			return
		}

		kind, description := flag.UnquoteUsage(each)

		named := "  " + dashed(each.Name)
		if kind != "" {
			named += " " + kind
		}

		if !zeroDefault(each.DefValue) {
			description += " (default " + each.DefValue + ")"
		}

		_, failed = fmt.Fprintf(out, "%-*s %s\n", helpWidth, named, description)
	})

	if failed != nil {
		return fmt.Errorf("writing the flags: %w", failed)
	}

	return nil
}

// dashed writes a flag the way this command line writes it: -o, --seed. The
// flag package accepts either spelling for either name; this is about what
// the reader is shown.
func dashed(name string) string {
	if len(name) == shortFlag {
		return "-" + name
	}

	return "--" + name
}

// zeroDefault reports a default not worth printing. The flag package's own
// isZeroValue is unexported; these three cover every flag this command line
// has, and a flag whose default is worth reading is never one of them.
func zeroDefault(value string) bool {
	return value == "" || value == "0" || value == "false"
}

// strategyChoices is a strategy flag's own description, read from the table
// the engine validates against rather than repeated here.
//
// A strategy added to POLICY.md and to chargen.Strategies is offered by the
// help without anyone remembering to come back for it. It is also where a
// `go install` user learns the values at all: the rejection message names
// POLICY.md, which is a repo file he does not have.
func strategyChoices(name string) string {
	return strings.Join(chargen.Strategies[name], ", ")
}

// serviceChoices is the --service flag's own description: the six of p. 10,
// in the book's order, spelled the way they are typed.
func serviceChoices() string {
	names := make([]string, 0, len(traveller.ServiceNames))

	for _, service := range traveller.ServiceNames {
		names = append(names, strings.ToLower(service.String()))
	}

	return strings.Join(names, ", ")
}
