package main

import (
	"io"
	"os"
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
		if _, err := parseProfile(strings.NewReader("mode: set\n" + line)); err == nil {
			t.Errorf("%s: parseProfile accepted %q", name, line)
		}
	}
}

const wantMissing = "not in the ratchet"

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
		case tc.wantErr == "" && err != nil:
			t.Errorf("%s: unexpected error %v", name, err)
		case tc.wantErr != "" && err == nil:
			t.Errorf("%s: want an error mentioning %q, got none", name, tc.wantErr)
		case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
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

func TestRunRejectsBadArguments(t *testing.T) {
	t.Parallel()

	for name, args := range map[string][]string{
		"none":         {},
		"too few":      {"check", "profile"},
		"unknown mode": {"polish", "profile", "ratchet"},
	} {
		if err := run(args, io.Discard); err == nil {
			t.Errorf("%s: run(%v) accepted the arguments", name, args)
		}
	}
}

func TestRunCheckRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profilePath := dir + "/coverage.out"
	ratchetPath := dir + "/coverage.ratchet"

	if err := os.WriteFile(profilePath, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"update", profilePath, ratchetPath}, io.Discard); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := run([]string{"check", profilePath, ratchetPath}, io.Discard); err != nil {
		t.Fatalf("check after update: %v", err)
	}
}
