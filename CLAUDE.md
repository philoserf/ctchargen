# CLAUDE.md

Guidance for Claude Code in this repository. `docs/PRD.md` is the contract;
this file adds only what an agent needs and a human reader does not.

## What this repo is

A Go CLI that generates rules-accurate Classic Traveller characters from
Books 1–3 (© 1977 text, FFE reprints). The repo was emptied at `41a213a` and
is being rebuilt from `docs/PRD.md`.

**Current state: every milestone of `docs/PRD.md` is complete.** All six
services run, every table of pp. 4–25 is lifted and consulted, the book's own
worked character replays against the engine, `docs/character.schema.json`
describes the record with every golden validated against it, characters are
written to files, generated in batches and read back, and interactive mode
walks the procedure a question at a time. What remains before a tag is a
prerelease review of the whole tool against the four documents.

This paragraph is the one thing here that goes stale on its own; correct it
when a milestone moves rather than letting the file describe a tree that no
longer exists.

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

3. **Transcribe every table twice.** Once into the embedded data, once into
   the `rules` tests, both from the same visual reading pass. The second
   transcription is the check the font trap needs. Retyping the second copy
   from the first checks nothing.

4. **Every implemented rule carries its printed-page cite**, in the code and
   in `COVERAGE.md`.

5. **Where the text is silent or ambiguous, the reading goes in
   `docs/ERRATA.md`** with its page cite and its stamping condition, and is
   named on every record it governed. Never applied silently.

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

| File               | Governs                                                                   |
| ------------------ | ------------------------------------------------------------------------- |
| `docs/PRD.md`      | The v1 contract: goals, domain model, FR1–FR11, determinism, milestones.  |
| `docs/ERRATA.md`   | Every recorded reading, with its page cite and stamping condition.        |
| `docs/POLICY.md`   | The `--auto` decision table: one row per `Decider` method.                |
| `docs/COVERAGE.md` | Every implemented rule of pp. 4–25, mapped to its page cite, its implementation and its test. A rule with no row is not implemented; a row with no test is a defect. |
| `docs/character.schema.json` | What this build writes, in draft 2020-12, with a generated minimal and complete example beside it. A description of the output kept honest by CI — never a promise to records already written. |
| `CLAUDE.md`        | This file.                                                                |

The documents are held to the code in both directions once code exists: every
erratum id stamped in code resolves to an `ERRATA.md` heading and every
heading is reachable by some path; every `POLICY.md` row names a `Decider`
method and every method has a row; every `COVERAGE.md` row names a test that
exists. Each gate is verified by breaking it.

## Once there is code

- **The gate is `task`** — formatting, `go vet`, golangci-lint, NilAway,
  `go test -race`. CI runs exactly `task`.
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
- Package arrows point one way: `traveller` imports none of the others;
  `rules` and `chargen` import `traveller`; `render` imports the record. A
  domain type that needs to know about dice or JSON is in the wrong package.

## Working conventions

- Markdown written here is reflowed by prettier immediately after the write
  (a user-level `PostToolUse` hook, not repo config). The reflow is expected;
  leave it alone rather than reverting it.
- Commits and PRs only when asked. Branch off `main` first; the history is
  squash-merged PRs with sentence-case subjects that say what changed
  ("Correct two typed ranges, name the font trap, and add milestone 0").
