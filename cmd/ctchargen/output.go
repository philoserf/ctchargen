package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// errWouldOverwrite reports an output path that already holds something.
var errWouldOverwrite = errors.New("already exists")

// recordFileMode is the mode a written record carries. A character sheet is
// the operator's own, and nothing else needs to read it.
const recordFileMode = 0o600

// destination is where a subcommand writes: the stream it was handed when no
// path was given, or a file it opens.
//
// A path that exists is never written over without --force. That refusal is
// the whole reason the flag exists: a referee who regenerates a character
// into the file holding last week's has lost last week's, and the tool that
// did it said nothing.
type destination struct {
	out   io.Writer
	close func() error
}

func openDestination(out io.Writer, path string, force bool) (destination, error) {
	if path == "" {
		return destination{out: out, close: func() error { return nil }}, nil
	}

	file, err := create(path, force)
	if err != nil {
		return destination{}, err
	}

	return destination{out: file, close: file.Close}, nil
}

// create opens a file for writing, refusing to replace one unless told to.
func create(path string, force bool) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(path, flags, recordFileMode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("%w: %s %w; pass --force to replace it",
				errUsage, path, errWouldOverwrite)
		}

		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	return file, nil
}

// isDirectory reports whether a path names a directory that already exists,
// which is what tells `-o dir` from `-o file.jsonl`.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// write writes text to a destination and closes it.
func (d destination) write(text string) error {
	_, err := io.WriteString(d.out, text)
	if err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return d.done()
}

func (d destination) done() error {
	err := d.close()
	if err != nil {
		return fmt.Errorf("closing the output: %w", err)
	}

	return nil
}

// memberPath is where batch writes one character when -o names a directory.
func memberPath(dir string, seed uint64) string {
	return filepath.Join(dir, fmt.Sprintf("%020d.json", seed))
}
