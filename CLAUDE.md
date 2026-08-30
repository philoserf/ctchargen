# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`ctchargen`: a Go CLI that generates rules-accurate Classic Traveller
characters (Books 1–3, © 1977 text), sibling to `philoserf/t5chargen`.

**Status: all four PRD milestones complete** — the full pp. 4–25
procedure across all six services, aging/medical crises, mustering out
with ships and titles, interactive and auto modes, `batch`, and `replay`
verification.
`docs/PRD.md` is the v1 contract — read it before doing any work here.
`docs/COVERAGE.md` maps rules to implementation; `docs/ERRATA.md` holds
the recorded readings; `docs/POLICY.md` is the auto-mode decision table.
Golden fixtures move only via `task goldens`, never by hand.

## Commands

```sh
task          # the full gate: check (modernize + gofumpt + prettier + vet + golangci-lint + nilaway) + test
task fmt      # format Go (gofumpt -extra) and JSON/Markdown (prettier)
task test     # go test -race ./...
task deps     # install the toolchain (brew bundle + pinned Go tools)
task hooks    # install the tracked pre-push hook (runs `task`)
go test -race ./cmd/ctchargen -run TestRun   # a single test
```

CI (`.github/workflows/ci.yml`) runs exactly `task` — never add checks to CI
that the local `task` gate doesn't run, and never add a tool to the gate
without also installing it there. golangci-lint runs with `default: all`
and a curated disable list (`.golangci.yml`); fix findings rather than
adding disables.

NilAway is part of the gate and is upstream-flagged as experimental, so it
does report false positives. Answer one with a real nil check where that
is honest — `run`'s `civilian || svc == nil` is the worked example — and
reach for a suppression only when no such check exists. Note that it and
golangci-lint can want opposite things: a nil-means-absent return that
satisfies NilAway trips `nilnil`, and the sentinel error `nilnil` suggests
would put a declined draft in the error channel, which the PRD forbids.

**`fieldalignment` findings are declined, deliberately.** Running it
reports ~27 struct-layout wins; do not act on them. `encoding/json`
marshals in declaration order, so for `Character`, `Event`, `Benefits`,
`Skill`, `Inputs`, and `Death` the field order _is_ the record's key
order, and for the `service` and `charts` types it is the shape of the
embedded rule tables. Reordering any of them rewrites every key in every
record: replay-breaking (the comparison is byte-for-byte), an
`engine_version` bump, all 26 fixtures moved, and both hand-maintained
`docs/character.*.json` examples plus the schema's documented order out
of step. The whole prize is ~96 bytes per `Character` on a program that
allocates one per generated character and immediately serializes it — under
2 KB on a `batch --count 20`. Field order here carries meaning the
analyzer cannot see; the provenance block leads the record because that is
what a reader and the schema expect first. This is why `govet` runs
without it: `fieldalignment` is opt-in within `govet`, not accidentally
missing.

## Authority model — the most important rule

Rules come **only** from the three held PDFs in
`~/Documents/Traveller/Classic/` (FFE reprints of the © 1977 text): Book 1
_Characters and Combat_ (chargen pp. 4–25), Book 2 _Starships_, Book 3
_Worlds and Adventures_. Books 2–3 are consulted only where Book 1 points at
them.

- **Never implement a Traveller rule from memory.** Training-data Traveller
  is mostly the 1981 revision and later editions; the held 1977 page governs
  even where it famously differs (e.g. survival failure is death, p. 5).
- Every implemented rule carries a printed-page cite. Where the text is
  ambiguous or silent, the chosen reading goes in `docs/ERRATA.md` with the page
  cite — never applied silently. The PRD already enumerates several
  (draft "may vs. must", per-term step order, rank-skill timing, medical
  crisis gaps).
- Page N as printed is PDF page N+6 in Book 1, N+5 in Books 2 and 3.
- Everything else in that collection (Consolidated Errata, Starter Edition,
  Books 4+, …) is out of authority.

## Clean room

Do **not** read, import from, or copy the sibling repos `philoserf/t5chargen`,
`philoserf/t5`, or `philoserf/traveller` unless the user explicitly asks. The
_contracts_ t5chargen proved (replay/provenance, event log, the
COVERAGE/ERRATA/POLICY documents) are adopted; its code is not.

## Architecture (planned — see PRD for detail)

- Packages: `dice`, `chargen` (engine; all choice points go through a
  `Decider` interface so interactive and auto modes share one procedure),
  `service` (data-driven definitions of the six services), `render`,
  `cmd/ctchargen`.
- Tables/thresholds/labels are embedded data files (`go:embed`) with
  load-time validation; procedural mechanics are typed Go. No rules language.
- The JSON character record is the source of truth; Markdown sheets are
  renders of it. Every record carries full provenance (seed, versions,
  inputs, event log) and must replay byte-identically — dice-stream
  consumption order is load-bearing, so procedure-order changes are
  replay-breaking.
- Death is an outcome, not an error: a killed character gets a complete
  record; nothing rerolls silently.
- Testing: golden fixtures per service (regenerated, never hand-edited),
  replay round-trips, schema validation, property tests on dice.

## Project documents

All project documents live in `docs/`:

- `docs/PRD.md` — the contract; milestones live there.
- `docs/ERRATA.md`, `docs/POLICY.md`, `docs/COVERAGE.md`,
  `docs/character.schema.json`.
