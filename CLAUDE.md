# CLAUDE.md

Guidance for Claude Code in this repository. `docs/PRD.md` is the contract;
this file adds only what an agent needs and a human reader does not.

## What this repo is

A Go CLI that generates rules-accurate Classic Traveller characters from
Books 1–3 (© 1977 text, FFE reprints). The repo was emptied at `41a213a` and
is being rebuilt from `docs/PRD.md`.

**Current state: milestone 0.** The tree holds `LICENSE`, this file, and
`docs/` — nothing else. `go.mod`, `Taskfile.yml`, the CI workflow,
`.golangci.yml`, `.gitignore`, and a README return with milestone 1's first
Go, and not before: the gate cannot hold a milestone it does not yet exist
for.

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

## Clean room, this repo's own past included

Sibling repos are not imported from or copied; consult them only when
explicitly asked. **The same rule governs this repository's own history at
and before `41a213a`.** That tree held a complete implementation with settled
errata, a policy table, goldens and a verified worked-example reproduction.
None of it comes forward.

Do not run `git show`, `git log -p`, or `git checkout` against `41a213a` or
earlier for rules content. The only thing carried forward is the **list of
places the book is silent** — already carried, in `docs/PRD.md` and now
decided in `docs/ERRATA.md`. A reading inherited unread has an authority it
never earned.

## Documents and what each governs

| File               | Governs                                                                   |
| ------------------ | ------------------------------------------------------------------------- |
| `docs/PRD.md`      | The v1 contract: goals, domain model, FR1–FR11, determinism, milestones.  |
| `docs/ERRATA.md`   | Every recorded reading, with its page cite and stamping condition.        |
| `docs/POLICY.md`   | The `--auto` decision table: one row per `Decider` method.                |
| `docs/COVERAGE.md` | Opens with milestone 1's first Go and grows with it — every implemented rule mapped to page cite, implementation, and test. Milestone 2's exit criterion is that it is living. |
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
