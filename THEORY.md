# THEORY.md

The understanding you need to change this system without damaging it. Not a tour
of the files — `ls -R` does that — but an account of what the code believes,
where those beliefs are enforced, and where they rest on nothing but care.

## What the system is for

This is a transcription problem wearing a code generator's clothes.

Classic Traveller's character generation, Book 1 pp. 4–25 of the 1977 text, is a
procedure a person walks with paper and two dice: roll six characteristics,
attempt enlistment in one of six services, then survive terms of four years each
— surviving, maybe being commissioned, maybe promoted, training on one of four
skill tables, throwing to reenlist, and aging — until the service ends by choice,
by refusal, or by death. Then muster out, take benefits off two printed tables,
and assess nobility.

Any competent programmer could implement that from memory in an afternoon. The
result would be wrong, and this repository's entire shape is organised around why.

The wrongness has two sources, and the code fights both.

**The rules in your head are the wrong rules.** Training-data Traveller is the
1981 revision. Under that revision a failed survival throw injures you; under the
1977 text it kills you. `CLAUDE.md`'s Authority section opens on this and the code
carries it through: `traveller.KilledBySurvivalThrow` is a case of the `Departure`
sum, not an error path, and `chargen.Character` is documented as "complete whether
it ended in mustering out or in death." Death is an outcome the tool prints, and
`batch` counts corpses rather than rerolling them.

**The pages in front of you lie under a text extractor.** The FFE reprints embed a
font whose glyph substitutions look like data. On p. 9 the em-dash cells of
Mustering Out Table 1 extract as the digit `4`; "Travellers' Aid" extracts as
`Travellers9`; a minus sign becomes a digit, so an `N−` target reads as `N3`. Each
of those is a plausible number in a table full of numbers. The defence is
threefold and every layer is visible in the tree: pages are read visually rather
than extracted, every table is transcribed **twice** — once into `rules/data/*.json`
and once independently into `rules/*_test.go` from the same visual pass — and the
lift refuses anything that does not resolve. `traveller.ParseTarget` is
deliberately strict about the sign for exactly this reason, and `rules/parse.go`
accepts both the ASCII hyphen and U+2212 because "someone transcribing p. 11 will
type what he sees."

So: the domain the code models is not "a character" but **a printed procedure and
the readings a person had to make to follow it**. That is why the record is an
audit log first and a character sheet second.

### The vocabulary

- A **service** is one of six columns of the Prior Service Table (p. 10).
- A **term** is four years. It has no upper bound as a type, because a 12 on the
  reenlistment throw forces another one and E003 reads that as recurring.
