# ctchargen Walkthrough

*2026-09-10T12:34:13Z by Showboat 0.6.1*
<!-- showboat-id: 98829844-2161-4ac5-b5c2-6b189cfe6120 -->

## Overview

`ctchargen` is a Go CLI that generates Classic Traveller characters by walking
Book 1's character generation procedure, pp. 4-25 of the 1977 text. It has no
dependencies outside the standard library except a JSON Schema validator used
only in tests.

What it produces is not just a character. It produces a **record**: a JSON
document carrying the finished character *and* an ordered log of every step
entered, every die thrown, every question answered and every consequence — with
each consequence naming the throw that caused it. A referee can walk that log
against the printed page and check the tool's work.

The entry point is `cmd/ctchargen/main.go`. Four subcommands:

| Command   | What it does                                                     |
| --------- | ---------------------------------------------------------------- |
| `new`     | generate one character, asking at every choice unless `--auto`   |
| `batch`   | generate many from one base seed, `--auto` only                  |
| `render`  | read a record saved earlier and write it as a sheet or transcript |
| `version` | write the build                                                  |

Everything below follows the call chain from `main` outward. Each code block is
a real command run against this repository; `uvx showboat verify WALKTHROUGH.md`
re-runs them all.

## Architecture

Six packages, and the import arrows point one way. This is enforced by review
rather than by a tool, but it is easy to check.

```bash
go list -deps ./... | grep philoserf | sed 's|github.com/philoserf/ctchargen|.|'
```

```output
./dice
./traveller
./rules
./chargen
./render
./cmd/ctchargen
./internal/docsgate
```

That listing is in dependency order — `dice` and `traveller` first because
neither imports anything else in the module. The edges themselves:

```bash
for p in traveller dice rules chargen render cmd/ctchargen; do printf "%-16s -> %s\n" "$p" "$(go list -f "{{join .Imports \" \"}}" ./$p | tr " " "\n" | grep philoserf | sed "s|github.com/philoserf/ctchargen/||" | paste -sd" " -)"; done
```

```output
traveller        -> 
dice             -> 
rules            -> traveller
chargen          -> dice rules traveller
render           -> chargen traveller
cmd/ctchargen    -> chargen render traveller
```

Read that as four layers:

- **`traveller`** — the domain. The alphabets Book 1 prints (six services, six
  characteristics, four skill tables), the values it works in (`Target`, `Age`,
  `Credits`, `Profile`), and the sums that say "exactly one of". It rolls no
  dice, reads no table and marshals no JSON.
- **`dice`** — a seeded stream. It rolls and never judges; whether a throw met
  its target is a `Target`'s business, which is why `dice` can depend on nothing.
- **`rules`** — every printed table, embedded as JSON, lifted into domain values.
- **`chargen`** — the procedure itself, plus the `Decider` interface it asks
  through and the `Policy` that answers under `--auto`.
- **`render`** — projections: JSON, a character sheet, a transcript.
- **`cmd/ctchargen`** — flag parsing, the interactive player, file output.

`internal/docsgate` has no non-test code at all. It exists to hold the documents
in `docs/` and the code to each other, and it is the one package allowed to know
about both `traveller` and `chargen` and about the documents beside them.

## Layer one: the domain

Start with `traveller`, because everything else is expressed in its vocabulary.

### Targets, and why the zero value fails closed

Book 1 pp. 2-3 fix the die-roll notation: `8+` means eight or more, `3-` means
three or less, and a bare number must be thrown exactly. A `Target` is that
number plus which of the three it is.

```bash
sed -n '18,28p' traveller/target.go
```

```output
// The three modalities (pp. 2-3).
//
// Exactly is first, and deliberately: it makes the zero value of a Target
// "0 exactly", which no throw of dice can meet. A Target that was never set
// therefore fails closed. Were AtLeast the zero value, an unset target would
// read as "0+" and every throw in the procedure would succeed against it.
const (
	Exactly Modality = iota // N, with no sign
	AtLeast                 // N+
	AtMost                  // N-
)
```

`ParseTarget` reads the book's own notation, once, when a table lifts — never
again at throw time. It is deliberately strict about the sign, and the reason is
the reprint's font: under text extraction a printed minus becomes a digit, so an
`N-` target silently reads as `N3`. It also accepts two different minus glyphs,
because someone transcribing p. 11 types what he sees on the page.

```bash
sed -n '60,62p;72,100p' traveller/target.go
```

```output
// minusSigns are the two characters a minus may arrive as: the ASCII hyphen
// a data file is typed with, and the U+2212 the page is set in.
const minusSigns = "-−"
func ParseTarget(s string) (Target, error) {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) == 0 {
		return Target{}, fmt.Errorf("%w: it is empty", ErrTarget)
	}

	modality := Exactly

	if last := runes[len(runes)-1]; last == '+' || strings.ContainsRune(minusSigns, last) {
		if len(runes) == 1 {
			return Target{}, fmt.Errorf("%w %q: a sign with no number", ErrTarget, s)
		}

		modality = AtLeast
		if last != '+' {
			modality = AtMost
		}

		runes = runes[:len(runes)-1]
	}

	// P. 3 puts the sign of a throw after its number and the sign of a
	// modifier before it: "throws are always followed by a sign unless the
	// number must be thrown exactly, and DMs are always preceded by a
	// sign." A leading sign here means a DM was passed off as a target.
	digits := string(runes)
	if digits == "" || digits[0] == '+' || digits[0] == '-' || strings.HasPrefix(digits, "−") {
		return Target{}, fmt.Errorf("%w %q: a leading sign marks a modifier, not a throw", ErrTarget, s)
	}
```

### Sums: "exactly one of", held by the compiler

The domain's most important idea is how it closes a set of alternatives. Go's
type switch over an interface promises nothing — the compiler will not tell you a
case is missing, and neither will the `exhaustive` linter. So the six places
where "exactly one of" is the rule are written as **sealed interfaces with a fold**.

