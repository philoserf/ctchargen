package render

import (
	"fmt"
	"regexp"
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
//
// Named Option and not Flag on purpose. cmd/ctchargen/usage.go declares
// seedOption, careerOption, skillsOption and musterOption too, bare - "career", not
// "--career" - because that is what a flag.FlagSet is keyed by, while these
// are words going onto a command line. Both are right for their own package
// and they used to read identically, so a constant moved between the two, or
// checked against the other, differed by two characters and compiled either
// way (#83).
const (
	autoOption    = "--auto"
	seedOption    = "--seed"
	serviceOption = "--service"
	nameOption    = "--name"
	careerOption  = "--career"
	skillsOption  = "--skills"
	musterOption  = "--muster"

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
	line := "Regenerate with " + codeSpan(strings.Join(command(r, rendering), " "))

	// A record generated in-process carries no build - stamp fills it only
	// from the command - and a build nobody recorded is not one to warn
	// about.
	if r.Build == "" {
		return line + "."
	}

	return fmt.Sprintf(
		"%s, on %s — the same seed on a different build is a different character.",
		line, r.Build,
	)
}

// command is the argument list that reproduces this character.
//
// Every value out of the record goes through shellQuote. The record is not
// something this tool wrote - render reads whatever it is handed - and this
// line is printed to be pasted, which makes an unquoted field a way to put
// words in the operator's shell.
func command(r record, rendering string) []string {
	// The seed is the one value that cannot need quoting: it is a uint64
	// written in base ten, so it is digits or nothing.
	args := []string{
		"ctchargen", "new", autoOption, seedOption, strconv.FormatUint(r.Inputs.Seed, 10),
	}

	// The service asked for, which is what reproduces the run - not the
	// service the character ended up in, which the draft may have decided.
	if r.Inputs.Service != "" {
		args = append(args, serviceOption, shellQuote(strings.ToLower(r.Inputs.Service)))
	}

	if r.Inputs.Name != "" {
		args = append(args, nameOption, shellQuote(r.Inputs.Name))
	}

	return append(args,
		careerOption, shellQuote(r.Inputs.Career),
		skillsOption, shellQuote(r.Inputs.Skills),
		musterOption, shellQuote(r.Inputs.Muster),
		rendering)
}

// codeSpan wraps the command in a markdown code span wide enough to hold it.
//
// CommonMark closes a span on the first run of backticks matching the run
// that opened it, so a command carrying one breaks a single-backtick span and
// the reader is shown half a line - and half a command line is worse than
// none. The fence is one backtick longer than the longest run inside.
//
// CommonMark also wants a space either side where the content begins or ends
// with a backtick. There is none here, and none is written: command builds a
// line starting with "ctchargen" and ending with the rendering flag, both
// constants in this file, so neither end can be one.
//
// Shell quoting already makes such a value inert; this is about the reader
// being shown the whole of what he is being offered.
func codeSpan(text string) string {
	longest := 0

	for i := 0; i < len(text); {
		if text[i] != '`' {
			i++

			continue
		}

		run := 0

		for i < len(text) && text[i] == '`' {
			run++

			i++
		}

		longest = max(longest, run)
	}

	fence := strings.Repeat("`", longest+1)

	return fence + text + fence
}

// safeToken matches a value a shell reads back as itself, which is every
// value this tool writes and none of the ones worth worrying about.
//
// No `=`: zsh sets EQUALS by default, so a word beginning with one is
// expanded to the path of the command named after it - `=ls` becomes
// /bin/ls, and `=nosuch` fails the line before the tool sees it. Nothing
// this tool writes carries one, so the class is narrowed rather than the
// position reasoned about.
var safeToken = regexp.MustCompile(`^[A-Za-z0-9@%+:,./_-]+$`)

// shellQuote writes a value so that a shell reads it back as one word.
//
// Not strconv.Quote, which is Go's syntax rather than the shell's. A shell
// still expands `$`, a backtick and a backslash inside double quotes, so
// strconv.Quote("a`id`") produces a command substitution the moment it is
// pasted - which is what the earlier quoting of --name did. Single quotes
// expand nothing at all, and a single quote inside them is closed, escaped
// and reopened.
//
// A value needing none is written bare, because the common case is a line a
// person reads and every value this tool writes is a bare token.
func shellQuote(value string) string {
	if safeToken.MatchString(value) {
		return value
	}

	// Single quotes carry a control character literally, and a newline
	// carried literally ends the line - which breaks the code span, ends
	// the markdown paragraph, and leaves the rest of the record rendering
	// as prose of its own. See escaped.
	if strings.ContainsFunc(value, isControl) {
		return escaped(value)
	}

	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// isControl reports a byte a printed line cannot carry as itself.
func isControl(r rune) bool {
	return r < 0x20 || r == 0x7f
}

// escaped writes a value carrying control characters as one word on one line.
//
// This is the one place the line leaves POSIX sh for the shells a person
// actually pastes into: $'...' is bash and zsh, and in sh it is a plain
// single-quoted string behind a stray $, so the value comes back wrong -
// wrong but inert, which is the trade. A record whose name carries a newline
// is already not a record this tool wrote, and the alternative is a sheet
// whose markdown the record gets to write.
//
// Inert is the half that has to be earned. It holds only because the body
// carries no quote of its own: a `\'` is the escape bash reads, but sh reads
// the same two bytes as a backslash and then the end of the string, and what
// follows is unquoted - a command substitution again. `\x27` says the same
// thing to bash and zsh and says nothing at all to sh.
func escaped(value string) string {
	var out strings.Builder

	out.WriteString("$'")

	for i := range len(value) {
		switch char := value[i]; char {
		case '\\':
			out.WriteString(`\\`)
		case '\'':
			out.WriteString(`\x27`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if isControl(rune(char)) {
				fmt.Fprintf(&out, `\x%02x`, char)

				continue
			}

			out.WriteByte(char)
		}
	}

	out.WriteString("'")

	return out.String()
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