- A **throw** has something to meet; a **roll** does not. That distinction is a
  type distinction (`ThrowEvent` vs `RollEvent`) and it was won the hard way — the
  two used to be one type and every targetless roll was written to the record as
  `"succeeded": true`, a claim about nothing (#50).
- An **erratum** is a recorded reading, `E001`–`E015`, made where the page is
  silent, ambiguous, or contradicts itself. Every one is named on every record it
  governed.
- A **choice point** is a place the procedure asks a question with more than one
  legal answer. There are exactly twelve.

## The organizing ideas

### 1. Exhaustiveness has two strengths, and the code knows which it is using

`traveller/doc.go` states this outright, and it is the single most load-bearing
idea in the tree:

- **Sums** — `Enlistment`, `TableResult`, `BenefitRow`, `Departure`, `Event`,
  `WeaponBenefit` — are sealed interfaces with a `Fold(cases)` method and an
  unexported seal. Adding a case adds a method to the cases interface, and _every
  implementation stops compiling_. That is the compiler, and nothing turns it off.
- **Plain enums** — `ServiceName`, `Characteristic`, `SkillTable` — are checked by
  the `exhaustive` linter. That is not the compiler. Dropping the linter drops the
  guarantee.

Go's type switch over an interface promises nothing, and the code says so. The
practical consequence: **when you add a case to a sum, follow the compiler
errors** — they are the complete list of places that must learn about it, and they
span packages (`chargen/enlist.go`'s `applyResult`, `chargen/muster.go`'s
`applyBenefit` and `takeWeapon`, `render/fold.go`'s codecs). When you add a value
to an enum, the compiler will not help you; grep and trust the linter.

The `Decider` interface is the same idea applied to questions. Twelve methods, one
per choice point, each typed to its own alphabet. `//nolint:interfacebloat` sits on
it with the reason: "one method per choice point closes the set." Spelling it as
one method over a label would lose exactly that.

### 2. A fold that cannot fail still returns an error, and the code says why each time

You will see this pattern constantly:

```go
// A fold over a codec cannot fail: every case only assigns.
_ = c.Enlistment.Fold(&codec)
```

The `error` return exists because `Fold` is one signature serving cases that _can_
fail (applying a result rolls dice and consults a decider) and cases that cannot
(projecting to JSON). Discarding it is correct at the codec sites and wrong at the
application sites. The comment is load-bearing; do not delete it and do not add
error handling that no test can reach.

### 3. Indexing a printed table is what makes a number data

`CLAUDE.md` Authority point 6, and it moves numbers **both ways**. The Aging
Table's last printed term indexes a column of the table, so it lives in
`aging.json` and `rules.Aging.LastPrintedTerm()` reads it from there — because it
is also E014's stamping condition, and holding it as a Go constant would let the
bands move without the term at which a record declares the reading moving with
them. The three-roll cap on Mustering Out Table 2 and the rank thresholds bound a
procedure and index nothing, so they are constants in `chargen/muster.go` and
`rules/rules.go` beside the code that applies them — and `rules.go` records that
they used to be data and the argument for that was wrong: "Being printed is not the
test. Indexing a printed table is."

If you find yourself moving a number, this is the rule you are arguing with. A
number on the wrong side of it is a finding, not a preference.

### 4. The dice stream is the meaning of a seed

`dice` depends on nothing and judges nothing. The die is `IntN(6) + 1`; a 2D throw
is two of those in sequence, first then second; `Among(n)` is `IntN(n)` from the
same stream, used only where the book hands a choice to a player and names no basis
for it. `Stream.Drawn()` exists so a test can pin that **a throw the procedure does
not make consumes nothing**.

Three corollaries you must hold:

- Changing the consumption order changes every seeded character. That is an
  ordinary change — regenerate the goldens and read the diff — but never an
  accidental one.
- A `Decider` gets `Vary`, not `Roller`. A decider handed the whole roller could
  throw dice the procedure never makes, moving a seed's meaning somewhere no page
  governs. `Vary` is `interface{ Among(n int) int }` and that is the only variation
  a decider may take.
- Reproducibility is promised **within** a build and explicitly not across builds.
  `render/provenance.go` prints "the same seed on a different build is a different
  character" onto every sheet that carries a build stamp.

### 5. Nothing decides on its own behalf, and the log is wired in from the first step

`chargen.Generate` takes a `Decider`. Three implementations exist — the auto
`Policy`, the interactive `player`, and the scripted deciders the tests use — and
the engine cannot tell them apart. A method is called **only when more than one
answer is legal**: `reenlist.go`'s `intent` settles a single-option case without
asking, because "a question with one answer is not a choice, and asking it would
put an entry in the generation record that no reader could have decided
differently."

The log is not added afterwards, "because a log that is added afterwards records
what the author remembered." Every event carries a sequence number, and an
`OutcomeEvent` carries `Because` — the sequence number of the throw or choice that
caused it, or zero where it follows from the procedure itself. That causal edge is
what makes the record auditable rather than merely verbose.

### 6. `logging` is the single gate, and it rests on a stated assumption

`chargen/logging.go` wraps every `Decider` — `generate.go` does this
unconditionally, so it is the only way a decider reaches the engine. It must
implement all twelve methods to compile, "so a choice point cannot reach the engine
without reaching the log."

Its `record` method does two jobs: it writes the choice event, and it **refuses an
answer that was not offered**. Checking there rather than at each application site
leaves one definition of "answered outside the offer." But it compares _rendered
names_, not values, because `MusterWeapon` answers with a sum and comparing a sum
against a list needs a fold — so one path would have to be strings whatever the
other eleven did.

That check is sound exactly as long as `String()` is injective on every alphabet
the engine offers. This was true and stated nowhere (#48); it is now stated in the
method's own comment and held by
`traveller.TestNoTwoValuesOfAnAlphabetShareASpelling` — which is itself held by
`TestEveryAlphabetDeclaredHereIsChecked`, a test that reads the package's own
source for `var X = [...]` declarations and fails on one nobody checks. The
data-driven weapon names are the exception, and they are covered from the other
end: `rules.liftWeapons` refuses a weapon listed twice.

This is the tightest reasoning in the repository. Understand it before you touch
`logging.go`.

### 7. The documents are code, and each gate is verified by breaking it

`internal/docsgate` has no non-test code and is the one package allowed to know
about `traveller`, `chargen`, and the documents at once. It holds, by reflection
and regex over the documents' own heading forms:

- every `Erratum` constant ↔ every `### E0NN` heading in `ERRATA.md`, both ways,
  with repeats reported before they are compacted away;
- every `Decider` method ↔ every `### \`Method(\``row in`POLICY.md`, both ways;
- every `ChoicePoint` ↔ every `Decider` method;
- `chargen.PolicyVersion` ↔ the version `POLICY.md` names;
- every `COVERAGE.md` row's cited test and golden ↔ files that exist;
- and, in `errata_test.go`, **each erratum's stamping condition as a predicate over
  a finished record** — written from the document's prose, never from the stamping
  code, because "a condition transcribed from the thing it checks is one reading
  written twice."

`CLAUDE.md` adds the discipline that makes these worth having: "A new invariant is
not done until a deliberate mutation has been shown to kill it, and the failure
names what was broken."

## The seams

### The package arrows

`traveller` imports nothing in the module. `rules` and `chargen` import
`traveller`. `render` imports `chargen` and `traveller`. `cmd/ctchargen` imports
all three. Verified against `go list`; the rule in `CLAUDE.md` holds.

`traveller` is the domain — alphabets, values, sums. It does not roll dice, read a
table, or marshal JSON. `rules` holds the printed tables as embedded JSON and lifts
them into domain values, and **the lift is the validation**: a cell naming no
service, no characteristic, no benefit any row prints, or a target in a notation
pp. 2–3 do not use fails at `Load()`, which happens once. A table that will not lift
is a build defect that surfaces immediately, not a runtime condition some path
might reach.

### The record boundary — the thinnest seam in the system

`render` projects a `*chargen.Character` into a flat `record` struct, and this is
where the domain's vocabulary and the wire's diverge. `render/json.go` is honest
about it: "Where the two disagree — a sum that must marshal flat, an array-backed
profile that must marshal as six named keys — the codec here absorbs the difference
and the domain type keeps its shape."

The write side is held by folds: `enlistmentCodec`, `departureCodec`, the event
codec. Add a case to a sum and this package stops compiling.

**The read side is held by nothing.** `render/decode.go` unmarshals into `record`
and deliberately does not rebuild domain types — "rebuilding an `Enlistment` or a
`Departure` back into an interface would give the renderer nothing it does not
already have, and would need a decoder for every sum." Consequently `SheetFrom` and
`TranscriptFrom` work on strings: `sheet.go` compares `ship.Kind ==
traveller.ScoutShip.String()`, `provenance.go` compares
`choice.By == traveller.ByPlayer.String()`. The compiler cannot see those. What
holds them is the round-trip test in `render/roundtrip_test.go` and the schema
tests, and `render/provenance.go` says so plainly of its own flag constants:
"Nothing here is held by the compiler; what holds it is the round-trip test in
`cmd/ctchargen`, which takes the line this file prints and runs it."

If you rename a `String()` on a domain value, the write side moves and the read
side silently does not. This is the seam to be careful at.

`docs/character.schema.json` is the third participant, and the three do not quite
agree — see the findings below.

### The command line

`cmd/ctchargen` is where strings become domain values and where they stop being
strings. `withStrategies` parses the three `--auto` strategies once, at the
boundary: "past here a `Career` is a `Career`, and no part of the engine asks again
whether it is one of three. That is why `chargen` has no `Validate` any more (#41)."

The two output channels are separated deliberately: `out` carries the record, the
sheet, the transcript; `asking` carries everything the tool says to the person
driving it, so that `ctchargen new --seed 7 | jq` pipes a record and not a
conversation. `batch`'s closing tally goes to `asking` for the same reason — "a
summary on standard output would be a line of JSONL that is not JSON."

`run` returns `nil` for help asked for, so help exits 0. Only a bare `ctchargen`
with no arguments is an error.

## What the system is shaped to accommodate

**A new erratum.** Add the constant to `traveller/erratum.go`, add it to `Errata`,
write the `### E0NN` heading in `ERRATA.md` with its page cite and its stamping
condition as a predicate over the finished record, stamp it at the outcome it
governs, and write the predicate into `internal/docsgate/errata_test.go` **from the
document's prose**. The gates will tell you what you forgot in both directions.

**A new choice point.** Add the `Decider` method; the compiler names every
implementation that must learn it. Add the `ChoicePoint` constant and the
`POLICY.md` row; `docsgate` holds the three together. Then read the finding about
the schema's `point` enum, because that part the compiler cannot reach.

**A new `--auto` strategy for an existing question.** Add the constant, the
`String()` case, the entry in the unexported strategy slice, the `POLICY.md` row,
and bump `PolicyVersion` if an existing answer moved. Same schema caveat.

**A new mustering-out benefit or departure kind.** Add the case to the sum and
follow the compiler across `chargen` and `render`.

**A new rules table.** Add the JSON under `rules/data/`, a wire type in `lift.go`,
a lift function that validates every cell against a closed alphabet, and — this is
not optional — an independent second transcription in `rules/*_test.go` from the
same visual reading pass. Retyping the second copy from the first checks nothing.

## What would require rethinking something fundamental

**Concurrency.** `rules.Load()` is `sync.OnceValues` over a mutable `*Rules` shared
process-wide. Generation is single-threaded everywhere today; `batch` is a
sequential loop. Parallelising it would need the ruleset made immutable in fact and
not only by convention — see the first finding.

**Cross-build record compatibility, before `v1.0.0`.** Deliberately unpromised.
`recordShape` is 1 and stays 1 until the tag; #96 explicitly declined the reading
that a pre-v1 shape change should bump it. Do not bump it to mark a change now.

**A `Roller` that never fails a reenlistment throw.** `run.serve()` is an unbounded
loop, and the comment is exact: termination is "a contract on the `Roller`, not a
property of this function." The scripted test that walks past the Aging Table's
last column hands one a _count_ of twelves for precisely this reason (#54). Nothing
enforces the contract.

**Injecting a decider that mutates what it is offered.** The engine's whole
premise is that a decider is a pure function of what it is handed. One site defends
that (`chooseMusterTable` clones); the others do not.

## Where a maintainer who did not understand the theory would cause damage

- **Editing the wording of a log line.** It used to be load-bearing:
  `run.depart` chose the departure type by comparing the reason string against the
  literal `"reenlistment denied"`, so editing the sentence changed the domain
  outcome (#45). Fixed — the departure is now passed in — but the instinct that log
  text is inert is wrong here in general, because `render`'s read side compares
  `String()` output.
- **"Simplifying" a fold into a type switch.** It compiles, and it silently
  discards the only exhaustiveness guarantee the tree has.
- **Adding a `Validate()` to `chargen`.** #41 deleted exactly that apparatus. A
  runtime check for a value a closed type cannot hold is a sign the type is wrong.
- **Restating a precondition in a caller.** #52's lesson: `Retirement.Pay` used to
  return a real-looking zero for four terms and the only thing between that and a
  record was a guard in `run.pension`. A precondition belongs to the function that
  has it.
- **Reading a table out of `pdftotext`, or retyping the second transcription from
  the first.** Both defeat the defence the whole repository is built around.
- **Consulting the repository's own history at or before `41a213a`.** The clean
  room covers it, at every rank but one: only that tree's _errata_ may be
  consulted, only where the page and `ERRATA.md` are both silent, and only as a
  pointer to a page to go read.

## Uncertainties

Marked because you should know which claims to trust.

**Inferred from code alone.** I did not open the Traveller PDFs — they are outside
this repository and outside the review's scope. Every page citation above is
repeated from the code's own comments and `docs/ERRATA.md`. I verified that the
citations are internally consistent and that the gates hold them to the documents;
I did **not** verify any of them against the printed page. If a transcription is
wrong, nothing in this document would catch it, and the twice-transcribed
discipline is the only thing that would.

**`docs/PRD.md` "governs nothing," but code still appeals to it.**
`CLAUDE.md`'s document table says the PRD governs nothing now and is kept as
history. Yet `cmd/ctchargen/render.go` justifies a parse decision with "the PRD's
CLI sketch says so plainly: flags precede the filename," and `new.go` argues from
"Neither flag is in the PRD's CLI sketch." Both are historical rationales rather
than live authority, and I read them that way — but a maintainer could reasonably
read them as the PRD still governing the CLI. I did not file this: nothing is
wrong, and the fix is a wording preference.

**Whether the `v1.0.0` freeze is meant to cover the strategy vocabulary.**
`CLAUDE.md` says the _record's shape_ freezes. The record contains
`inputs.career`/`skills`/`muster` as closed enums, so growing the `--auto`
vocabulary is a shape change under a literal reading. Whether that is the intended
consequence or an unnoticed one I genuinely cannot tell from the tree; the finding
below asks rather than asserts.

**The `Roller` termination contract is documented and unenforced**, and I take that
as deliberate rather than as drift — the comment names the failure mode and the one
test that could hit it works around it explicitly. A reader who wanted a hard bound
could reasonably disagree. Not filed.

**Test coverage of `--survivors` under adversarial policies.** I read
`survivors_test.go`'s structure but did not trace every case. The `survivorAttempts
= 100` bound is argued from a referee's measurement of 74 deaths in 100 under the
default strategy; I did not re-measure.

**One consistency I checked and it held.** `CLAUDE.md` claims "fourteen nested
objects" carry `additionalProperties: false`. The schema has fifteen
`additionalProperties` sites: one `true` at the root and fourteen `false` inside.
The count is right.

## Index

| #   | Severity | Issue                                                                        | Primary location                                            |
| --- | -------- | ---------------------------------------------------------------------------- | ----------------------------------------------------------- |
| 1   | medium   | `weapon-list-handed-to-deciders-is-the-shared-singleton`                     | `rules/rules.go:174`, `chargen/enlist.go:172`               |
| 2   | medium   | `schema-refuses-the-unknown-event-kind-its-own-description-cites`            | `docs/character.schema.json:6,225-237`, `CLAUDE.md:52-58`   |
| 3   | medium   | `freezing-the-schema-closes-two-extension-points-the-code-treats-as-routine` | `docs/character.schema.json:290-300`, `chargen/decider.go`  |
| 4   | low      | `errsearch-is-distinguished-in-prose-and-nowhere-a-script-can-read`          | `cmd/ctchargen/main.go:18-27`, `cmd/ctchargen/batch.go:203` |

**Total: 4 issues (0 critical, 0 high, 3 medium, 1 low)**

Findings 2 and 3 share a root — the record's closed vocabularies versus the
extensions the code plans to make — and both are materially cheaper to settle
before `v1.0.0` than after, because the schema is what the freeze makes a public
contract. Finding 1 touches the `rules` package's exported API, which the tag
freezes in the other sense.