```bash
sed -n '3,15p' traveller/sums.go
```

```output
// The sums: the places where "exactly one of" is the rule.
//
// Each is an interface with a Fold method taking a cases interface that has
// one method per case, and an unexported seal so the set cannot be extended
// from outside this package. Adding a case adds a method to the cases
// interface, and every implementation stops compiling until it handles it.
//
// This is the same mechanism the Decider uses for choice points, and it is
// here for the same reason: Go's compiler does not check that a type switch
// covers every implementation of an interface, and neither does the
// exhaustive linter. A type switch over a sealed interface is an
// exhaustiveness promise that nothing keeps.

```

`Enlistment` is the smallest one, and it shows the pattern whole. There are
exactly three ways p. 5's enlistment step can end, and "declined the draft" is a
case of the type rather than a nil service with a flag beside it.

```bash
sed -n '17,53p' traveller/sums.go
```

```output
//
// The civilian who declined the draft is a case of the type, not a nil
// service with a flag beside it. The draft is declinable per E001: p. 5
// prints both "he may submit to the draft" and "the character must submit to
// the draft", and the may reading governs.
type Enlistment interface {
	Fold(cases EnlistmentCases) error
	sealedEnlistment()
}

// EnlistmentCases handles each way p. 5's enlistment step can end.
type EnlistmentCases interface {
	Enlisted(service ServiceName) error
	Drafted(service ServiceName) error
	DeclinedTheDraft() error
}

// Enlisted is a successful enlistment throw against the Prior Service Table
// (pp. 5, 10).
type Enlisted struct{ Service ServiceName }

// Drafted is entry by the draft: one die, and the service with that draft
// number — "possibly the very service which had just previously rejected his
// enlistment" (p. 5).
type Drafted struct{ Service ServiceName }

// DeclinedTheDraft ends generation with an eighteen-year-old civilian: no
// service, no terms, no skills, no benefits. A complete record (E001).
type DeclinedTheDraft struct{}

func (e Enlisted) Fold(c EnlistmentCases) error       { return c.Enlisted(e.Service) }
func (d Drafted) Fold(c EnlistmentCases) error        { return c.Drafted(d.Service) }
func (DeclinedTheDraft) Fold(c EnlistmentCases) error { return c.DeclinedTheDraft() }
func (Enlisted) sealedEnlistment()                    {}
func (Drafted) sealedEnlistment()                     {}
func (DeclinedTheDraft) sealedEnlistment()            {}

```

The six sums are `Enlistment`, `TableResult`, `BenefitRow`, `Departure`,
`WeaponBenefit` and `Event`. Their cases interfaces are where you will feel the
compiler if you add a case:

```bash
grep -n 'Cases interface' traveller/*.go
```

```output
traveller/event.go:19:type EventCases interface {
traveller/sums.go:28:type EnlistmentCases interface {
traveller/sums.go:106:type TableResultCases interface {
traveller/sums.go:146:type BenefitRowCases interface {
traveller/sums.go:218:type DepartureCases interface {
traveller/sums.go:269:type WeaponBenefitCases interface {
```

## Layer two: the tables

`rules` holds every table of pp. 4-25 as embedded JSON and lifts it into domain
values. The package's own doc comment says why the lift is where validation
lives:

```bash
sed -n '1,17p' rules/rules.go
```

```output
// Package rules holds every table of Book 1 pp. 4-25 as embedded data, and
// lifts it into the domain values of the traveller package.
//
// The lift is the validation. A cell that names no service, no
// characteristic, no benefit any row prints, or a target in a notation
// pp. 2-3 do not use, fails here — and because loading happens once on first
// use, a table that will not lift is a build defect that surfaces
// immediately rather than a runtime condition some path might reach.
//
// Every table is transcribed twice: once into data/, and once into this
// package's tests, from the same visual reading of the page. The two must
// agree. That is what the reprint's font requires — its glyph substitutions
// look like data, so on Book 1 p. 9 the dash cells of Mustering Out Table 1
// extract as the digit 4 and "Travellers' Aid" extracts as Travellers9. A
// run that trusted the extraction would give a Scout a seventh benefit he
// does not have.
package rules
```

```bash
ls rules/data/
```

```output
aging.json
mustering.json
nobility.json
services.json
ships.json
skills.json
weapons.json
```

Each file carries its own page citation. Here is the head of the Prior Service
Table's data — the six services of p. 10, their enlistment throws, and the die
modifiers that apply:

```bash
sed -n '1,22p' rules/data/services.json
```

```output
{
  "_cite": "Book 1 p. 10 (Prior Service Table, Table of Ranks); p. 21 (Annual Retirement Pay); p. 23 (Rank and Service Skills)",
  "_note": "Transcribed from a visual read of the page. Never from extracted text: the reprint's font turns a printed minus into a digit and an em dash into a 4.",

  "services": ["Navy", "Marines", "Army", "Scouts", "Merchants", "Other"],

  "enlistment": [
    { "target": "8+", "dms": [{ "dm": 1, "if": "Intelligence 8+" }, { "dm": 2, "if": "Education 9+" }] },
    { "target": "9+", "dms": [{ "dm": 1, "if": "Intelligence 8+" }, { "dm": 2, "if": "Strength 8+" }] },
    { "target": "5+", "dms": [{ "dm": 1, "if": "Dexterity 6+" }, { "dm": 2, "if": "Endurance 5+" }] },
    { "target": "7+", "dms": [{ "dm": 1, "if": "Intelligence 6+" }, { "dm": 2, "if": "Strength 8+" }] },
    { "target": "7+", "dms": [{ "dm": 1, "if": "Strength 7+" }, { "dm": 2, "if": "Intelligence 6+" }] },
    { "target": "3+", "dms": [] }
  ],

  "draft": [1, 2, 3, 4, 5, 6],

  "survival": [
    { "target": "5+", "dms": [{ "dm": 2, "if": "Intelligence 7+" }] },
    { "target": "6+", "dms": [{ "dm": 2, "if": "Endurance 8+" }] },
    { "target": "5+", "dms": [{ "dm": 2, "if": "Education 6+" }] },
    { "target": "7+", "dms": [{ "dm": 2, "if": "Endurance 9+" }] },
```

