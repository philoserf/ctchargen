# PRERELEASE REVIEW

The bar v1.0.0 has to clear, and where it stands. Every line is either
verified with the evidence that verified it, or open with what closing it
costs. Nothing here is aspirational: an item is met or it is not.

No tags exist yet. `ctchargen version` reports `(devel)` until one does —
`runVersion` reads `debug.ReadBuildInfo`, so the tag is what makes that
line mean anything.

## Met

| Bar                                                                | Evidence                                                                                                                                                                                                                                                     |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| All four PRD milestones implemented                                | `docs/COVERAGE.md`; `docs/PRD.md` Milestones                                                                                                                                                                                                                 |
| Every rule of Book 1 pp. 4–25 mapped, with its one exclusion named | `docs/COVERAGE.md` header + table                                                                                                                                                                                                                            |
| Rule data matches the held pages                                   | Verified cell by cell 2026-08-30 against Book 1 pp. 2–25 and Book 3 p. 22: Prior Service Table, Table of Ranks, all 24 skills tables, both muster tables, Aging Table, Rank and Service Skills box, retirement pay table, both weapons lists, Nobility table |
| Every interpretation recorded, cited, and stamped                  | `docs/ERRATA.md` E001–E009; E009 added 2026-08-30 when the pass found it applied silently                                                                                                                                                                    |
| Page cites correct in code, docs, and record text                  | The p. 4 → p. 5 Titles correction (#3) was the last known wrong cite                                                                                                                                                                                         |
| Worked example reproduces                                          | Jamison (p. 25): UPP 779C99, four merchant-ship receipts → 30 years old / 10 years owed, CR 4,000 retirement, two extra muster rolls at rank 5, aging saves at 34 and 38                                                                                     |
| Replay is byte-exact and provenance-checked                        | `TestReplayRoundTrip` over the whole roster; `TestReplayChecksProvenance`; `TestReplayWaivesTheAlgorithmStamp`                                                                                                                                               |
| Records validate against the schema                                | `TestRecordsConformToSchema` over `fixture.All()` and both documented examples                                                                                                                                                                               |
| Schema pinned to the structs both ways                             | `TestSchemaMatchesStructs`                                                                                                                                                                                                                                   |
| Gate green and reproducible                                        | `task`: modernize, gofumpt, prettier, vet, golangci-lint `default: all`, NilAway, `go test -race ./...`; CI runs exactly `task`                                                                                                                              |
| Three code-audit passes cleared                                    | five findings, then seven (#1), then four (#2) — the last found 0 critical, 0 high                                                                                                                                                                           |

## Open

Neither blocks a v1.0.0 tag. Both are listed because leaving them
unrecorded is how they get forgotten.

### 1. `age_months > 0` reaches the schema through no record

`docs/character.schema.json` bounds `age_months` at `maximum: 11`, and no
record in the suite carries a non-zero one — only a medical-crisis
**survivor** accrues months, and the one crisis fixture dies. The engine
side is covered (`TestMedicalCrisis` drives both branches;
`TestAddAgeMonths` pins the 12-month carry), and `render` covers the sheet
with a hand-built record it documents as a workaround
(`render/render_test.go`). Only the schema bound is unpinned.

Closing it means finding a seed whose character survives a crisis and
adding a `medical-crisis-survivor` fixture: one `chargen/testdata` file,
two `render/testdata` files, and a line in `internal/fixture`'s roster
comment. No engine change, so no version bump.

### 2. The gate has no claim-rot check

Three defects in three passes were documents drifting from code, not code
being wrong:

- E009 was implemented and unstamped, contradicting the rule that every
  reading is stamped (fixed #3).
- `COVERAGE.md` named a test that no longer described the row's coverage
  (fixed #2, #3).
- A wrong page cite propagated from a doc into the record text itself
  (fixed #3).

Only the third was found by reading the book. The first two are
mechanically checkable and should be tests, not vigilance:

- **ERRATA-id gate.** Every id passed to `stampErratum` or listed in
  `appliedErrata` has a `## Exxx` heading in `docs/ERRATA.md`, and every
  heading is reachable from the engine. This would have caught E009.
- **POLICY-version gate.** `docs/POLICY.md`'s stated `policy_version`
  equals the `PolicyVersion` the engine stamps.
- **COVERAGE test-name gate.** Every `` `TestXxx` `` named in
  `docs/COVERAGE.md` exists in the test tree.

A related hole, found the same day: a new conditional shipped through a
fully green `task` with zero coverage (the E009 stamp, before its test was
written). The gate has no coverage floor, so nothing but review catches an
untested branch.

## Deliberately not in v1

From `docs/PRD.md` Non-goals, restated here so a reader of this file does
not mistake them for gaps: anything outside Books 1–3, later printings'
rules, psionics, experience and self-improvement, world generation,
combat, trade, encounters, and non-human characters. The in-play skill
effects of Book 1 pp. 14–20 are excluded for the same reason and are named
in `docs/COVERAGE.md`.

## Versioning at the tag

`EngineVersion` (0.6.0), `SchemaVersion` (2), and `PolicyVersion` (3) are
independent of the release tag and stay where they are. They track record
behaviour, not releases: `EngineVersion` changes when generation
behaviour or **any event's text** changes — `Replay` compares whole
`Event` values — and bumping it in sympathy with a release would move
every golden and break replay of every existing record for nothing.
