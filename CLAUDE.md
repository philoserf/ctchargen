# CLAUDE.md

Guidance for Claude Code in this repository. `docs/PRD.md` was the v1
contract and is delivered and historical; this file carries the authority
model live, and adds what an agent needs and a human reader does not.

## What this repo is

A Go CLI that generates rules-accurate Classic Traveller characters from
Books 1–3 (© 1977 text, FFE reprints). The repo was emptied at `41a213a` and
rebuilt from `docs/PRD.md`.

**Current state: the v1 contract is delivered and the prerelease review is
done.** All six services run, every table of pp. 4–25 is lifted and consulted,
the book's own worked character replays against the engine,
`docs/character.schema.json` describes the record with every golden validated
against it, characters are written to files, generated in batches and read
back, interactive mode walks the procedure a question at a time, and
`docs/PRERELEASE.md`'s three passes closed with no finding open.

**What governs the work now is what Classic Traveller referees report about
using the tool**, not a remaining milestone — there are none. Issue #26 is the
first such report and named the gap plainly: the engine earned trust, and
everything around it was still alpha. **That report is now answered in full.**
Every command describes its own flags, the sheet names whose service it is and
prints the seed, a session that stops offers the way back in, the line the tool
tells you to paste is quoted, and a release carries binaries a referee can
download without a Go toolchain. `v1.0.0-beta.2` is the current release.
`v1.0.0-alpha.1` and `v1.0.0-alpha.2` predate the rebuild at `41a213a` and
install a different tool; their notes say so.