Nothing in that file is trusted. `rules/parse.go` checks every printed string
against the domain's own closed alphabet, indexing each type by the name its
values give themselves — so a data file's spelling is checked against the type
and against nothing else:

```bash
sed -n '16,40p' rules/parse.go
```

```output

// named indexes an alphabet by the name its values give themselves, so that
// a data file's spelling is checked against the type and nothing else.
func named[T fmt.Stringer](values []T) map[string]T {
	index := make(map[string]T, len(values))
	for _, v := range values {
		index[v.String()] = v
	}

	return index
}

var (
	characteristics  = named(traveller.Characteristics[:])
	serviceNames     = named(traveller.ServiceNames[:])
	skillTables      = named(traveller.SkillTables[:])
	weaponCategories = named(traveller.WeaponCategories[:])
	passageClasses   = named(traveller.PassageClasses[:])
	shipKinds        = named(traveller.ShipKinds[:])
	titles           = named(traveller.Titles[:])
)

func lookup[T any](index map[string]T, name, kind string) (T, error) {
	value, ok := index[name]
	if !ok {
```

The tables are loaded once per process and validated on the way in:

```bash
sed -n '58,61p' rules/rules.go
```

```output
// Load returns the lifted rules, loading and validating them once.
func Load() (*Rules, error) { return loaded() }

var loaded = sync.OnceValues(load)
```

A cell that will not lift is therefore a build defect that surfaces on the first
call, not a runtime condition some rare path might reach. The `rules` tests
carry the *second* transcription of every table, typed independently from the
same visual reading pass — that second copy is the check the reprint's font
requires, and retyping it from the first copy would check nothing.

## Layer three: the dice

`dice` is thirty lines of substance. Its whole contract is that a seed means
something, and the package doc states the properties that make it so:

```bash
sed -n '1,20p' dice/dice.go
```

```output
// Package dice draws the die rolls Classic Traveller's character generation
// procedure calls for, from a seeded stream.
//
// It rolls and it never judges. Whether a throw meets its target is the
// business of a Target (Book 1 pp. 2-3), which lives in the traveller
// package; keeping that out of here is what lets dice depend on nothing.
//
// Two properties of this package are load-bearing, in the sense that a seed
// means nothing without them:
//
//   - The die is IntN(6) + 1. An IntN(36), or a masked Uint64, is the same
//     PCG under the same seed and an entirely different character.
//   - A 2D throw is two of those in sequence, first die then second.
//   - Among is IntN(n), drawn from the same stream, for the choices the book
//     leaves to a player without naming a basis.
//
// Changing either changes every seeded character. That is an ordinary
// change, not a breaking one, but it is never an accidental one.
package dice

```

(That comment says "Two properties" over a list of three, and "Changing either"
of three — a stale count left by the change that added `Among`. It is recorded
as a finding below; the properties themselves are right.)

`Among` deserves a note, because it is not a die. Where the book hands a choice
to a player and names no basis for making it — which weapon to take off the p. 12
blade list — the tool draws rather than always taking the first name. It draws
from the same stream, so a varied choice consumes like a throw does and the same
seed still makes the same character.

```bash
sed -n '43,58p' dice/dice.go
```

```output
// begins on one page and ends on the other).
func (s *Stream) Die() int {
	s.drawn++

	return s.r.IntN(faces) + 1
}

// Among returns an index into a set of n alternatives: 0 through n-1.
//
// It is not a die. The procedure has no throw here - it is drawn where the
// book hands the choice to a player and names no basis for it, so that
// --auto stops answering every such question the same way (#34). A die and a
// modulo would not do: the weapon lists of pp. 12-13 are longer than six, and
// even where they are not, 6 mod n makes the first names likelier.
//
// It draws from the same stream, so a varied choice consumes like a throw
```

## Layer four: the engine

`chargen.Generate` is the whole public entry point. It loads the tables, builds a
`run`, applies options, wraps the caller's `Decider` in the logging decorator,
and walks the procedure.

```bash
sed -n '52,90p' chargen/generate.go
```

```output
//
// The order within a term is the exposition's, not the worked example's
// (E002): survival, commission, promotion, skills, reenlistment, and then
// the aging round at the end of the term (E006). That order is what a seed
// means, so it is a fixed choice rather than an incidental one.
func Generate(in Inputs, decider Decider, options ...Option) (*Character, error) {
	tables, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading the rules: %w", err)
	}

	record := newLog()

	run := &run{
		tables: tables,
		roll:   dice.New(in.Seed),
		log:    record,
		by:     traveller.ByPolicy,
		char: &Character{
			Name: in.Name, Age: traveller.NewAge(startingAge), Inputs: in, Ruleset: Ruleset,
		},
	}

	for _, option := range options {
		option(run)
	}

	run.decide = logging{to: decider, by: run.by, log: record}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.sortSkills()

	run.char.Events = record.events
	run.char.Errata = record.stamped()

```

`run.generate` is the procedure at its coarsest. Note that a civilian — someone
who declined the draft — returns before `serve` is ever entered, and that the
dead skip mustering out but are still assessed for a title.

```bash
sed -n '131,159p' chargen/generate.go
```

```output
func (r *run) generate() error {
	r.rollProfile()

	err := r.enlist()
	if err != nil {
		return err
	}

	_, served := r.char.ServedIn()
	if !served {
		return r.assessTitle()
	}

	err = r.serve()
	if err != nil {
		return err
	}

	if !r.dead {
		err := r.musterOut()
		if err != nil {
			return err
		}

		r.pension()
	}

	return r.assessTitle()
}
```

### One term

