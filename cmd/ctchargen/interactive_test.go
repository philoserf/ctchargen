package main

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// The prompt loop, driven from a scripted reader so the whole of it is
// covered without a terminal.

// answers supplies the same answer as often as it is asked for, which walks
// a character by always taking the first option offered.
func answers(choice string, times int) io.Reader {
	return strings.NewReader(strings.Repeat(choice+"\n", times))
}

// A bare new walks the procedure, asking at every choice point and showing
// what happened between the questions.
func TestInteractiveWalksACharacter(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{cmdNew, flagSeed, "7", flagService, other, flagSheet},
		answers("1", 200), &out, &out)
	if err != nil {
		t.Fatalf("walking a character: %v", err)
	}

	shown := out.String()

	for _, want := range []string{
		// The procedure, shown as it runs.
		"## characteristics (p. 4)",
		"Strength: rolled",
		"## term 1 (pp. 5-7)",
		"survival: rolled",
		// A question, with the alternatives the engine offered.
		"Which skills table? It is designated before the die (p. 11)",
		"1) Personal Development Table",
		// And the sheet at the end.
		"UPP ",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the run never showed %q", want)
		}
	}

	// The choices are not echoed back: the player answered them a line
	// earlier, and repeating them pushes the useful lines off the screen.
	if strings.Contains(shown, "player chose") {
		t.Error("the run echoed the player's own choices back at him")
	}
}

// An answer it cannot read is asked again rather than guessed at.
func TestInteractiveReAsksWhatItCannotRead(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	// The first three answers to the first question are unusable: a word, a
	// number below the range, and one above it.
	script := strings.NewReader("banana\n0\n99\n" + strings.Repeat("1\n", 200))

	err := run([]string{cmdNew, flagSeed, "7", flagService, other, flagSheet}, script, &out, &out)
	if err != nil {
		t.Fatalf("walking a character: %v", err)
	}

	shown := out.String()

	if strings.Count(shown, "answer with a number from 1 to") != 3 {
		t.Errorf("three unusable answers drew %d re-prompts, want 3",
			strings.Count(shown, "answer with a number from 1 to"))
	}

	if !strings.Contains(shown, "UPP ") {
		t.Error("the run did not finish after the answers became usable")
	}
}

// The input ending is an error, not a default. A character finished by a
// decision nobody made is not the character anybody asked for.
func TestInteractiveRefusesToInventAnAnswer(t *testing.T) {
	t.Parallel()

	for name, script := range map[string]io.Reader{
		"nothing at all":  strings.NewReader(""),
		"nil reader":      nil,
		"answers run out": answers("1", 2),
	} {
		err := run([]string{cmdNew, flagSeed, "7", flagService, other}, script, io.Discard, io.Discard)

		switch {
		case err == nil:
			t.Errorf("%s: the run finished without being answered", name)
		case !strings.Contains(err.Error(), "the input ended"):
			t.Errorf("%s: error %q does not say the input ended", name, err)
		}
	}
}

// The observer shows the procedure. Without it the player would answer
// blind, which is most of what a guided run is for.
func TestInteractiveShowsWhatAutoDoesNot(t *testing.T) {
	t.Parallel()

	var asked, automatic strings.Builder

	err := run([]string{cmdNew, flagSeed, "7", flagService, other, flagSheet},
		answers("1", 200), &asked, &asked)
	if err != nil {
		t.Fatalf("walking a character: %v", err)
	}

	err = run([]string{cmdNew, flagAuto, flagSeed, "7", flagService, other, flagSheet},
		nil, &automatic, &automatic)
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	if strings.Contains(automatic.String(), "survival: rolled") {
		t.Error("--auto showed the procedure; nobody is reading, so it passes no observer")
	}

	if !strings.Contains(asked.String(), "survival: rolled") {
		t.Error("the guided run did not show the survival throw")
	}
}

