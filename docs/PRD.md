# PRD: Classic Traveller Character Generator (Go CLI)

2026-08-29. Status: draft — the v1 contract, before any code.

## Problem

Classic Traveller character generation is a chart-driven prior-service
procedure: six services, enlistment or the draft, survival throws that can
kill the character, commissions, promotions, skills, aging, and mustering
out. Doing it by hand is slow and error-prone. Build a Go CLI that generates
rules-accurate Classic Traveller characters, as a sibling to
`philoserf/t5chargen`.

Ruleset baseline: **Books 1–3 only** — Book 1 _Characters and Combat_,
Book 2 _Starships_, Book 3 _Worlds and Adventures_, as held in
`~/Documents/Traveller/Classic/` (the FFE reprints of the © 1977 text). All
page cites are to those three artifacts and nothing else. Character
generation lives in Book 1 pp. 4–25; Books 2 and 3 are consulted only where
Book 1 points at them (ship benefits, nobility titles).

## User

Mark: solo referee and developer. Secondary: any Classic Traveller referee
needing NPCs in bulk or players wanting a guided prior-service run.

## Goals

1. Generate a complete character per Book 1's character generation chapter
   (pp. 4–25): characteristics, service, terms, skills, aging, mustering
   out — or death, which is a completed generation too.
2. Two modes: **interactive** (player makes each choice) and **auto** (tool
   decides; supports batch NPC generation).
3. Deterministic replay: re-running the engine from the recorded seed and
   choices reproduces the identical character (see Replay and provenance
   contract).
4. Output a character record as JSON (canonical) and a Markdown character
   sheet in the style the book itself summarizes a character (the UPP line
   plus skills and possessions, as in the Jamison example, pp. 23–25).
5. Emit a generation record: the full chronological history — every throw,
   choice, and outcome — embedded in the JSON and renderable as a Markdown
   transcript.

## Non-goals (v1)

- Anything outside Books 1–3: Books 4+ (Mercenary, High Guard, Scouts …),
  supplements, the Starter Edition, The Traveller Book, JTAS, and the
  _Consolidated Errata_ PDF in the same collection — all out of authority.
- Later printings' rules. The held text is the © 1977 text; where later
  printings famously differ (e.g. an optional survival rule), the held page
  governs: survival failure is death (p. 5).
- Psionics (Book 3 pp. 33–42) — in-play material, not part of prior service.
- Experience and self-improvement (Book 2 pp. 40–41) — in-play advancement.
- World generation (Book 3 pp. 1–12), combat, trade, encounters.
- Non-human characters. The rules' own note (p. 8): no gender or race
  requirement exists; the record simply doesn't ask.

## Functional requirements

Rule citations are to the printed page numbers of the three books.

**FR1 — Characteristics.** Roll 2D each for Strength, Dexterity, Endurance,
Intelligence, Education, Social Standing, in that order (p. 4). Store as
numeric values with the UPP hex string derived (base 16, digits A–F for
10–15; p. 8). Initial range 2–12; through play of the procedure values stay
in 1–15 and never exceed 15 (p. 4). All characters begin at age 18 (p. 4).

**FR2 — Enlistment and the draft.** One enlistment attempt into one of the
six services (Navy, Marines, Army, Scouts, Merchants, Other) against the
Prior Service Table's enlistment throw with its cumulative characteristic
DMs (pp. 5, 10). On rejection the character _may_ submit to the draft: one
die, entering the service with that draft number — possibly the very
service that just rejected him (p. 5). P. 5 prints both readings of the
draft: the Enlistment paragraph's "may submit to the draft" and the Draft
paragraph's opening "must submit to the draft." The **may** reading governs
here — declining the draft ends generation with an 18-year-old civilian, a
valid if bare record — and the choice of clause goes in ERRATA.md as an
interpretation. Draftees are not eligible for commission during their
first term (p. 5).