`run.serve` loops terms until one of them ends the service. The loop has no
bound and needs none under dice — the reenlistment throw fails eventually — but
that is a **contract on the `Roller`**, not a property of the function, and the
comment says so. A test roller that answers every 2D throw with 12 would never
return.

```bash
sed -n '10,31p' chargen/serve.go
```

```output
//
// The loop has no bound, and needs none under dice: the reenlistment throw of
// p. 6 fails eventually and the survival throw of p. 5 ends things sooner
// than that, so a career terminates with probability 1. That is a contract on
// the Roller, not a property of this function - a roller that answers every
// 2D throw with 12 never fails a reenlistment and this never returns, which is
// why the scripted career that walks past the Aging Table's last column hands
// one a count of twelves rather than an endless supply (#54).
func (r *run) serve() error {
	for term := 1; ; term++ {
		done, err := r.term(traveller.Term(term))
		if err != nil {
			return err
		}

		if done {
			return nil
		}
	}
}

// term runs one term of service in the exposition's order (E002): survival,
```

`run.term` is the six steps of a term, in the order the exposition gives rather
than the order the book's worked example rolls them (E002), with the aging round
last (E006):

```bash
sed -n '39,86p' chargen/serve.go
```

```output
	r.log.step(fmt.Sprintf("term %d", term), "pp. 5-7")

	r.char.Terms = int(term)

	if !r.survive(term) {
		// The fatal term counts, and its four years with it (E004).
		r.char.Age = r.char.Age.PlusYears(traveller.Years)
		r.char.Departure = traveller.KilledBySurvivalThrow{}
		r.dead = true

		return true, nil
	}

	err := r.commission(term)
	if err != nil {
		return false, err
	}

	err = r.promote(term)
	if err != nil {
		return false, err
	}

	r.grantEligibility(term)

	err = r.trainSkills()
	if err != nil {
		return false, err
	}

	intent, forced, err := r.reenlist(term)
	if err != nil {
		return false, err
	}

	r.char.Age = r.char.Age.PlusYears(traveller.Years)

	err = r.agingRound(term)
	if err != nil {
		return false, err
	}

	if r.dead {
		return true, nil
	}

	if forced || intent == traveller.Continue {
		return false, nil
```

### The Decider, and the one gate every answer passes

The engine never decides anything on its own behalf. Every question goes to a
`Decider` — twelve methods, one per choice point, each typed to its own alphabet.
That typing is what closes the set: adding a thirteenth question breaks every
implementation at compile time.

```bash
grep -nE '^	[A-Z][A-Za-z0-9]*\(' chargen/decider.go | sed 's/(.*//' | cat -n
```

```output
     1	34:	Service
     2	38:	SubmitToDraft
     3	43:	AttemptCommission
     4	47:	AttemptPromotion
     5	58:	SkillTable
     6	67:	Weapon
     7	76:	ReenlistIntent
     8	82:	MusterTable
     9	86:	MusterTable1DM
    10	90:	MusterTable2DM
    11	97:	MusterWeapon
    12	107:	AssumeTitle
```

Twelve, and the enum `traveller.ChoicePoint` has twelve constants whose
`String()` is each method's name. That is unavoidably a second list kept
parallel to the interface, so a reflection gate in `internal/docsgate` holds the
two together — and holds both to `docs/POLICY.md`, which carries one row per
method.

Every decider — the auto policy, the interactive player, the scripted test
deciders — reaches the engine through one decorator, `chargen/logging.go`. It
must implement all twelve to compile, so a choice point cannot reach the engine
without reaching the log. Its `record` method does two jobs:

```bash
sed -n '206,232p' chargen/logging.go
```

```output
// engine (generate.go wraps every one of them), and it already holds both
// the offered set and the answer, because it records both. Checking here
// rather than at each application site leaves one definition of answering
// outside the offer, which two cannot disagree with.
//
// It compares rendered names rather than values, and every method above has
// the typed offered set and the typed answer in hand before rendering. That
// is deliberate: MusterWeapon answers with a sum, and comparing a sum against
// a list of them needs a fold, so one path here would have to be strings
// whatever the other eleven did.
//
// What it costs is an assumption. The check is sound exactly as long as
// String() is injective on every alphabet the engine offers - two values that
// spelled the same would let a decider answer with one and have the other
// recorded, and nothing would notice. That was true when this was written and
// stated nowhere (#48). It is stated here now, and held by
// traveller.TestNoTwoValuesOfAnAlphabetShareASpelling.
func (l logging) record(point traveller.ChoicePoint, from []string, chosen string) error {
	if !slices.Contains(from, chosen) {
		return fmt.Errorf("%w: %v offered %v, and was answered %q",
			errNotOffered, point, from, chosen)
	}

	l.log.choice(point, l.by, from, chosen)

	return nil
}
```

That injectivity assumption is itself held by a test — and that test's claim to
cover *every* alphabet is held by a second test that reads the package's own
source for `var X = [...]` declarations and fails on one nobody checks:

```bash
grep -n 'var alphabet' -A 1 traveller/spelling_test.go; grep -c 'spelledOnce(t,' traveller/spelling_test.go
```

```output
84:var alphabet = regexp.MustCompile(`(?m)^var ([A-Z]\w*) = \[\.\.\.\]`)
85-
12
```

### The auto policy

`chargen/policy.go` is the `--auto` decider. It is total, deterministic, and a
pure function of what it is handed: where it needs to know an option is legal,
the engine supplies that by what it puts in the offered set, never by the policy
reaching into the character.

Three orthogonal strategies, each a closed type rather than a string — which is
why `chargen` has no `Validate` method. A flag word becomes a domain value once,
at the boundary, and past there a `Career` is a `Career`.

```bash
sed -n '68,90p' chargen/policy.go
```