// The yes-or-no points, which the walk through Other never reaches: it is
// never rejected, holds no rank, and ends nobody's noble.
//
// Seed 45 into the Navy is rejected, drafted into the Marines, and serves
// seven terms to Colonel and a ducal title. Each prompt below is asserted
// rather than merely walked past, so a change to the dice stream fails here
// instead of quietly leaving these branches unasked.
func TestInteractiveAsksTheYesOrNoPoints(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{cmdNew, flagSeed, "45", flagService, navy, flagSheet},
		answers("1", 300), &out, &out)
	if err != nil {
		t.Fatalf("walking a character: %v", err)
	}

	shown := out.String()

	for _, want := range []string{
		"Rejected. Submit to the draft? (p. 5)",
		"Attempt a commission this term? (p. 6)",
		"Attempt a promotion this term? (p. 6)",
		"Your rank allows +1 on this table. Take it? (p. 9)",
		"Assume the title? (p. 5; Book 3 p. 22)",
		// Both alternatives are offered at each of them.
		"1) yes",
		"2) no",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the run never asked %q", want)
		}
	}
}

// The gambling modifier is offered only to someone who has the expertise, so
// reaching it takes a run that draws the skill: seed 3 through Other, taking
// the Service Skills Table every time.
func TestInteractiveOffersTheGamblingModifier(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	err := run([]string{cmdNew, flagSeed, "3", flagService, other, flagSheet},
		answers("2", 300), &out, &out)
	if err != nil {
		t.Fatalf("walking a character: %v", err)
	}

	if !strings.Contains(out.String(), "Your gambling expertise allows +1 on this table") {
		t.Error("a character with Gambling was never offered the modifier p. 9 gives him")
	}
}

// brokenPipe fails every write, as a terminal does when the reader on the
// other end goes away.
type brokenPipe struct{}

func (brokenPipe) Write([]byte) (int, error) { return 0, errClosedPipe }

var errClosedPipe = errors.New("the other end closed")

// A player nobody can be shown anything is not asked anything either. The
// failure is remembered and reported at the next question, rather than being
// swallowed once per line for the length of a career.
func TestInteractiveStopsWhenItCannotBeRead(t *testing.T) {
	t.Parallel()

	err := run([]string{cmdNew, flagSeed, "7", flagService, other, flagSheet},
		answers("1", 300), io.Discard, brokenPipe{})

	switch {
	case err == nil:
		t.Fatal("the run finished while nobody could see the questions")
	case !errors.Is(err, errClosedPipe):
		t.Errorf("error %q does not carry the write failure that caused it", err)
	}
}

// The questions do not go where the record goes.
//
// A guided run still writes JSON to stdout, so `ctchargen new --seed 7 | jq`
// has to receive a record and not a conversation. The two channels are
// separate arguments for that reason, and this is what holds them apart.
func TestTheQuestionsStayOutOfTheRecord(t *testing.T) {
	t.Parallel()

	var record, asked strings.Builder

	err := run([]string{cmdNew, flagSeed, "7", flagService, other},
		answers("1", 300), &record, &asked)
	if err != nil {
		t.Fatalf("walking a character: %v", err)
	}

	var written map[string]any

	invalid := json.Unmarshal([]byte(record.String()), &written)
	if invalid != nil {
		t.Fatalf("the data channel does not hold a record on its own: %v", invalid)
	}

	if !strings.Contains(asked.String(), "Which skills table?") {
		t.Error("the questions did not go to the channel meant for them")
	}

	// And the record names who answered. A guided run's choices were the
	// player's, and a record that calls them the policy's misdescribes how
	// the character was made - the one thing the log exists to say.
	events, _ := written["events"].([]any)

	choices := 0

	for _, event := range events {
		fields, _ := event.(map[string]any)
		if fields["kind"] != "choice" {
			continue
		}

		choices++

		if fields["by"] != "player" {
			t.Errorf("choice %v says %v answered it, want the player",
				fields["point"], fields["by"])
		}
	}

	if choices == 0 {
		t.Fatal("the record holds no choice events, so it asserts nothing about who answered")
	}
}
