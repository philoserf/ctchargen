# PRD: Classic Traveller Character Generator, domain-typed (Go CLI)

2026-08-30. Status: draft — the v1 contract, before any code.


## Problem

Classic Traveller character generation is a chart-driven prior-service
procedure: six services, enlistment or the draft, survival throws that can
kill the character, commissions, promotions, skills, aging, and mustering
out. Doing it by hand is slow and error-prone. Build a Go CLI that
generates rules-accurate Classic Traveller characters.

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
3. Model the procedure in **domain types**, not strings: the six
   characteristics, the four skills tables, the choice points, the throw
   targets, and the table results are closed types the compiler checks. An
   illegal generation should be a program that does not build.
4. Output a character record as JSON and a Markdown character sheet in the
   style the book itself summarizes a character (the UPP line plus skills
   and possessions, as in the Jamison example, pp. 23–25).
5. Emit a generation record: the full chronological history — every throw,
   choice, and outcome — embedded in the JSON and renderable as a Markdown
   transcript, so any character can be audited against Book 1 by walking
   the log.

## Non-goals (v1)

- **Replay verification.** No `replay` subcommand, no byte-for-byte
  re-derivation of a stored record, no verified provenance stamps. This is
  a generator; the record is its output, not a contract with the future.
  See Determinism for what _is_ promised.
- **Cross-version record compatibility.** A record written by one build is
  not promised to be readable, renderable, or reproducible by the next.
  Key order, field naming, and event wording are internal detail, pinned
  only by regenerable golden fixtures.
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

## Domain model

The procedure's alphabet is finite and printed. Every part of it that the
book names is a Go type with a closed set of values, constructed once at
load and thereafter unforgeable. The rule tables remain embedded data
(`go:embed` JSON), but the data are **wire types** that lift into domain
values at load; the lift is the validation, and a table that will not lift
is a build defect that fails on first use.

**Closed alphabets** — enums, not strings:

- `Characteristic`: the six, in rolled order (p. 4). A `Profile` is an
  array indexed by it, so there is no lookup that can miss and no
  "unknown characteristic" error to return.
- `ServiceName`: the six of the Prior Service Table (p. 10), in book order,
  which is also display order and the auto policy's tie-break.
- `SkillTable`: Personal Development, Service Skills, Advanced Education,
  Advanced Education (Education 8+) (p. 11).
- `WeaponCategory`: blade, gun (pp. 12–13).
- `PassageClass`: high, middle, low (pp. 21–22).
- `ShipKind`: scout (Type S, Book 2 p. 18), free trader (Type A, Book 2
  p. 19).
- `ChoicePoint`: the event log's label for a choice, derived from the
  `Decider` methods of FR9 — a rendering of that interface, never a second
  list kept parallel to it.
- `Erratum`: the recorded readings of ERRATA.md, by id.

**Value types** — meaning, not primitives:

- `Target`: a throw target with its modality (`N+`, `N−`, exact; pp. 2–3),
  parsed once when the data lifts, never re-parsed at throw time.
- `Credits`: money, integer credits, distinct from every other int.
- `Age`: whole years plus the 0–11 months only a medical-crisis recovery
  can accrue (pp. 7–8), carrying into years on its own.
- `Rank`: 0–6, with the service's own title from the Table of Ranks
  (p. 10); rank 0 is "not commissioned", not a missing value.
- `Term`: 1–7 (p. 7).
- `Skill`: a name and a level, with the specific weapon where the rules
  demand one ("Dagger-1", p. 25) and the category that demanded the pick.
- `WeaponName`: the one name that does _not_ close at compile time. The
  blade and gun lists are printed data (pp. 12–13), so weapon names lift
  from the embedded lists and are checked against the list for their
  category; the category is the closed type, the name is data.

**Sums** — the places where "exactly one of" is the rule:

- `Enlistment`: enlisted | drafted into a service | declined the draft. The
  civilian is a case of the type, not a nil service with a flag beside it.
- `TableResult`, one row of an Acquired Skills table (p. 12): a
  characteristic alteration | a named skill | a weapon-category choice.