```output
// The skills strategies of docs/POLICY.md, in its order.
const (
	SkillsAdvanced Skills = iota
	SkillsService
	SkillsPersonal
)

// The mustering out strategies of docs/POLICY.md, in its order.
const (
	MusterCash Muster = iota
	MusterGoods
	MusterSpartan
)

// The three alphabets, for parsing a flag and for listing the choices. They
// are unexported: nothing outside needs to iterate a strategy set, and the
// two things that do need one - the parser and the help - are below.
//
//nolint:gochecknoglobals // three immutable tables, and Go has no const slice.
var (
	careerStrategies = []Career{CareerServe, CareerRetire, CareerOneTerm}
	skillsStrategies = []Skills{SkillsAdvanced, SkillsService, SkillsPersonal}
	musterStrategies = []Muster{MusterCash, MusterGoods, MusterSpartan}
```

Most methods are one line — `AttemptCommission` is always yes, because nothing
in the rules costs a character a failed attempt. Two are more interesting.

`Service` computes the odds of meeting each offer's printed target on two dice
after its printed modifier, and takes the best, ties going to book order. It is
not a table and not a rule; it is arithmetic over what the offer carries:

```bash
sed -n '228,245p' chargen/policy.go
```

```output
// odds is the chance of meeting an offer's target on two dice after its
// modifier. It is computed from the printed target and the printed DM; it is
// not a table of its own and it is not a rule.
func odds(offer traveller.EnlistmentOffer) float64 {
	ways := 0

	const faces = 6

	for first := 1; first <= faces; first++ {
		for second := 1; second <= faces; second++ {
			if offer.Target.Satisfied(first + second + offer.DM) {
				ways++
			}
		}
	}

	return float64(ways) / float64(faces*faces)
}
```

`SkillTable` is the one that had to be re-thought. Ranking alone designated the
Advanced Education table every time it was offered and Personal Development
never, so a character generated under the default never raised a characteristic
— though p. 11's first table is exactly how that is done. The engine now hands
the decider a `taken` slice parallel to `from`, and the policy prefers the tables
he has trained on least before applying its ranking:

```bash
sed -n '315,340p' chargen/policy.go
```

```output

	return prefer(thinnest(from, taken), ranked), nil
}

// thinnest is the offered tables the character has had fewest results off.
//
// A caller with no counts - a test constructing a policy directly - gets the
// whole offered set back, so the ranking decides alone and the answer is what
// it was before the counts existed.
func thinnest(from []traveller.SkillTable, taken []int) []traveller.SkillTable {
	if len(taken) != len(from) {
		return from
	}

	fewest := slices.Min(taken)

	thin := make([]traveller.SkillTable, 0, len(from))

	for i, table := range from {
		if taken[i] == fewest {
			thin = append(thin, table)
		}
	}

	return thin
}
```

`Weapon` is the only method handed a `Vary`, and the only one that draws. The
page prints a list in an order and printed order is not a preference — taking
the first name is as invented as drawing, and drawing admits it. Before this
change, thirty auto-generated characters carried twenty-two Body Pistols and
seventeen Daggers between them, though every name on both lists was offered.

## Layer five: rendering

`render` projects a `*chargen.Character` into a flat `record` struct and then
into JSON, a character sheet, or a transcript. The domain types are the
interface; JSON is a projection, and where the two disagree the codec absorbs it.

```bash
sed -n '1,10p' render/json.go
```

```output
// Package render projects a generated character into the shapes a reader
// wants: JSON, a character sheet in the book's own style, and a transcript
// of the generation record.
//
// The domain types are the interface and JSON is a projection. Where the two
// disagree - a sum that must marshal flat, an array-backed profile that must
// marshal as six named keys - the codec here absorbs the difference and the
// domain type keeps its shape.
package render

```

The wire shape is one struct. Its JSON tags are the record's public field names,
and `docs/character.schema.json` describes it in draft 2020-12:

```bash
sed -n '81,103p' render/json.go
```

```output
	Record          int               `json:"record"`
	Name            string            `json:"name"`
	UPP             string            `json:"upp"`
	Characteristics map[string]int    `json:"characteristics"`
	Age             ageRecord         `json:"age"`
	Terms           int               `json:"terms"`
	Enlistment      enlistmentRecord  `json:"enlistment"`
	Service         string            `json:"service,omitempty"`
	Rank            int               `json:"rank,omitempty"`
	RankTitle       string            `json:"rankTitle,omitempty"`
	Skills          []skillRecord     `json:"skills"`
	Benefits        benefitsRecord    `json:"benefits"`
	Pension         int64             `json:"annualRetirementPay,omitempty"`
	Departure       *departureRecord  `json:"departure,omitempty"`
	Title           *titleRecord      `json:"title,omitempty"`
	Inputs          inputsRecord      `json:"inputs"`
	Errata          []string          `json:"errata"`
	Ruleset         string            `json:"ruleset"`
	Policy          int               `json:"policy,omitempty"`
	Events          []json.RawMessage `json:"events"`
	Build           string            `json:"build,omitempty"`
}

```

The write side is held by folds — one codec per sum, so adding a case stops this
package compiling until the projection learns to write it:

```bash
grep -n 'Codec struct\|^func fold' render/fold.go
```

```output
13:type enlistmentCodec struct{ out enlistmentRecord }
33:func foldEnlistment(from traveller.Enlistment) enlistmentRecord {
46:type departureCodec struct{ out departureRecord }
80:func foldDeparture(from traveller.Departure) departureRecord {
158:type eventCodec struct {
```

The **read** side is not. `render/decode.go` unmarshals into the same `record`
struct and deliberately does not rebuild domain types, so `SheetFrom` and
`TranscriptFrom` work on strings — comparing, for example, a ship's kind against
`traveller.ScoutShip.String()`. The compiler cannot see those comparisons; what
holds them is a round-trip test.

```bash
sed -n '11,20p' render/decode.go
```

```output
var ErrNotARecord = errors.New("not a character record")

// decode reads a record back from what JSON wrote.
//
// Nothing is rebuilt into a domain type. The record is a projection of the
// domain values, and a sheet is a projection of the record: rebuilding an
// Enlistment or a Departure back into an interface would give the renderer
// nothing it does not already have, and would need a decoder for every sum
// that could go wrong in a way the wire shape cannot.
func decode(text []byte) (record, error) {
```

