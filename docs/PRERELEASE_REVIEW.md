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
| Every schema-constrained field reaches the schema through a record | `medical-crisis-survivor` fixture (Navy 231) carries `age_months: 6`, the last field whose declared bound faced no real value                                                                                                                                |
| The documents cannot drift from the code unnoticed                 | `TestErrataIDsMatchTheDocument`, `TestErrataEntriesCiteAPage`, `TestPolicyDocumentStatesTheStampedVersion`, `TestPolicyTableCoversEveryChoicePoint`, `TestCoverageNamesRealTests` — each verified to fail on a real drift, not merely to pass                |
| A new untested statement cannot ship green                         | Per-package uncovered-statement ceilings in `Taskfile.yml`, verified to trip on one added guarded branch                                                                                                                                                     |

## Open

None. The bar is met; v1.0.0 can be tagged.

## What the gates do not catch, and a correction

An earlier revision of this file claimed the ERRATA-id gate "would have
caught E009." That was wrong, and the claim is withdrawn.

E009's actual history was: the recursive reading of the 12-past-the-cap
rule was implemented in `reenlistment` from the beginning, with no ERRATA
entry and no stamp. A gate comparing _ids stamped in code_ against _ids in
ERRATA.md_ sees both sets missing E009, finds them equal, and passes. What
found it was reading p. 7 and noticing that "an additional term" is
singular.

So the honest statement of what the gates buy:

- **They catch** an id stamped with no entry to explain it, an entry no
  code path can stamp, an entry with no page cite, a policy version the
  document and the engine disagree on, a choice point with no policy row,
  a policy row for no choice point, and a COVERAGE row naming a test that
  no longer exists.
- **They do not catch** a reading applied with neither an entry nor a
  stamp. That is invisible to any comparison of the code against the
  documents, because it is absent from both. Only reading the held page
  finds it, which is what the claim-by-claim review of 2026-08-30 did and
  what any future rules change still needs.

The same correction applies to the coverage ceiling. A _percentage_ floor
would not have caught the untested E009 stamp either: a guarded branch adds
one covered statement (the condition) and one uncovered (the body), moving
`chargen` from 795/878 to 796/880 — 90.55% to 90.45%, both of which
`go test -cover` prints as **90.5%**. The gate therefore counts uncovered
statements rather than percentages, which is integer-exact and trips on the
first one. This was found by trying to make the percentage version fail and
watching it pass.

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