- `BenefitRow`, one row of Mustering Out Table 1 (p. 9): cash | passage |
  characteristic alteration | weapon category | Travellers' Aid | ship |
  nothing (the table's "—" rows).
- `Departure`: discharged | forced out | retired | died.
- `Event`: step | throw | choice | outcome (FR10), each carrying only its
  own fields.

No rules language, and no stringly-typed dispatch: where the engine
switches on one of these types, the switch is exhaustive and the compiler
says so when a case is added.

## Functional requirements

Rule citations are to the printed page numbers of the three books.

**FR1 — Characteristics.** Roll 2D each for Strength, Dexterity, Endurance,
Intelligence, Education, Social Standing, in that order (p. 4). Store as
numeric values with the UPP hex string derived (base 16, digits A–F for
10–15; p. 8). Initial range 2–12; through play of the procedure values stay
in 1–15 and never exceed 15 (p. 4). All characters begin at age 18 (p. 4).
Alterations go through the `Profile`, which clamps; aging is the one path
with a floor of 0 (p. 4).

**FR2 — Enlistment and the draft.** One enlistment attempt into one of the
six services against the Prior Service Table's enlistment throw with its
cumulative characteristic DMs (pp. 5, 10). On rejection the character _may_
submit to the draft: one die, entering the service with that draft number —
possibly the very service that just rejected him (p. 5). P. 5 prints both
readings of the draft: the Enlistment paragraph's "may submit to the draft"
and the Draft paragraph's opening "must submit to the draft." The **may**
reading governs here — declining ends generation with an 18-year-old
civilian, a valid if bare record — and the choice of clause goes in
ERRATA.md as an interpretation. Draftees are not eligible for commission
during their first term (p. 5). The step returns an `Enlistment`.

**FR3 — Terms of service.** Each term is 4 years (p. 5). Per term, in Book
1's order: survival throw — failure is death and the record says so (p. 5);
commission attempt (once per term until achieved; not draftees' first term;
not Scouts or Other, p. 6, p. 10); promotion attempt (one per term, only if
commissioned; p. 6); skills (FR4); reenlistment throw, made every term
whether or not the character wants to stay — failure forces him out, a 12
exactly forces him to stay, even past term 7 — the page grants "an
additional term" in the singular and is silent on a second 12, so the
recurrence is a reading and goes in ERRATA.md (pp. 6–7). Ranks per the
Table of Ranks (p. 10). Voluntary service caps at 7 terms; retirement is
available after the 5th term (p. 7). The per-term order is an
interpretation the book forces: the exposition presents survival,
commission, promotion, skills, then reenlistment (pp. 5–6), but the Jamison
example rolls reenlistment before its term's skills (p. 24). The
exposition's order governs; the reading goes in ERRATA.md.

**FR4 — Skills and training.** Eligibility per the Basic Skill Eligibility
box (p. 6): 2 for the initial term, 1 per subsequent term, 1 on commission,
1 on promotion. Each eligibility is one die on one of the four Acquired
Skills tables for the character's own service — the fourth gated on
Education 8+ — with the table declared before the die is rolled (p. 11).
Three result kinds (p. 12), which is what makes `TableResult` a sum:
characteristic alterations, applied immediately; weapon expertise, where
Blade Combat and Gun Combat require the specific weapon to be chosen
immediately (pp. 11–13, weapons lists pp. 12–13); and basic skills,
accumulating as Skill-1, Skill-2, … with no cap (p. 13). Rank and service
skills accrue automatically, outside eligibility, per the Rank and Service
Skills box (p. 23): Marine Cutlass-1, Army Rifle-1, Scout Pilot-1, the
officers' weapons, the Navy flag ranks' +1 Social, Merchant 1st Officer
Pilot-1. The box's own timing rule — "as soon as he becomes eligible" —
leaves _when_ underspecified for the service-wide entries (on enlistment is
the natural reading); the reading goes in ERRATA.md.

**FR5 — Aging.** From age 34 (end of term 4) and each 4 years after, apply
the Aging Table's saving throws and reductions per term as the ages are
crossed (pp. 7, 9) — per term, not batched at muster out; the Jamison
example batches "for simplicity" and says so (p. 25). A characteristic
reduced to zero is a medical crisis: saving throw 8+, survival means
immediate recovery to 1 plus 1D months of age (pp. 7–8). Two gaps need
recorded readings in ERRATA.md: the page states what happens on survival
and on absent medical care but never states the failed throw's outcome, and
the printed DM for attending medical expertise has no referent during solo
generation (no DM applies).

**FR6 — Mustering out.** On any `Departure` except death (p. 7): rolls
equal to terms served, plus 1 for rank 1–2 or 2 for rank 3+; at most 3
rolls on Table 2 (cash), the rest on Table 1 (material benefits); table
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
sale, no mortgage, duplicates lost (p. 23). Retirement pay, for a
5th-term-or-later departure from Navy, Marines, Army, or Merchants only:
CR 4,000 per year at 5 terms, per the table, +CR 2,000 per term past 8
(pp. 7, 21).

**FR7 — Titles.** Social Standing 11+ may assume the hereditary title
(p. 5); the full range is Book 3's Nobility table, knight/dame at 11
through duke/duchess at 15 (Book 3 p. 22). Assuming it is a choice point;
the record stores the eligibility and the choice.

**FR8 — Character record.** Track name, age, terms served, service, rank,
characteristics and UPP, full skill list with weapon specifics, benefits
(cash, passages, TAS, weapons, ship with its age and remaining payments),
retirement pay, title, and — for the dead — the term and cause. The record
is a projection of the domain types, marshalled through hand-written
codecs where a domain type's JSON shape differs from its in-memory one; the
Markdown sheet is a render of the same values. A deliberate delta from
sibling tools: Books 1–3 have no calendar or birthdate rules, so the record
carries age and terms, not a date. That is the books, not an omission.

**FR9 — Choice points.** Every point at which the procedure asks goes
through a `Decider`, which interactive mode, auto mode, and the tests all
implement. The interface is **one method per choice point**, each typed to
its own alphabet — that is what makes the set of choice points closed:

```go
type Decider interface {
    Service(from []ServiceName) (ServiceName, error)
    SubmitToDraft() (bool, error)
    AttemptCommission() (bool, error)
    AttemptPromotion() (bool, error)
    SkillTable(from []SkillTable) (SkillTable, error)
    Weapon(cat WeaponCategory, from []WeaponName) (WeaponName, error)
    // … reenlist intent, muster table, the two optional DMs,
    //   the muster weapon-or-expertise option, title assumption
}
```

Adding a choice point breaks every implementation at compile time, which is
the whole point of spelling it this way rather than as one method over a
label. Strings appear only where the engine writes the choice into the
event log, where they are prose for a reader.

**FR10 — Dice engine.** 2D throws with DMs against a `Target` per the die
roll conventions (pp. 2–3), plus the single-die rolls the procedure uses
(draft, skills tables, mustering out, recovery months). All rolls drawn
from a seeded stream and logged.

**FR11 — Generation record.** An ordered event log of the entire
generation: each procedure step entered, each throw (dice, target, DMs,
result), each choice (who decided — player or policy — and what the
alternatives were), each consequence (skill gained, rank gained,
characteristic change, years elapsed, death). Stored as an `events` array
in the character JSON with monotonic sequence numbers; consequence events
reference the causing throw or choice. It serves audit — verify any
character against Book 1 by walking the log — and narrative: a readable
service-record transcript.

## Determinism

Not a verification contract; a working guarantee, stated plainly so nobody
mistakes it for more.

- The dice come from Go `math/rand/v2` PCG seeded by `--seed`, defaulted
  from the system when not given and recorded either way.
- **Within one build**, the same seed, inputs, and decisions produce the
  same character, byte for byte. That is what makes golden fixtures work
  and what makes a bug report reproducible.
- **Across builds**, nothing is promised. Changing the procedure, the
  dice-stream consumption order, or an event's wording changes the output
  for a given seed, and that is an ordinary change, not a breaking one.
  Regenerate the goldens and read the diff.
- The record carries its seed, its inputs, and the ruleset it was generated
  under, so a character can be regenerated and so a reader knows what he is
  looking at. These are provenance for a human, not stamps a tool verifies.

## JSON conventions

Characteristics stored numeric with the UPP hex string derived and stored
alongside; money as integer credits; age in whole years with terms served.
Skills are name plus level, with the specific weapon where the rules demand
one. `character.schema.json` (draft 2020-12) describes what the engine
writes, with a minimal and a complete example beside it, and is validated
against in tests — a description of the output, kept honest by CI, not a
promise to records already written.

## CLI sketch

```
ctchargen new [--seed N] [--auto] [--service navy] [--name X] [-o file]
ctchargen batch --count 20 --auto [--service ...] [-o dir|file.jsonl]
ctchargen render [--history] character.json
ctchargen version
```

`--service` forces the _enlistment attempt_ only: the throw is still made,
and a failed throw still goes to the draft, which can land anywhere — that
is what one attempt plus a one-die draft means (p. 5). Interactive mode
walks the procedure step by step; auto mode applies the policy of
POLICY.md, whose selectable rows the `--skills`, `--muster`, and `--career`
flags choose between. `new` writes JSON to stdout unless `-o`; `batch`
emits JSONL (or one file per character with `-o dir`), requires `--auto`,
and derives each member's seed from the base seed plus index, recorded in
each record. Existing files are never overwritten without `--force`. Flags
precede the filename. `version` reports the build, read from the
toolchain's embedded build info.

The auto policy is **total** — it can answer every method of the `Decider`
— and deterministic, tie-breaking by first-listed order in Book 1. Its
selectable rows are named strategies; every strategy is a pure function of
the choice it is handed. The decision table lives in POLICY.md, and a
record generated under a non-default strategy names it in the record's
inputs.

## Decisions

- **Death is an outcome, not an error.** The record is the product: a
  character killed by a survival throw gets a complete record and a
  rendered sheet saying when and where. `batch` emits what it rolled;
  filtering the dead is the caller's business, and no flag rerolls them
  silently.
- **The held printing governs.** Where the © 1977 text differs from the
  1981 revision most people remember, implement the page as held; do not
  adjudicate editions from memory. Deviations and interpretations go in
  ERRATA.md with the page cite, never applied silently.
- **Never implement a rule from memory.** Every implemented rule carries a
  printed-page cite; where the text is ambiguous or silent, the chosen
  reading is recorded before it is coded.
- **Types over checks.** Where a rule constrains a value to a printed set,
  that set is a type and the constraint is unrepresentable rather than
  validated. A runtime check for an impossible value is a sign the type is
  wrong, not a sign the check is needed.
- **The domain types are the interface; JSON is a projection.** Where the
  two disagree — a sum that must marshal flat, an array-backed profile that
  must marshal as six named keys — the codec absorbs the difference and the
  domain type keeps its shape.
- **Names**: blank by default, `--name` to supply one. The book's naming
  section is advice, not a table (pp. 4–5); no generator in v1.
- **Clean room.** Sibling repos are not imported from or copied; consult
  them only when explicitly asked. The _contracts_ they proved — the event
  log, the COVERAGE/ERRATA/POLICY documents — are adopted; their code is
  not.

## Architecture notes

- Repo: new and empty; code does not live in the document collection.
- Packages: `dice`; `traveller` (the domain types of the Domain model
  section, and nothing that rolls or decides); `rules` (the embedded
  tables: wire types, their lift into domain values, and the registry of
  the six services); `chargen` (the engine walking pp. 4–25, consuming a
  `Decider`); `render`; `cmd/ctchargen`.
- The dependency arrow points one way: `traveller` imports nothing of the
  others, `rules` and `chargen` import `traveller`, `render` imports the
  record. A domain type that needs to know about dice or JSON is in the
  wrong package.
- Data/logic boundary: tables, thresholds, and labels are embedded data
  files loaded with `go:embed`, lifted into domain types at load with the
  errors that lift produces; orchestration and the procedure's exceptional
  mechanics are typed Go. No rules language.
- Testing: golden character fixtures per service (including a draftee, a
  death, and a civilian who declined the draft), regenerated and never
  hand-edited; schema validation; property tests on the dice engine;
  table-lift tests that assert a malformed data file fails loudly. What the
  compiler proves is not also tested.

## Documents

- `PRD.md` — this contract; milestones live here.
- `COVERAGE.md` — every step and per-service rule of pp. 4–25 mapped to its
  page cite, implementation, and test.
- `ERRATA.md` — the recorded readings, each with its page cite.
- `POLICY.md` — the auto-mode decision table.
- `character.schema.json` plus a minimal and a complete example.

## Milestones

1. `traveller` and `dice`: the domain types, the seeded stream, the
   `Target` throw. Then a walking skeleton through one service (Other — no
   commissions, no ranks, the shortest path through the table) with the
   event log wired in from the start, JSON out, and a rendered sheet.
2. All six services: enlistment, draft, survival, commission, promotion,
   reenlistment, ranks, and the four skills tables with weapon choices and
   rank-and-service skills. Exit criterion: a living COVERAGE.md.
3. Aging, retirement, and mustering out, including the ship benefits' Book
   2 values and the Book 3 titles.
4. Interactive mode polish; batch mode; the Markdown transcript.

## Sources

- `~/Documents/Traveller/Classic/Book 1 Characters and Combat.pdf`
  (authoritative for the whole procedure; chargen pp. 4–25).
- `~/Documents/Traveller/Classic/Book 2 Starships.pdf` (Type S p. 18,
  Type A p. 19 — consulted only because Book 1 p. 22 points there).
- `~/Documents/Traveller/Classic/Book 3 Worlds and Adventures.pdf`
  (Nobility p. 22 — consulted only because Book 1 p. 5 points there).

Page N as printed is PDF page N+6 in Book 1, N+5 in Books 2 and 3.