### Provenance: the line at the bottom of every sheet

A sheet is what a referee reads and keeps. The seed reached the JSON record and
the transcript's opening line and never reached the sheet — so a sheet spun,
read and liked was a character discarded. `render/provenance.go` fixes that by
printing the *command*, not the seed, because running it is what the reader
wants to do.

The command is built out of the record, and the record is not something this
tool wrote — `render` reads whatever it is handed — so every value goes through
a shell quoter before it reaches a line a person is told to paste:

```bash
sed -n '155,180p' render/provenance.go
```

```output
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
```

And a character whose choices were made at the keyboard gets a different line,
because the seed alone does not bring him back:

```bash
sed -n '52,68p' render/provenance.go
```

```output
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

```

## Layer six: the command line

`cmd/ctchargen/main.go` is a switch over the first argument and nothing else.
Note that help asked for leaves by `out` and returns nil, so `--help` exits 0; a
bare `ctchargen` with no arguments stays an error, because nothing was asked for.

```bash
sed -n '35,68p' cmd/ctchargen/main.go
```

```output
func run(args []string, in io.Reader, out, asking io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: ctchargen %s; run `ctchargen --help`", errUsage, commandList)
	}

	switch args[0] {
	case "new":
		return newCharacter(args[1:], in, out, asking)
	case "batch":
		return batch(args[1:], out, asking)
	case "render":
		return renderRecord(args[1:], out)
	case "version":
		return version(args[1:], out)
	// Help asked for is not a misuse: it leaves by out and exits 0. A bare
	// `ctchargen` stays an error, because nothing was asked for at all.
	// There is no `help <command>`; --help on the command is the one way in,
	// so there is one place a command's flags are described - and a word
	// after it is refused rather than dropped, as `new` refuses one.
	case "help", "-h", "-help", "--help":
		if len(args) > 1 {
			return fmt.Errorf(
				"%w: help takes no arguments, and was given %q; run `ctchargen <command> --help` for one command's flags",
				errUsage, args[1],
			)
		}

		return writeTopLevelHelp(out)
	default:
		return fmt.Errorf("%w: unknown command %q; want %s; run `ctchargen --help`",
			errUsage, args[0], commandList)
	}
}

```

The two output channels are taken apart deliberately: `out` is the data channel —
the record, the sheet, the transcript — and `asking` is where the tool talks to
the person driving it, so that `ctchargen new --seed 7 | jq` pipes a record and
not a conversation.

Here is the tool actually running. Every command below is executed against this
repository when the walkthrough is verified. `-buildvcs=false` keeps the embedded
build stamp from changing with the working tree.

```bash
go run -buildvcs=false ./cmd/ctchargen --help
```

```output
usage: ctchargen <command> [flags]

  new      generate one character, asking at every choice unless --auto
  batch    generate many characters from one base seed, under --auto
  render   write a record saved earlier as a sheet or as a transcript
  version  write the build

Run `ctchargen <command> --help` for that command's flags.
```

`new` has eleven flags, and the help lists them from the flag set itself rather
than from a hand-typed list — so the list cannot go stale, and the defaults are
printed because a strategy flag left alone still chooses:

```bash
go run -buildvcs=false ./cmd/ctchargen new --help
```

```output
usage: ctchargen new [flags]

  --answers string   answers to replay before asking, as 1,2,1; what a stopped session prints
  --auto             decide by the policy at every choice, rather than asking
  --career strategy  the --auto career strategy: serve, retire, oneterm (default serve)
  --force            replace the output file if it already exists
  --history          write the generation record rather than JSON
  --muster strategy  the --auto mustering out strategy: cash, goods, spartan (default cash)
  --name string      the character's name
  -o string          write to this file rather than to standard output
  --seed uint        the seed to generate from; one is drawn if absent
  --service string   attempt enlistment in this service: navy, marines, army, scouts, merchants, other
  --sheet            write the character sheet rather than JSON
  --skills strategy  the --auto skills strategy: advanced, service, personal (default advanced)
```

### A character, end to end

One seed, the auto policy, rendered as the sheet a referee would keep:

```bash
go run -buildvcs=false ./cmd/ctchargen new --auto --seed 42 --name 'Alexander Jamison' --sheet
```

```output
# Alexander Jamison

UPP 798782, age 22, Army Captain (service chosen by the policy), 1 term, forced out

## Skills

- ATV-1
- Electronic-1
- Rifle-1
- Submachine Gun-1
- Tactics-1

## Possessions

- CR 22000

## Service record

- Army, enlisted
- forced out after term 1

---

Regenerate with `ctchargen new --auto --seed 42 --name 'Alexander Jamison' --career serve --skills advanced --muster cash --sheet`, on github.com/philoserf/ctchargen (devel) — the same seed on a different build is a different character.
```

The name carried a space, so `shellQuote` wrapped it in single quotes and the
line is safe to paste. `(service chosen by the policy)` appears because the
policy named the Army and the enlistment throw succeeded — had the throw failed
and the draft put him somewhere else, the headline would say so instead.

The same seed as a transcript is the audit trail. Its opening:

```bash
go run -buildvcs=false ./cmd/ctchargen new --auto --seed 42 --history | sed -n '1,26p'
```

```output
# Generation record: (unnamed)

Regenerate with `ctchargen new --auto --seed 42 --career serve --skills advanced --muster cash --history`, on github.com/philoserf/ctchargen (devel) — the same seed on a different build is a different character.


## characteristics (p. 4)

  2. Strength: rolled 4+3
  3. Strength 7 (from 2)
  4. Dexterity: rolled 4+4
  5. Dexterity 8 (from 4)
  6. Endurance: rolled 6+2
  7. Endurance 8 (from 6)
  8. Intelligence: rolled 2+5
  9. Intelligence 7 (from 8)
 10. Education: rolled 5+3
 11. Education 8 (from 10)
 12. Social Standing: rolled 1+1
 13. Social Standing 2 (from 12)
 14. UPP 788782 at age 18

## enlistment (pp. 5, 10)

 16. Service: policy chose Army from Navy, Marines, Army, Scouts, Merchants, Other
 17. enlistment, Army: rolled 4+6 +3 = 13 against 5+, made
 18. enlisted in the Army (from 17)
```

