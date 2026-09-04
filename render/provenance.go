package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// The flags the reproduce command spells, and the two renderings it can ask
// for.
//
// This is the one place the render package knows how the command line is
// written. Nothing here is held by the compiler; what holds it is the
// round-trip test in cmd/ctchargen, which takes the line this file prints and
// runs it, so a flag misspelled here is a flag the tool rejects there.
const (
	autoFlag    = "--auto"
	seedFlag    = "--seed"
	serviceFlag = "--service"
	nameFlag    = "--name"
	careerFlag  = "--career"
	skillsFlag  = "--skills"
	musterFlag  = "--muster"

	sheetRendering   = "--sheet"
	historyRendering = "--history"
)

// provenance is how a reader gets this character back, or why he cannot.
//
// A sheet is what a referee reads and what he keeps. The seed reached the
// JSON record and the transcript's opening line and never reached here, so a
// sheet spun, read and liked was a character discarded - which is the whole
// of the finding this answers.
//
// What it prints is the command rather than the seed, because running it is
// what the reader wants to do with it. The strategies are named even where
// they are the defaults: a default that moves under a reader is exactly how
// a published page came to document flags that no longer work.
func provenance(r record, rendering string) (string, error) {
	byPlayer, err := decidedByPlayer(r)
	if err != nil {
		return "", err
	}

	if byPlayer {
		return answeredByThePlayer(r), nil
	}

	return regenerateWith(r, rendering), nil
}

// regenerateWith is the line for a character the policy decided, which the
// seed does bring back.
func regenerateWith(r record, rendering string) string {
	line := "Regenerate with `" + strings.Join(command(r, rendering), " ") + "`"

	// A record generated in-process carries no build - stamp fills it only
	// from the command - and a build nobody recorded is not one to warn
	// about.
	if r.Build == "" {
		return line + "."
	}

	return fmt.Sprintf(
		"%s, on %s — the same seed on a different build is a different character.",
		line, r.Build)
}

// command is the argument list that reproduces this character.
func command(r record, rendering string) []string {
	args := []string{
		"ctchargen", "new", autoFlag, seedFlag, strconv.FormatUint(r.Inputs.Seed, 10),
	}

	// The service asked for, which is what reproduces the run - not the
	// service the character ended up in, which the draft may have decided.
	if r.Inputs.Service != "" {
		args = append(args, serviceFlag, strings.ToLower(r.Inputs.Service))
	}

	// Quoted, always. It is the one value that is not a bare token, and this
	// line is meant to be pasted into a shell before it is anything else.
	if r.Inputs.Name != "" {
		args = append(args, nameFlag, strconv.Quote(r.Inputs.Name))
	}

	return append(args,
		careerFlag, r.Inputs.Career,
		skillsFlag, r.Inputs.Skills,
		musterFlag, r.Inputs.Muster,
		rendering)
}

// answeredByThePlayer is the line for a character the seed does not bring
// back, which says so rather than offering a command that would not work.
func answeredByThePlayer(r record) string {
	var out strings.Builder

	fmt.Fprintf(&out, "Seed %d", r.Inputs.Seed)

	if r.Inputs.Service != "" {
		fmt.Fprintf(&out, ", service %s", strings.ToLower(r.Inputs.Service))
	}

	fmt.Fprintf(&out,
		", strategies %s/%s/%s. The choices were the %s's rather than the %s's,"+
			" so the seed alone does not bring this character back — keep the"+
			" JSON record.",
		r.Inputs.Career, r.Inputs.Skills, r.Inputs.Muster,
		traveller.ByPlayer, traveller.ByPolicy)

	return out.String()
}

// choices reads the record's choice events, which are the only place it says
// who decided anything.
//
// Two readers want them - whether the seed reproduces the character, and who
// named the service - so the walk is here once rather than in each. A record
// whose events will not read then fails the same way for both.
func choices(r record) ([]eventJSON, error) {
	var found []eventJSON

	for _, raw := range r.Events {
		var event eventJSON

		err := unmarshalEvent(raw, &event)
		if err != nil {
			return nil, err
		}

		if event.Kind == kindChoice {
			found = append(found, event)
		}
	}

	return found, nil
}

// decidedByPlayer reports whether any choice was answered at the keyboard.
//
// The question is about choices actually made, not about the mode the run was
// started in - which the record does not carry and does not need to. A record
// with no choice events at all is reproducible from its seed whoever was
// sitting there, and this gets that right by asking nothing else.
func decidedByPlayer(r record) (bool, error) {
	made, err := choices(r)
	if err != nil {
		return false, err
	}

	for _, choice := range made {
		if choice.By == traveller.ByPlayer.String() {
			return true, nil
		}
	}

	return false, nil
}

// choiceAt returns a named choice point's event, and whether it was put at
// all. A point nobody was asked is not the same as one the policy took.
//
// It hands back the whole event rather than who answered, because both halves
// are wanted together: who chose, and what they chose. Asking only who leads
// to crediting a decider with an outcome he did not pick.
func choiceAt(r record, point string) (eventJSON, bool, error) {
	made, err := choices(r)
	if err != nil {
		return eventJSON{}, false, err
	}

	for _, choice := range made {
		if choice.Point == point {
			return choice, true, nil
		}
	}

	return eventJSON{}, false, nil
}