**The prerelease code audit (#27) is answered too, and the backlog is empty.**
Its six principle violations, its medium and low findings, and the five open
questions it raised are all closed; so is every milestone. What governs the
work now is what a referee reports, and nothing is queued behind it. The page
and the clean room govern how a report gets answered.

**What `v1.0.0` means, decided in #74.** Two things, and both must hold: a
referee can trust what the tool prints at the table, **and** the record is
frozen and supported. The `v1` label carries which issues that covers today —
this paragraph states the bar and never the list, because a list here goes
stale every time one closes.

**The record freezes at `v1.0.0`, and is free to move before it (#62).** Until
that tag, a record written by one build is not promised to be readable by the
next; from it, the shape is a public contract and a build that changes it says
so in the record itself. `record` is the field it says it in: a required
integer set in `project()` beside every other field, so it is present on every
record, and counting the shape and nothing else — `build` names the writer and
`ruleset` names the source text, and neither says what the file looks like.

**`additionalProperties` is open at the root, and closed at every object
inside it (#113).** A later build may add a top-level section, and a reader
holding this schema will load the record and render what it understands — which
is what `render` already did for an unknown event kind, printing
`(unknown event kind %q)` rather than failing, while the schema contradicted it.
A key added inside an existing object is still refused: the fourteen nested
objects are what a sheet is rendered from, and pinning them is what keeps the
schema worth validating against. #76 decided that the schema should open and
left the scope open; #113 settled it at the root and no further.

**The schema opened in `v1.0.0-beta.2`, ahead of the tag, which is not the
same as opening ahead of the freeze.** A file changes before a tag rather than
at it, and a last beta is where the shape `v1.0.0` will freeze gets put in
front of a referee first. What dates to `v1.0.0` is the **freeze** — the
promise that a record written by one build reads in the next — and that is
still unmade.

**`record` is 1 until `v1.0.0` and counts from there (#96).** Not from every
change: the record moves freely before the freeze, so a build that changes the
shape before it leaves the number alone, and two pre-release shapes that both
say 1 are what that freedom costs. Do not bump it to mark a pre-v1 change —
that is the reading #96 declined. The **schema** accepts any shape from 1
upward, which is a different statement: a reader must be able to validate a
record from a later build rather than refuse it before looking at its contents,
so `minimum` is what the schema says and `TestThisBuildWritesShapeOne` is what
holds this build to 1.

**Windows binaries ship, and no document mentions Windows.**
`release.yml` builds `windows/amd64` and `windows/arm64` alongside the other
four and attaches them like the rest. Nothing else names the platform — not
the README, not the help, not a release note, not a reply on an issue. The
silence is the decision and not a gap in one: an artefact sitting on a
releases page promises nothing, while a sentence saying either "supported" or
"not supported" starts a conversation this repo has chosen not to have. A
Windows defect is not a referee's finding in the sense that governs the work
here. So: do not add the README line, and do not delete the two matrix rows
for want of a document naming them.

**The paragraphs above the `v1.0.0` ones** are the thing here that goes
stale on its own — what is delivered, and which release is current. Correct
them when the work moves rather than letting the file describe a tree that no
longer exists. The rules below them do not go stale: each states a bar,
a promise or a scope, and holds until a decision changes it. They are not
counted here, because a count is one more thing to get wrong — this file has
already had to correct one from two to three.

## Authority — read this before implementing any rule

1. **Never implement a rule from memory.** Training-data Traveller is mostly
   the 1981 revision and later editions. The held © 1977 text governs even
   where it differs — most notoriously, survival failure is death (Book 1
   p. 5), with no "injured instead" option.

2. **Never read a table out of `pdftotext`.** The reprints' embedded font
   substitutes glyphs and the substitutions look like data: on Book 1 p. 9 the
   "—" cells of Mustering Out Table 1 extract as the digit **4**, "Travellers'
   Aid" extracts as `Travellers9`, and a minus sign goes the same way, so an
   `N−` target reads as `N3`. A run that trusts the extraction gives a Scout a
   seventh benefit he does not have.

   Read pages **visually** instead — the `Read` tool with a `pages` range on
   the PDF. This is the sanctioned exception to preferring Bash for file
   reads.

   **One page at a time when the page carries a table.** A range renders each
   page smaller, and the prerelease review lost a cell to it: in a six-page
   read, Merchants row 3 of the Service Skills Table came out `Electronic`
   where p. 11 prints `Blade Cbt`. Re-reading that page alone was right at
   once. Ranges are fine for prose; a table gets its own call.

   **And the visual read is not the whole defence.** The substitution reaches
   the rendered image too, not only the extracted text: p. 23's Rank and
   Service Skills box shows `Rifl3-1` in the page image, and Rifle-1 is what
   it means. What catches that is a semantic cross-check — no skill or weapon
   named `Rifl3` is defined anywhere in the three books, and Rifle is on the
   p. 13 gun list. Read every extracted name back against the description
   headings and the weapon lists, and treat one that resolves to nothing as a
   broken glyph rather than a new name.

3. **Transcribe every table twice.** Once into the embedded data, once into
   the `rules` tests, both from the same visual reading pass. The second
   transcription is the check the font trap needs. Retyping the second copy
   from the first checks nothing.

4. **Every implemented rule carries its printed-page cite**, in the code and
   in `COVERAGE.md`.

5. **Where the text is silent or ambiguous, the reading goes in
   `docs/ERRATA.md`** with its page cite and its stamping condition, and is
   named on every record it governed. Never applied silently.

6. **A number that indexes a printed table is data; a bare procedural bound is
   a constant with its cite** (#65). The aging table's last printed term
   indexes a column, so it belongs in `aging.json` and not in Go; the three-roll
   cash cap and the rank thresholds bound a procedure and index nothing, so they
   belong beside the code that applies them. The rule is stated once here rather
   than argued per number, and it moves numbers both ways — a number on the
   wrong side of it is a finding, not a preference.

### Page offsets

Printed page N is PDF page **N+6** in Book 1, **N+5** in Books 2 and 3.

- `~/Documents/Traveller/Classic/Book 1 Characters and Combat.pdf` —
  authoritative for the whole procedure; chargen is printed pp. 4–25 (PDF
  10–31).
- `~/Documents/Traveller/Classic/Book 2 Starships.pdf` — Type S p. 18, Type A
  p. 19, consulted only because Book 1 p. 22 points there.
- `~/Documents/Traveller/Classic/Book 3 Worlds and Adventures.pdf` — Nobility
  p. 22, consulted only because Book 1 p. 5 points there.

Everything else in that directory is **out of authority**: Books 4+,
supplements, the Starter Edition, The Traveller Book, JTAS, and the
_Consolidated Errata_ PDF. Do not open them for rules.

## Precedence

Three sources can answer a rules question. They rank, and a lower rank never
overrides a higher one:

1. **The held page.** Book 1, and Books 2–3 where Book 1 points at them. If
   the page settles it, nothing else is consulted — not memory, not an
   erratum, not the past.
2. **`docs/ERRATA.md`, the readings decided here.** Each was decided against
   the page and carries its cite, so it governs wherever the page is silent
   or self-contradictory.
3. **The errata of the implementation removed at `41a213a`.** Last resort,
   and only where 1 and 2 are both silent — that is, a gap this pass has not
   yet found.

A third-rank answer is never applied as it stands. Take it as a **pointer to
a page**, go read that page, and decide the reading here: it earns an id in
`docs/ERRATA.md`, its own rationale, and its own stamping condition, at which
point it is a rank-2 reading like any other and its origin is history. A
reading inherited unread has an authority it never earned — that is the whole
reason the ordering puts it third rather than first.

## Clean room, this repo's own past included

Sibling repos are not imported from or copied; consult them only when
explicitly asked. **The same rule governs this repository's own history at
and before `41a213a`**, with the single exception the ordering above carves
out. That tree held a complete implementation: Go code, tests, table
transcriptions, a policy table, goldens and a verified worked-example
reproduction. **None of that comes forward at any rank.** Do not read the old
code, tests, transcriptions, fixture roster or policy table — a transcription
re-used is the font trap uncaught, and a policy row re-used is a decision
never made.

What may be consulted, and only at rank 3 above, is that tree's **errata** —
its list of places the book is silent, and the readings it reached there.
`docs/PRD.md` already carries the list; the readings are the fallback.

## Documents and what each governs

| File                         | Governs                                                                                                                                                                                        |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `docs/PRD.md`                | Nothing now. The delivered v1 contract, kept as the record of why the tree has this shape.                                                                                                     |
| `docs/ERRATA.md`             | Every recorded reading, with its page cite and stamping condition.                                                                                                                             |
| `docs/POLICY.md`             | The `--auto` decision table: one row per `Decider` method.                                                                                                                                     |
| `docs/COVERAGE.md`           | Every implemented rule of pp. 4–25, mapped to its page cite, its implementation and its test. A rule with no row is not implemented; a row with no test is a defect.                           |
| `docs/character.schema.json` | What this build writes, in draft 2020-12, with a generated minimal and complete example beside it. A description of the output kept honest by CI — never a promise to records already written. |
| `docs/PRERELEASE.md`         | One section per tag: what it ships with open and why, and the review that preceded it where there was one. A finding is recorded before it is fixed.                                           |
| `CLAUDE.md`                  | This file.                                                                                                                                                                                     |

The documents are held to the code in both directions once code exists: every
erratum id stamped in code resolves to an `ERRATA.md` heading and every
heading is reachable by some path; every `POLICY.md` row names a `Decider`
method and every method has a row; every `COVERAGE.md` row names a test that
exists. Each gate is verified by breaking it.

## The shape of the tree

The packages, and the arrows point one way: `traveller` imports none of the
others; `rules` and `chargen` import `traveller`; `render` imports the record.
A domain type that needs to know about dice or JSON is in the wrong package.
Each package carries its own contract in its doc comment, which is the current
statement of it; what follows is only the map.

**`traveller`** is the domain, and imports nothing else here. It holds the
alphabets Book 1 prints, the values it works in, and the sums that say
"exactly one of". The sums that are folds carry a cases interface — adding a
case adds a method and every implementation stops compiling. The plain enums
are held by the `exhaustive` linter instead, which is the gate and not the
compiler, so dropping the linter drops the guarantee.

**`rules`** holds every table of pp. 4–25 as `data/*.json`, embedded, and lifts
it into `traveller` values once through `sync.OnceValues`. **The lift is the
validation**: a cell naming no service, no characteristic, no benefit any row
prints, or a target in a notation pp. 2–3 do not use fails there, so a table
that will not lift is a build defect that surfaces immediately rather than a
runtime condition some path might reach. Each table is transcribed twice —
into `data/`, and into `transcription_test.go` and `tables_test.go` — from one
visual reading. That is authority rule 3, and `rules` is where it lives.

**`dice`** draws from a seeded PCG and never judges a throw against a target;
that is `traveller.Target`. One seed reproduces one character, which is what
`--seed`, `--answers`, `batch` and every golden rest on.

**`chargen`** walks the procedure. `Generate(Inputs, Decider, ...Option)` is
the entry point: it throws through a `Roller` and asks through a `Decider`, and
it cannot tell auto mode from a person from a test. `Decider` has one method
per choice point — that is what closes the set, and what `docs/POLICY.md`
carries a row apiece for. `Policy` is the `--auto` implementation: total,
deterministic, and a pure function of what it is handed. The run writes an
event log as it goes, which is what `--history` prints and what an observer
watches live.

**`render`** projects the finished record — `JSON`, `JSONLine`, `Sheet`,
`Transcript`, `EventLine` for a live watcher — and reads one back in
`decode.go`. Where the domain shape and the wire shape disagree, the codec
absorbs it and the domain type keeps its shape.

**`internal/docsgate`** has no non-test code. It is the one package allowed to
know about `traveller`, `chargen` and the documents at once, and it holds
`ERRATA.md`, `POLICY.md` and `COVERAGE.md` to the code in both directions. It
reads the checked-in goldens rather than generating records, so it reads what a
referee would be handed.

**`cmd/ctchargen`** is the subcommands and their flag sets. It splits the data
channel from the asking channel: the record, the sheet and the transcript to
stdout, every question, count and warning to stderr — so
`ctchargen new --seed 7 | jq` pipes a record and not a conversation.

## Commands

`task` is the gate and the default. `task --list` carries the current set;
narrower than a task, in a loop:

```sh
go test ./chargen                     # one package
go test ./chargen -run TestGoldens    # one test
go test ./chargen -regenerate         # rewrite the goldens
go run ./cmd/ctchargen new --auto --seed 145 --sheet
```

- **A package that passes on its own can still move the gate.** `-race`,
  `-coverpkg` and the ratchet run only under `task`, and `-coverpkg` is what
  makes the profile count code exercised across package lines — render driven
  by chargen's goldens, the engine by the command's. Run `task` before calling
  a change done.
- **`task fix` is not a safe blind operation here**, on a repository that
  quotes a primary source. Two of its fixers have already rewritten this tree
  incorrectly: `godot` appended a period after a closing quotation mark in
  eleven doc comments, and `dupword` deleted a word from a verbatim quotation
  of Book 1 p. 3. Both are configured off that behaviour now. Read the diff.
- **`go test ./chargen -regenerate` rewrites more than `chargen/testdata/`.**
  The same fixtures write `docs/character.minimal.json` and
  `docs/character.complete.json`, so the schema's published examples cannot
  drift from what the engine emits. `docs/character.schema.json` itself is not
  regenerated — it is written by hand and validated against.

## Once there is code

- **The gate is `task`** — `go mod tidy -diff`, `go vet`, golangci-lint
  (which is where gofumpt runs, so there is one definition of formatted),
  NilAway, `go test -race`, and the coverage ratchet. CI runs exactly `task`.
- **A ratchet failure is usually not lost coverage.** `coverage.ratchet` holds
  each package's count of uncovered statements, and a blank line splits a
  coverage block — so a `wsl_v5` reflow or an extracted helper moves the counts
  without changing what the tests reach. Read the named packages first: a
  reflow or a refactor is answered by `task ratchet:update`; anything else
  stopped being covered. It fails in both directions, because a number that has
  fallen is a ratchet that has stopped holding.
- **The toolchain is deliberately unpinned.** A red gate on untouched code is
  the signal working. Answer the finding; do not pin a tool to silence it.
- **Dice-stream consumption order is load-bearing.** The die is `IntN(6) + 1`
  and a 2D throw is two of those in sequence. A throw the procedure does not
  make consumes nothing. Changing the order changes every seeded character;
  that is an ordinary change, not a breaking one — regenerate the goldens and
  read the diff.
- **Goldens are regenerated, never hand-edited.**
- **A new invariant is not done until a deliberate mutation has been shown to
  kill it**, and the failure names what was broken.
- **An erratum's stamping condition is machine-checked, not just reachable**
  (#66). `ERRATA.md` states each as a predicate over the finished record, and a
  gate holds what a record stamps to what the document says it should. Write the
  predicate from the document's prose, never from the stamping code: a condition
  transcribed from the thing it checks is one reading written twice, which is
  the same trap rule 3 above exists for.

## Working conventions

- Markdown written here is reflowed by prettier immediately after the write
  (a user-level `PostToolUse` hook, not repo config). The reflow is expected;
  leave it alone rather than reverting it.
- **`docs/character.schema.json` is hand-formatted.** Never run prettier over
  it; the reflow hook is a markdown one and this file is outside it.
- Commits and PRs only when asked. Branch off `main` first; the history is
  squash-merged PRs with sentence-case subjects that say what changed
  ("Correct two typed ranges, name the font trap, and add milestone 0").
- **Cutting a release bumps three things nothing checks**: the README's
  `go install` line, the README's `curl` line — which names the tag twice, once
  in the URL path and once in the filename — and the current-state paragraph
  above. A published document that went stale while nobody noticed is what
  issue #31 was, and none of these is held by the gate.
