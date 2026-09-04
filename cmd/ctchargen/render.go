package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/philoserf/ctchargen/render"
)

// renderRecord reads a record written earlier and writes it as a character
// sheet, or as the generation transcript with --history.
//
// It reads the record rather than regenerating it. Regenerating from the
// recorded seed would produce the same character within this build and a
// different one across builds, which is precisely the promise the PRD does
// not make; reading is what lets a record written by another build still be
// shown.
func renderRecord(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		history = flags.Bool("history", false, "write the generation record rather than the sheet")
		output  = flags.String("o", "", "write to this file rather than to standard output")
		force   = flags.Bool("force", false, "replace the output file if it already exists")
	)

	err := flags.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, renderUsage, flags)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen render --help`", errUsage, err)
	}

	// The flag package stops at the first non-flag argument, and the PRD's
	// CLI sketch says so plainly: flags precede the filename. Someone who
	// puts them after gets more than one positional argument, and deserves
	// to be told which of the two mistakes he made.
	if flags.NArg() > 1 {
		return fmt.Errorf("%w: flags precede the filename; %s", errUsage, renderUsage)
	}

	if flags.NArg() == 0 {
		return fmt.Errorf("%w: %s", errUsage, renderUsage)
	}

	path := flags.Arg(0)

	text, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	rendered, err := renderText(text, *history)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	where, err := openDestination(out, *output, *force)
	if err != nil {
		return err
	}

	return where.write(rendered)
}

func renderText(text []byte, history bool) (string, error) {
	if history {
		rendered, err := render.TranscriptFrom(text)

		return rendered, wrapRender(err)
	}

	rendered, err := render.SheetFrom(text)

	return rendered, wrapRender(err)
}