Three things to read out of that.

The `(from N)` suffixes are the causal edges: an `OutcomeEvent` carries the
sequence number of the throw or choice that produced it, so "enlisted in the
Army" points at the throw two lines above it. That is what makes the log
auditable rather than merely verbose.

The numbers skip — 14, then 16 — because step headings are events too. They
carry a sequence number in the JSON record and render as an unnumbered `##`
heading here.

And the UPP is `788782` at 18 but `798782` on the sheet above: something raised
Dexterity by one along the way. The transcript is where you find out what.

Here is the term itself:

```bash
go run -buildvcs=false ./cmd/ctchargen new --auto --seed 42 --history | sed -n '/## term 1/,/## mustering/p'
```

```output
## term 1 (pp. 5-7)

 22. survival: rolled 2+3 +2 = 7 against 5+, made
 23. survived term 1 (from 22)
 24. AttemptCommission: policy chose yes from yes, no
 25. commission: rolled 6+4 +1 = 11 against 5+, made
 26. commissioned: Lieutenant (from 25)
 27. Submachine Gun-1 (from 25)
 28. AttemptPromotion: policy chose yes from yes, no
 29. promotion: rolled 6+3 +1 = 10 against 6+, made
 30. promoted: Captain (from 29)

## skills and training (pp. 6, 11)

 32. SkillTable: policy chose Advanced Education Table (education 8+) from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 33. Advanced Education Table (education 8+): rolled 2
 34. Tactics-1 (from 33) [E002]
 35. SkillTable: policy chose Advanced Education Table from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 36. Advanced Education Table: rolled 3
 37. Electronic-1 (from 36) [E002]
 38. SkillTable: policy chose Service Skills Table from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 39. Service Skills Table: rolled 1
 40. ATV-1 (from 39) [E002]
 41. SkillTable: policy chose Personal Development Table from Personal Development Table, Service Skills Table, Advanced Education Table, Advanced Education Table (education 8+)
 42. Personal Development Table: rolled 2
 43. Dexterity +1, 8 to 9 (from 42) [E002]
 44. reenlistment: rolled 3+3 = 6 against 7+, missed
 45. left the service after term 1: reenlistment denied (from 44)

## mustering out, 2 rolls (pp. 7, 9, 21-23)
```

There is the Dexterity: event 43, off the Personal Development Table.

And there is `thinnest` working. Four eligibilities were spent — two for the
initial term, one for the commission, one for the promotion — and the policy
designated a *different* table each time: the Education 8+ table first (its
ranking), then Advanced Education, then Service Skills, then Personal
Development. Under pure ranking all four would have been the 8+ table.

`[E002]` is an erratum stamp. Every skill taken this term names the reading that
put the training step where it is — the exposition's per-term order rather than
the worked example's. Fifteen such readings are recorded in `docs/ERRATA.md`, and
a record names every one that governed it.

The term ends on a failed reenlistment throw, which is `traveller.ForcedOut` —
one of the five cases of the `Departure` sum. The other four are `Discharged`,
`Retired`, and two deaths.

### Death is an outcome, not an error

Under the 1977 text a failed survival throw kills the character. There is no
"injured instead" option — that is the 1981 revision, and it is the single
likeliest thing for a programmer to get wrong from memory. The tool prints the
corpse:

```bash
go run -buildvcs=false ./cmd/ctchargen new --auto --seed 3 --service marines --sheet | head -5
```

```output
# (unnamed)

UPP 324994, age 46, Other (drafted after the Marines refused him), 7 terms, killed by the survival throw

## Skills
```

That one line carries three of the ideas above at once. `--service marines`
forced the enlistment *attempt* only — the throw was still made, it failed, and
the draft put him in the Other service, "possibly the very service which had just
previously rejected his enlistment" (p. 5). The headline says
`drafted after the Marines refused him` rather than silently showing a service
nobody asked for. And he died in term 7, which still counts: the fatal term and
its four years are included, so he is 46 and not 42 (E004).

### Batches

`batch` generates many characters from one base seed — member *i* uses seed
`base + i`, so `batch --count 1 --seed N` produces exactly what `new --seed N`
produces. It always says what it did, on the `asking` channel, because a run that
wrote seventy-four corpses and printed nothing was a real complaint:

```bash
go run -buildvcs=false ./cmd/ctchargen batch --auto --count 8 --seed 100 2>&1 >/dev/null
```

```output
8 written, 3 died
```

`--survivors` does not reroll anyone. It passes over a dead character and goes on
to the next seed, so every character written is still exactly the character his
own seed makes — what changes is which seeds appear:

```bash
go run -buildvcs=false ./cmd/ctchargen batch --auto --count 8 --seed 100 --survivors 2>&1 >/dev/null
```

```output
8 written, 3 passed over for dying
```

The records themselves go to standard output as JSONL, one per line, streamed as
they are generated so a pipe sees the first one immediately:

```bash
go run -buildvcs=false ./cmd/ctchargen batch --auto --count 3 --seed 100 2>/dev/null | cut -c1-96
```

```output
{"record":1,"name":"","upp":"759A66","characteristics":{"Dexterity":5,"Education":6,"Endurance":
{"record":1,"name":"","upp":"86A7CC","characteristics":{"Dexterity":6,"Education":12,"Endurance"
{"record":1,"name":"","upp":"068289","characteristics":{"Dexterity":6,"Education":8,"Endurance":
```

### Reading a record back