**FR3 — Terms of service.** Each term is 4 years (p. 5). Per term, in Book
1's order: survival throw — failure is death and the record says so (p. 5);
commission attempt (once per term until achieved; not draftees' first term;
not Scouts or Other, p. 6, p. 10); promotion attempt (one per term, only if
commissioned; p. 6); skills (FR4); reenlistment throw, made every term
whether or not the character wants to stay — failure forces him out, a 12
exactly forces him to stay, even past term 7 (pp. 6–7). Ranks per the Table
of Ranks (p. 10). Voluntary service caps at 7 terms; retirement is
available after the 5th term (p. 7). The per-term order is an
interpretation the book forces: the exposition presents survival,
commission, promotion, skills, then reenlistment (pp. 5–6), but the Jamison
example rolls reenlistment before its term's skills (p. 24), and replay
makes the order load-bearing because it fixes dice-stream consumption. The
exposition's order governs; the reading goes in ERRATA.md.

**FR4 — Skills and training.** Eligibility per the Basic Skill Eligibility
box (p. 6): 2 for the initial term, 1 per subsequent term, 1 on commission,
1 on promotion. Each eligibility is one die on one of the four Acquired
Skills tables for the character's own service — Personal Development,
Service Skills, Advanced Education, and the second Advanced Education table
gated on Education 8+ — with the table declared before the die is rolled
(p. 11). Three result kinds (p. 12): characteristic alterations, applied
immediately; weapon expertise, where Blade Combat and Gun Combat require
the specific weapon to be chosen immediately (pp. 11–13, weapons lists
pp. 12–13); and basic skills, accumulating as Skill-1, Skill-2, … with no
cap (p. 13). Rank and service skills accrue automatically, outside
eligibility, per the Rank and Service Skills box (p. 23): Marine Cutlass-1,
Army Rifle-1, Scout Pilot-1, the officers' weapons, the Navy flag ranks'
+1 Social, Merchant 1st Officer Pilot-1. The box's own timing rule — "as
soon as he becomes eligible" — leaves _when_ underspecified for the
service-wide entries (on enlistment is the natural reading); the reading
goes in ERRATA.md as an interpretation, not silently.

**FR5 — Aging.** From age 34 (end of term 4) and each 4 years after, apply
the Aging Table's saving throws and reductions per term as the ages are
crossed (pp. 7, 9) — per term, not batched at muster out; the Jamison
example batches "for simplicity" and says so (p. 25). A characteristic
reduced to zero is a medical crisis: saving throw 8+, survival means
immediate recovery to 1 plus 1D months of age (pp. 7–8). Two gaps in that
rule need recorded readings in ERRATA.md: the page states what happens on
survival and on absent medical care (incapacitation for the rolled months)
but never states the failed throw's outcome, and the printed DM for
attending medical expertise has no referent during solo generation (no DM
applies).

**FR6 — Mustering out.** On leaving for any reason except death (p. 7):
rolls equal to terms served, plus 1 for rank 1–2 or 2 for rank 3+; at most
3 rolls on Table 2 (cash), the rest on Table 1 (material benefits); table
designated before each roll; +1 DM available on Table 1 at rank 5–6 and on
Table 2 with gambling skill (pp. 7, 9). Benefits per the tables (p. 9) and
their definitions (pp. 21–23): cash; +1 characteristic alterations, applied
immediately; passages (High CR 10,000, Middle CR 8,000, Low CR 1,000,
sellable at 90%; pp. 21–22); Travellers' Aid membership, once per character
ever, duplicate rolls wasted (p. 22); weapons, where a repeat receipt may
take the same weapon, a different one, or +1 expertise in a weapon already
received as a benefit (p. 22); and ships. The Free Trader is a Type A
(Book 2 p. 19), received owing ~CR 150,000 monthly for 40 years, each
additional receipt paying off 10 years and aging the ship 10 years — five
receipts is a 40-year-old ship owned free and clear (pp. 22–23). The Scout
ship is a Type S (Book 2 p. 18) in constructive possession: no title, no
sale, no mortgage, duplicates lost (p. 23). Retirement pay, for a 5th-term-
or-later departure from Navy, Marines, Army, or Merchants only: CR 4,000
per year at 5 terms, per the table, +CR 2,000 per term past 8 (pp. 7, 21).

