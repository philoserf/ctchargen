package main

import (
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

const profile = `mode: atomic
github.com/philoserf/ctchargen/dice/dice.go:20.30,22.2 1 1
github.com/philoserf/ctchargen/dice/dice.go:30.40,33.2 3 0
github.com/philoserf/ctchargen/rules/lift.go:10.20,12.2 2 1
`

func TestParseProfileCountsOnlyUnexecutedStatements(t *testing.T) {
	t.Parallel()

	got, err := parseProfile(strings.NewReader(profile))
	if err != nil {
		t.Fatalf("parseProfile: %v", err)
	}

	want := map[string]int{
		"github.com/philoserf/ctchargen/dice":  3,
		"github.com/philoserf/ctchargen/rules": 0,
	}
	if len(got) != len(want) {
		t.Fatalf("packages = %v, want %v", got, want)
	}

	for pkg, n := range want {
		if got[pkg] != n {
			t.Errorf("%s: uncovered = %d, want %d", pkg, got[pkg], n)
		}
	}
}

// A package that is fully covered must still appear, or a later regression
// in it would read as a brand new package rather than as a rise.
func TestParseProfileKeepsFullyCoveredPackages(t *testing.T) {
	t.Parallel()

	got, err := parseProfile(strings.NewReader("mode: set\nx/y/z.go:1.1,2.2 1 1\n"))
	if err != nil {
		t.Fatalf("parseProfile: %v", err)
	}

	if _, ok := got["x/y"]; !ok {
		t.Errorf("fully covered package x/y is absent from %v", got)
	}
}

func TestParseProfileRejectsMalformedLines(t *testing.T) {
	t.Parallel()

	for name, line := range map[string]string{
		"no counts":     "x/y/z.go:1.1,2.2\n",
		"one count":     "x/y/z.go:1.1,2.2 1\n",
		"no location":   "nonsense 1 1\n",
		"bad statement": "x/y/z.go:1.1,2.2 many 1\n",
		"bad count":     "x/y/z.go:1.1,2.2 1 often\n",
	} {
		_, err := parseProfile(strings.NewReader("mode: set\n" + line))
		if err == nil {
			t.Errorf("%s: parseProfile accepted %q", name, line)
		}
	}
}

const (
	wantMissing = "not in the ratchet"
	modeCheck   = "check"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	measured := map[string]int{"a": 3, "b": 0}

	for name, tc := range map[string]struct {
		recorded map[string]int
		wantErr  string
	}{
		"holding":      {map[string]int{"a": 3, "b": 0}, ""},
		"risen":        {map[string]int{"a": 2, "b": 0}, "gained uncovered statements"},
		"missing":      {map[string]int{"a": 3}, wantMissing},
		"stale":        {map[string]int{"a": 3, "b": 0, "c": 1}, "no longer exist"},
		"fallen":       {map[string]int{"a": 4, "b": 0}, "coverage improved"},
		"all at fault": {map[string]int{"a": 2, "c": 1}, wantMissing},
	} {
		err := check(measured, tc.recorded, io.Discard)

		switch {
		case tc.wantErr == "":
			if err != nil {
				t.Errorf("%s: unexpected error %v", name, err)
			}
		case err == nil:
			t.Errorf("%s: want an error mentioning %q, got none", name, tc.wantErr)
		case !strings.Contains(err.Error(), tc.wantErr):
			t.Errorf("%s: error %q does not mention %q", name, err, tc.wantErr)
		}
	}
}

// One run says everything that is wrong, rather than stopping at the first.
func TestCheckReportsEveryFault(t *testing.T) {
	t.Parallel()

	err := check(
		map[string]int{"a": 3, "b": 1},
		map[string]int{"a": 2, "c": 1},
		io.Discard,
	)
	if err == nil {
		t.Fatal("want an error")
	}

	for _, want := range []string{"gained uncovered", wantMissing, "no longer exist"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q:\n%s", want, err)
		}
	}
}

// Each case names what the error must say. Asserting only that some error
// came back is what let the unknown-mode case pass without ever reaching the
// mode check: it failed on the profile path that was never there.
func TestRunRejectsBadArguments(t *testing.T) {
	t.Parallel()

	const wantUsage = "usage:"

	good := []string{modeCheck, "profile", "ratchet"}

	for name, tc := range map[string]struct {
		args    []string
		wantErr string
	}{
		"none":         {nil, wantUsage},
		"too few":      {good[:2], wantUsage},
		"too many":     {append(slices.Clone(good), "extra"), wantUsage},
		"unknown mode": {[]string{"polish", "profile", "ratchet"}, "unknown mode"},
	} {
		err := run(tc.args, io.Discard)
		switch {
		case err == nil:
			t.Errorf("%s: run(%v) accepted the arguments", name, tc.args)
		case !strings.Contains(err.Error(), tc.wantErr):
			t.Errorf("%s: error %q does not mention %q", name, err, tc.wantErr)
		}
	}
}

func TestRunCheckRoundTrip(t *testing.T) {
	t.Parallel()

	dir, profilePath := writeProfile(t)
	ratchetPath := dir + "/coverage.ratchet"

	err := run([]string{"update", profilePath, ratchetPath}, io.Discard)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	err = run([]string{modeCheck, profilePath, ratchetPath}, io.Discard)
	if err != nil {
		t.Fatalf("check after update: %v", err)
	}
}

// The profile and the ratchet are read at different moments. Neither absence
// may pass as an empty reading: an unread profile would name every package
// stale, and an unread ratchet would name every package missing.
func TestRunReportsUnreadableFiles(t *testing.T) {
	t.Parallel()

	dir, profilePath := writeProfile(t)

	err := run([]string{modeCheck, dir + "/absent.out", dir + "/coverage.ratchet"}, io.Discard)
	if err == nil {
		t.Error("check accepted a profile that is not there")
	}

	err = run([]string{modeCheck, profilePath, dir + "/absent.ratchet"}, io.Discard)
	if err == nil {
		t.Error("check accepted a ratchet file that is not there")
	}
}

// writeProfile lays the sample profile down in a temp directory.
//
// inside it; the names are what tell them apart at the call site.
//
//nolint:nonamedreturns // two bare strings, one a directory and one a file
func writeProfile(t *testing.T) (dir, profilePath string) {
	t.Helper()

	dir = t.TempDir()

	profilePath = dir + "/coverage.out"

	err := os.WriteFile(profilePath, []byte(profile), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	return dir, profilePath
}