`render` takes a record written earlier and writes it as a sheet or a transcript.
It **reads** the record rather than regenerating it from the seed: regenerating
would give the same character within this build and a different one across
builds, which is precisely the promise the tool does not make.

The repository's goldens are records, so they are what `render` can be pointed
at directly:

```bash
go run -buildvcs=false ./cmd/ctchargen render chargen/testdata/navy-captain.json
```

```output
# (unnamed)

UPP 766BDF, age 46, Navy Captain, 7 terms, duke/duchess, retired

## Skills

- Engineer-1
- Forward Observer-1
- Jack of all Trades-1
- Navigation-2
- Pilot-1
- Ship's Boat-2
- Vacc Suit-2

## Possessions

- CR 35000
- High Passage
- Travellers' Aid Society membership
- CR 8000 a year in retirement pay

## Service record

- Navy, enlisted
- retired after term 7

---

Regenerate with `ctchargen new --auto --seed 4 --service navy --career serve --skills advanced --muster cash --sheet`.
```

That provenance line ends with a bare full stop and no build clause, because the
golden was generated in-process by a test rather than by the command — and a
build nobody recorded is not one to warn about. `stamp()` in `main.go` is the
only thing that fills the field.

Each golden is a triple: the record, the sheet it renders to, and the transcript.
All three are regenerated, never hand-edited.

```bash
ls chargen/testdata/ | grep '^navy-captain'
```

```output
navy-captain.json
navy-captain.sheet.md
navy-captain.transcript.md
```

## The gates

Five documents live in `docs/`, and each governs something. What keeps them from
drifting away from the code is `internal/docsgate`, a package with no non-test
code whose whole job is to hold documents and code to each other in both
directions:

```bash
grep -h '^func Test' internal/docsgate/*.go | sed 's/func //; s/(t \*testing.T) {//'
```

```output
TestEveryReadingIsStampedWhereItsConditionHolds
TestEveryStepTheConditionsReadIsFound
TestEveryChoicePointIsReachedByARecord
TestErrataMatchTheDocument
TestThePolicyVersionMatchesTheDocument
TestPolicyRowsMatchTheDecider
TestChoicePointsMatchTheDecider
TestCoverageCitesTestsThatExist
TestCoverageCitesGoldensThatExist
```

The sharpest of those is `TestEveryReadingIsStampedWhereItsConditionHolds`.
`ERRATA.md` states each reading's stamping condition as a predicate over the
finished record; the gate re-states that predicate in Go and checks every golden
both ways — a record whose condition holds must stamp the erratum, and one whose
condition does not must not. The predicate is written from the document's prose
and never from the stamping code, because a condition transcribed from the thing
it checks is one reading written twice.

The whole gate is `task`, and CI runs exactly `task`: formatting, `go vet`,
golangci-lint, NilAway, `go test -race`, and a coverage ratchet that counts
*uncovered statements per package* and fails in both directions. Here is the test
suite alone, with the timings stripped so the output is stable:

```bash
go test ./... -count=1 2>&1 | sed -E 's/[[:space:]]+[0-9.]+s$//'
```

```output
ok  	github.com/philoserf/ctchargen/chargen
ok  	github.com/philoserf/ctchargen/cmd/ctchargen
ok  	github.com/philoserf/ctchargen/dice
ok  	github.com/philoserf/ctchargen/internal/docsgate
ok  	github.com/philoserf/ctchargen/render
ok  	github.com/philoserf/ctchargen/rules
ok  	github.com/philoserf/ctchargen/traveller
```

## Where the linear order breaks down

Two places, recorded because a reader following the call chain will hit them.

**The decider you pass is not the decider the engine calls.** Reading `chargen`
top to bottom, you meet `r.decide.AttemptCommission()` in `rank.go` and naturally
look for the implementation you handed to `Generate`. It is one hop further out:
`generate.go` wraps every decider in `logging`, which records the answer and
refuses one that was not offered before delegating. That indirection is the whole
point — it is what makes the offered-set check exist in one place rather than
twelve — but the call chain does not show it at the call site.

**"Rank" and "Title" each name more than one thing, and they cross.** Tracing
`assessTitle` means holding `traveller.Rank` (a position in a service's Table of
Ranks), `traveller.Title` (a rank of nobility), `chargen.Character.RankTitle` (the
*name* of a service rank, "Captain"), `chargen.Title` (a struct about nobility),
and `rules.Service.Title(rank) (string, bool)` (which returns a service rank's
name, not a nobility) — all at once. `chargen.Title.Rank` is a nobility while
`chargen.Character.Rank` is a service rank. Each name is defensible where it
stands; together they cost a reader a pause. Recorded as a finding below.

Everything else reads in order: `main` dispatches, `newCharacter` parses flags
into `Inputs`, `Generate` walks the procedure asking a `Decider`, `render`
projects the result. The engine is genuinely linear.

## Findings from this pass

Two things surfaced while tracing the code that a reader of this document should
not have to rediscover. Both are recorded in full under `.issues/`.

| #   | Severity | Issue                                                             | Primary location                       |
| --- | -------- | ----------------------------------------------------------------- | -------------------------------------- |
| 1   | low      | `rank-and-title-each-name-more-than-one-thing-across-the-exported-api` | `chargen/character.go`, `render/json.go` |
| 2   | low      | `dice-package-doc-says-two-properties-over-a-list-of-three`       | `dice/dice.go:8-18`                    |

**Total: 2 issues (0 critical, 0 high, 0 medium, 2 low)**

Neither is a defect. The first is a naming overlap that costs a reader a pause
and is cheaper to settle before `v1.0.0` freezes the record and the exported
packages; the second is a count in a comment left behind by the change that added
`Among`.

A separate pass over the same tree (`code-theory`) filed four more findings,
including two about `docs/character.schema.json` and the `v1.0.0` freeze that
bear on the same surface as finding 1 above. See `THEORY.md` and `.issues/` for
those; they are not counted here.

