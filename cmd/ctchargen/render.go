package main

import (
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
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if flags.NArg() != 1 {
		return fmt.Errorf("%w: render [--history] [-o file] [--force] <character.json>", errUsage)
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