**FR7 — Titles.** Social Standing 11+ may assume the hereditary title
(p. 4); the full range is Book 3's Nobility table, knight/dame at 11
through duke/duchess at 15 (Book 3 p. 22). Assuming it is a choice point;
the record stores the eligibility and the choice.

**FR8 — Character record.** Track name, age, terms served, service, rank,
characteristics and UPP, full skill list with weapon specifics, benefits
(cash, passages, TAS, weapons, ship with its age and remaining payments),
retirement pay, title, and — for the dead — the term and cause. The JSON
record is the source of truth; the Markdown sheet is a render of it. A
deliberate delta from t5chargen: Books 1–3 have no calendar or birthdate
rules, so the record carries age and terms, not a date. That is the books,
not an omission.

**FR9 — Dice engine.** 2D throws with DMs against `N+`/`N−`/exact targets
per the die roll conventions (pp. 2–3), plus the single-die rolls the
procedure uses (draft, skills tables, mustering out, recovery months). All
rolls consumed from a seeded stream and logged for replay. No Flux — that
is a later edition's device.

**FR10 — Generation record.** An ordered event log of the entire
generation: each procedure step entered, each throw (dice, target, DMs,
result), each choice (who decided — player or policy — and what the
alternatives were), each consequence (skill gained, rank gained,
characteristic change, years elapsed, death). Stored as an `events` array
in the character JSON, with monotonic sequence numbers; consequence events
reference the causing throw or choice. Serves audit (verify any character
against Book 1 by walking the log), replay (verification data for goal 3),
and narrative (a readable service-record transcript).

## Replay and provenance contract

Adopted from t5chargen, which proved it:

- Every character JSON carries `schema_version`, `ruleset` (pinned: Books
  1–3, © 1977 text, FFE reprints as held), `engine_version`,
  `policy_version`, `rng` (algorithm + seed), an `inputs` block (including
  any `--service` force), and any applied ERRATA.md deviations.
- RNG: Go `math/rand/v2` PCG, named in the record. Changing algorithm or
  default policy is a version bump.
- Replay re-runs the engine from the recorded seed, inputs, and choice
  events, recomputing every throw; the stored event log is verification
  data, not input. `ctchargen replay character.json` exits non-zero at the
  first mismatch, reporting the diverging event's sequence number.
- `policy_version` is not verified on replay: recorded choices are
  reapplied, the policy is never consulted, so records made under any
  policy replay under any other. `replay --ignore-provenance` waives the
  version match and nothing else.

## JSON conventions

Characteristics stored numeric with the UPP hex string derived and stored
alongside; money as integer credits; age in whole years with terms served.
Skills are name plus level, with the specific weapon where the rules demand
one. Derived values are stored and recomputed on replay. The schema is
`docs/character.schema.json` (draft 2020-12) with a minimal and a complete
example beside it, versioned by `schema_version`, which tracks the shape of
the records the engine writes: a constraint that only narrows the schema to
what the engine already produced is a clarification; one that would
invalidate a record the current engine writes is a bump.

## CLI sketch

```
ctchargen new [--seed N] [--auto] [--service navy] [--name X] [-o file]
ctchargen batch --count 20 --auto [--service ...] [-o dir|file.jsonl]
ctchargen render [--history] character.json
ctchargen replay [--ignore-provenance] character.json
ctchargen version
```

`--service` forces the _enlistment attempt_ only: the throw is still made,
and a failed throw still goes to the draft, which can land anywhere — that
is what one attempt plus a one-die draft means (p. 5). Interactive mode
walks the procedure step by step; auto mode applies the fixed default
policy. `new` writes JSON to stdout unless `-o`; `batch` emits JSONL (or
one file per character with `-o dir`), requires `--auto`, and derives each
member's seed from the base seed + index, recorded in each record. Existing
files are never overwritten without `--force`. Flags precede the filename
(Go `flag` stops at the first non-flag argument). `version` reports the
build and the versions a record stamps, read from the toolchain's embedded
build info.

The auto policy is **total** — it can decide every valid choice point:
service pick, draft submission, commission and promotion attempts, skill
table allocation, weapon picks (including the muster-out
weapon-vs-expertise option), reenlistment intent and when to retire, the
Table 1/Table 2 split and the optional DMs, and title assumption — and
deterministic, tie-breaking by first-listed order in Book 1. The decision
table lives in `POLICY.md`; `policy_version` identifies it in every record.

## Decisions (2026-08-29)

- **Death is an outcome, not an error.** The record is the product: a
  character killed by a survival throw gets a complete record and a
  rendered sheet saying when and where. `batch` emits what it rolled;
  filtering the dead is the caller's business, and no flag rerolls them
  silently.
- **The held printing governs.** Where the © 1977 text differs from the
  1981 revision most people remember, implement the page as held; do not
  adjudicate editions from memory. Deviations and interpretations go in
  ERRATA.md with the page cite, never applied silently.
- **Names**: blank by default, `--name` to supply one. The book's naming
  section is advice, not a table (pp. 4–5); no generator in v1.
- **Clean room.** Sibling repos `philoserf/t5chargen`, `philoserf/t5`, and
  `philoserf/traveller` are not imported from or copied; consult them only
  when explicitly asked. The _contracts_ proven in t5chargen (replay,
  event log, COVERAGE/ERRATA/POLICY documents) are adopted; its code is
  not.

## Architecture notes

- Repo: new `philoserf/ctchargen` (or similar); code does not live in the
  document collection.
- Packages: `dice`, `chargen` (engine; consumes a `Decider` interface for
  all choice points), `service` (data-driven definitions of the six
  services: the Prior Service Table, ranks, the four skills tables, the
  mustering-out tables), `render`, `cmd/ctchargen`.
- Data/logic boundary: tables, thresholds, and labels are embedded data
  files loaded with `go:embed` plus load-time validation; orchestration
  and the procedure's exceptional mechanics are typed Go. No rules
  language.
- Testing: golden character fixtures per service (including a draftee, a
  death, and a civilian who declined the draft), replay round-trips,
  schema validation, property tests on the dice engine. Fixtures move only
  via regeneration, never by hand.

## Milestones

1. Dice engine + characteristics + character record/render, with the
   generation event log wired in from the start (end-to-end walking
   skeleton, one service: Other — no commissions, no ranks, the shortest
   path through the table).
2. All six services: enlistment, draft, survival, commission, promotion,
   reenlistment, ranks, and the four skills tables with weapon choices and
   rank-and-service skills. Exit criterion: a living `COVERAGE.md` mapping
   every step and per-service rule of pp. 4–25 to its page cite,
   implementation, and golden test.
3. Aging, retirement, and mustering out, including the ship benefits'
   Book 2 values and the Book 3 titles.
4. Interactive mode polish; batch mode; replay verification.

## Sources

- `~/Documents/Traveller/Classic/Book 1 Characters and Combat.pdf`
  (authoritative for the whole procedure; chargen pp. 4–25).
- `~/Documents/Traveller/Classic/Book 2 Starships.pdf` (Type S p. 18,
  Type A p. 19 — consulted only because Book 1 p. 22 points there).
- `~/Documents/Traveller/Classic/Book 3 Worlds and Adventures.pdf`
  (Nobility p. 22 — consulted only because Book 1 p. 4 points there).
- Page cites are printed page numbers. In the held PDFs, printed page N is
  PDF page N+6 in Book 1 and N+5 in Books 2 and 3.
- Everything else in the collection — the _Consolidated Errata_, the
  Starter Edition, the facsimile, the _Rules Companion_, the _Guide to
  Classic Traveller_ — is explicitly **out of authority** for this tool.
