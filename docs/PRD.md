# PRD: Classic Traveller Character Generator, domain-typed (Go CLI)

2026-08-30. Status: draft — the v1 contract, before any code.

## The second attempt

A first implementation of this tool shipped as `v1.0.0-alpha.2` and was
removed at `41a213a`. It worked. It was not built out of the domain, and
it carried a replay-and-provenance contract it did not need. This is the
rebuild, and what it takes from the first attempt is deliberately small:
**the questions, not the answers.**

Where the first pass found the held text silent or ambiguous, the gap is
real and is named below — but every reading is decided again against the
page at milestone 0, and none is inherited. No code, no test, no table
transcription, and no policy table comes forward. A reading carried over
unread is the past contaminating the future with an authority it never
earned.

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

## Authority

Rules come only from the three held PDFs, and the page governs over memory
and over any tool that reads the page for you.

- **Never implement a rule from memory.** Training-data Traveller is
  mostly the 1981 revision and later editions; the held © 1977 text
  governs even where it differs.
- **Never read a table out of `pdftotext`.** The held reprints' embedded
  font substitutes glyphs, and the substitutions look like data. Verified
  on Book 1 p. 9 (2026-09-02): Mustering Out Table 1's "—" cells — the
  Scout column's row 7 and the Other column's rows 6 and 7 — extract as
  the bare digit **4**, and "Travellers' Aid" extracts as
  **`Travellers9`**, the apostrophe becoming a 9. The minus sign goes the
  same way, so an `N−` target reads as `N3`. A run that trusts the
  extraction gives the Scout a seventh benefit he does not have.

  So every table is transcribed from a **visual** read of the page (`Read`
  the PDF with a `pages` range) and then transcribed a second time inside
  the `rules` tests, so the two must agree. That second transcription is
  the check the font trap needs, and it is where a new table belongs.

- Every implemented rule carries a printed-page cite, in the code and in
  COVERAGE.md.
- Where the text is ambiguous or silent, the chosen reading goes in
  ERRATA.md with its page cite and its stamping condition, and is named in
  every record it governed — never applied silently.
- Printed page N is PDF page N+6 in Book 1, N+5 in Books 2 and 3
  (Sources).

## Goals

1. Generate a complete character per Book 1's character generation chapter
   (pp. 4–25): characteristics, service, terms, skills, aging, mustering
   out — or death, which is a completed generation too.
2. Two modes: **interactive** (player makes each choice) and **auto** (tool
   decides; supports batch NPC generation).
3. Model the procedure in **domain types**, not strings: the six
   characteristics, the four skills tables, the choice points, the throw
   targets, and the table results are closed types the compiler checks.
   What that buys is exact: no lookup that can miss and no "unknown value"
   error to return, a switch the compiler re-checks when a case is added,
   and a `Decider` that cannot grow a choice point silently. It does not
   buy a procedure whose every illegal state fails to compile — see Domain
   model for where the line falls.
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
  only by regenerable golden fixtures. The reason this is affordable here:
  a record is regenerable from its seed and its inputs, and nothing in it
  is hand-written. A tool whose records a referee annotates could not say
  that, and would owe a render-forward promise instead.
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

The rule dividing the types from the data: **types carry identity, never
rule invariants.** A name, a notation, or an identifier is a type — a
service that is not one of the six is not a service, and a characteristic
outside the six does not exist. A _range a page prints_ is not. Putting
one in a struct definition puts a rules claim where no reader will look
for it, beside the data file and the test that transcribe the same
numbers, which is the drift this design exists to prevent — and a range
guessed rather than read is worse than an absent one, because it rejects a
character the book permits. The procedure's own constraints — a draftee's
first term barring commission, the three-roll cash cap, the Education 8+
gate — are steps, checked where they are applied, with their page cite.

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
  list kept parallel to it. The mechanism is a `Decider` wrapper that logs
  and delegates: it must implement every method to compile, so a new
  choice point cannot reach the engine without reaching the log.
- `Erratum`: the recorded readings of ERRATA.md, by id.

**Value types** — meaning, not primitives:

- `Target`: a throw target with its modality (`N+`, `N−`, exact; pp. 2–3),
  parsed once when the data lifts, never re-parsed at throw time.
- `Credits`: money, integer credits, distinct from every other int.
- `Age`: whole years plus the 0–11 months only a medical-crisis recovery
  can accrue (pp. 7–8), carrying into years on its own.
- `Rank`: an index into the service's own Table of Ranks (p. 10), which
  carries the title; rank 0 is "not commissioned", not a missing value.
  The highest rank is the service's — the table does not run to the same
  length in every column, and two services have no ranks at all — so the
  ceiling is data with a cite, not a bound in the type.
- `Term`: the term's ordinal, counting from 1 (p. 5), with **no upper
  bound**. Seven is the cap on _voluntary_ service (p. 7), and a 12 on the
  reenlistment throw puts a character past it (pp. 6–7). The cap is a rule
  applied at a choice point; a type that stopped at 7 would reject a
  character the book permits.
- `Skill`: a name and a level, with the specific weapon where the rules
  demand one ("Dagger-1", p. 25) and the category that demanded the pick.
- `WeaponName`: the one name that does _not_ close at compile time. The
  blade and gun lists are printed data (pp. 12–13), so weapon names lift
  from the embedded lists and are checked against the list for their
  category; the category is the closed type, the name is data.

**Sums** — the places where "exactly one of" is the rule:

- `Enlistment`: enlisted | drafted into a service | declined the draft. The
  civilian is a case of the type, not a nil service with a flag beside it.
- `TableResult`, one row of an Acquired Skills table (pp. 11–12): a
  characteristic alteration | a named skill | a weapon-category choice.
- `BenefitRow`, one row of the two Mustering Out Tables (p. 9): cash (which
  is Table 2's whole content; Table 1 prints no money row) | passage |
  characteristic alteration | weapon category | Travellers' Aid | ship |
  nothing (Table 1's "—" rows).
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
10–15; p. 8). Initial range 2–12. The page states the bounds as a hard
ceiling and a qualified floor: values "may never exceed 15, and do not go
below 1 except for calamitous injury or aging" (p. 4). So 1 is the floor for
every ordinary alteration — including the procedure's one negative
skills-table result, Other's −1 Social (p. 11) — and aging alone may carry a
characteristic below it. All characters begin at age 18 (p. 4). Alterations
go through the `Profile`, which applies those bounds; a `Profile` that
clamped at 1 unconditionally would make FR5's medical crisis unreachable and
with it every path that puts months on an age. How far below 1 aging
reaches — a floor of 0, or the Aging Table's full −2 (p. 9) taken off a 1 —
the page never fixes, and the answer decides whether the crisis fires at
all; that reading goes in ERRATA.md, citing pp. 4, 7–9.

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
recurrence is a reading and goes in ERRATA.md — weighed against p. 21's
retirement pay table, which prints an 8-terms row and a rate for service
beyond it, so the book does contemplate a career past the singular reading's
ceiling (pp. 6–7, 21). Ranks per the Table of Ranks (p. 10). Voluntary
service caps at 7 terms; retirement is available after the 5th term (p. 7).
The per-term order is an interpretation the book forces: the exposition
presents survival, commission, promotion, skills, then reenlistment
(pp. 5–6), but the Jamison example rolls reenlistment before its term's
skills (p. 24). The exposition's order governs; the reading goes in
ERRATA.md. One further silence: survival failure is death, but the page does
not say how old the dead character is, and the record has to state an age
(FR8). That gap needs its own reading.

**FR4 — Skills and training.** Eligibility per the Basic Skill Eligibility
box (p. 6): 2 for the initial term, 1 per subsequent term, 1 on commission,
1 on promotion. Each eligibility is one die on one of the four Acquired
Skills tables for the character's own service — the fourth gated on
Education 8+ — with the table declared before the die is rolled (p. 11).
Three result kinds (p. 12), which is what makes `TableResult` a sum:
characteristic alterations, applied immediately; weapon expertise, where
Blade Combat and Gun Combat require the specific weapon to be chosen
immediately (pp. 11–13, weapons lists pp. 12–13); and basic skills,
accumulating as Skill-1, Skill-2, … with no cap (p. 12). Rank and service
skills accrue automatically, outside eligibility, per the Rank and Service
Skills box (p. 23): Marine Cutlass-1, Army Rifle-1, Scout Pilot-1, the
officers' weapons, the Navy Captain's and Admiral's +1 Social, Merchant
1st Officer Pilot-1. The box's own timing rule — "as soon as he becomes
eligible" — leaves _when_ underspecified for the service-wide entries (on
enlistment is
the natural reading); the reading goes in ERRATA.md.

Skill **names** are normalized to the descriptions' own headings
(pp. 13–20), so one skill accumulates under one name: the p. 11 tables
abbreviate, they do not spell every name the same way twice, and the p. 23
box carries at least one reprint typo. The normalizations are listed in
ERRATA.md and are the one place a printed string is deliberately not
reproduced. They are stamped on no record, because a spelling is a
transcription rather than a reading: nothing about the character changes
with the choice, only what the skill is called.

**FR5 — Aging.** From age 34 (end of term 4) and each 4 years after, apply
the Aging Table's saving throws and reductions per term as the ages are
crossed (pp. 7, 9) — per term, not batched at muster out; the Jamison
example batches "for simplicity" and says so (p. 25). A characteristic
reduced to zero is a medical crisis: saving throw 8+, survival means
immediate recovery to 1 plus 1D months of age (pp. 7–8). Three gaps need
recorded readings in ERRATA.md: the page never fixes where the aging round
sits within the term — before or after the reenlistment throw — nor the
order of the saving throws within a round, and both fix dice-stream
consumption; the page states what happens on survival and on absent
medical care but never states the failed throw's outcome; and the printed
DM for attending medical expertise has no referent during solo generation
(no DM applies). A fourth belongs with them when milestone 0 assembles the
list, and is stated in FR1 because it is a bound on the `Profile`: how far
below 1 aging may carry a characteristic, which decides whether the crisis
this paragraph describes can fire at all.

**FR6 — Mustering out.** On any `Departure` except death (p. 7): rolls equal
to terms served, plus 1 for rank 1–2 or 2 for rank 3+; at most 3 rolls on
Table 2 (cash), the rest on Table 1 (material benefits); table designated
before each roll; +1 DM available on Table 1 at rank 5–6 and on Table 2 with
gambling skill (pp. 7, 9). Benefits per the tables (p. 9) and their
definitions (pp. 21–23): cash; characteristic alterations, +1 or +2 as the
row prints (p. 9), applied immediately; passages (High CR 10,000, Middle CR
8,000, Low CR 1,000, sellable at 90%; pp. 21–22); Travellers' Aid
membership, once per character ever, duplicate rolls wasted (p. 22);
weapons, where a repeat receipt may take the same weapon, a different one,
or +1 expertise in a weapon already received as a benefit (p. 22); and
ships. The Free Trader is a Type A (Book 2 p. 19), received owing ~CR
150,000 monthly for 40 years, each additional receipt paying off 10 years
and aging the ship 10 years — five receipts is a 40-year-old ship owned free
and clear (pp. 22–23). The Scout ship is a Type S (Book 2 p. 18) in
constructive possession: no title, no sale, no mortgage, duplicates lost
(p. 23). Retirement pay, for a 5th-term-or-later departure from Navy,
Marines, Army, or Merchants only: CR 4,000 per year at 5 terms, per the
table, +CR 2,000 per term past 8 (pp. 7, 21).

**FR7 — Titles.** Social Standing 11+ may assume the hereditary title
(p. 5); the full range is Book 3's Nobility table, knight/dame at 11
through duke/duchess at 15 (Book 3 p. 22). Assuming it is a choice point;
the record stores the eligibility and the choice.

_When_ eligibility is assessed needs a recorded reading. The page fixes no
moment, and Social Standing is not fixed at 18: it rises on a personal
development table, on a rank grant from the p. 23 box, and on a Table 1
row (pp. 10–11, 23; p. 9), and it falls on one service's personal
development table. The reading decides two visible things — whether a
mustering-out alteration can confer a title the character never held while
serving, and whether a character killed in service is assessed at all.

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

- The dice come from Go `math/rand/v2` PCG. The one recorded seed fills
  both words of the state — `rand.New(rand.NewPCG(seed, seed))` — so a
  single field reproduces the stream.
- **The die is `IntN(6) + 1`**, and a 2D throw is two of those in
  sequence, first die then second. This is as load-bearing as the
  consumption order it feeds: `IntN(36)`, or a masked `Uint64`, is the
  same PCG under the same seed and an entirely different character.
- **Dice-stream consumption order is load-bearing** — it is what a seed
  means. The per-term order of FR3 and the aging round's position in FR5
  are therefore fixed choices, not incidental ones, and a throw the
  procedure does not make consumes nothing.
- **Within one build**, the same seed, inputs, and decisions produce the
  same character, byte for byte. That is what makes golden fixtures work
  and what makes a bug report reproducible.
- A seed is always recorded. Without `--seed`, one is drawn from the
  system and bounded to 2^53 − 1, above which consecutive integers are no
  longer all exactly representable in an IEEE-754 double: a reader that
  parses JSON numbers as doubles would otherwise round it silently, and a
  rounded seed is a different character. An explicit `--seed` is the
  operator's own number and is not bounded, so `--seed 0` is an explicit
  choice and not a request for a random one. The bound covers only the base
  seed the tool draws: an explicit seed above 2^53, and a derived `base + i`
  that passes it because the base landed within `--count` of the bound, are
  both written to the record as given, not silently re-bounded.
- Batch member _i_ is seeded `base + i` as a wrapping unsigned 64-bit
  addition, and that derived seed — not the base — is what its record
  carries. **Members number from zero**, so `batch --count 1 --seed N`
  produces exactly what `new --seed N` produces, and every seed a referee
  can type is reachable as a member.
- **Across builds**, nothing is promised. Changing the procedure, the
  dice-stream consumption order, or an event's wording changes the output
  for a given seed, and that is an ordinary change, not a breaking one.
  Regenerate the goldens and read the diff.
- The record carries its seed, its inputs, the ruleset it was generated
  under, and the build that wrote it — read from `debug.ReadBuildInfo`,
  the same source `version` reports — so a character can be regenerated
  and so a reader knows what he is looking at. With cross-build
  reproduction unpromised, naming the build is the whole of what makes a
  record reasonable about. These are provenance for a human, not stamps a
  tool verifies.

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
ctchargen new [--seed N] [--auto] [--service navy] [--name X]
              [-o file] [--force]
ctchargen batch --count 20 --auto [--service ...]
              [-o dir|file.jsonl] [--force]
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
- **Never implement a rule from memory, and never out of `pdftotext`.**
  Stated in full under Authority; it is the decision the rules data turns
  on.
- **Types for identity, data for ranges.** Where a rule names a closed
  _set_ — the six services, the four tables, the two weapon categories —
  that set is a type and an unknown value is unrepresentable rather than
  validated; a runtime check for an impossible value is a sign the type is
  wrong. Where a page prints a _range_, the range stays in the data with
  its cite. Stated in full under Domain model, which is where the line
  between the two is drawn.
- **The domain types are the interface; JSON is a projection.** Where the
  two disagree — a sum that must marshal flat, an array-backed profile that
  must marshal as six named keys — the codec absorbs the difference and the
  domain type keeps its shape.
- **Names**: blank by default, `--name` to supply one. The book's naming
  section is advice, not a table (pp. 4–5); no generator in v1.
- **Clean room, this repo's own past included.** Sibling repos are not
  imported from or copied; consult them only when explicitly asked. The
  _contracts_ they proved — the event log, the COVERAGE/ERRATA/POLICY
  documents, the unpinned gate, the second transcription — are adopted;
  their code is not. The same rule governs the first attempt at `41a213a`
  and before: take only what is essential, which is the list of places the
  book is silent, and re-derive everything else from the page.

## Architecture notes

- Repo: emptied at `41a213a` and rebuilt from this document; code does not
  live in the document collection. Only `LICENSE` and this document
  survived, so `go.mod`, `Taskfile.yml`, the CI workflow, `.golangci.yml`,
  `.gitignore`, and a README return with milestone 1's first Go — the gate
  below cannot hold a milestone it does not yet exist for.
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
- Testing and the gate have their own section below.

## Testing and the gate

- **Golden character fixtures**, regenerated and never hand-edited: one
  per service, plus the cases only a particular path produces — a draftee,
  a civilian who declined the draft, a death in service, a medical-crisis
  survivor (the one path that puts months on an age), each kind of ship
  benefit, a character who assumes a title, and one per selectable auto
  strategy. A field whose declared bound no fixture exercises is a bound
  nothing has tested.
- **The worked example reproduces.** Book 1 pp. 23–25 generates a
  character in full, and FR9 makes the instrument free: a scripted
  `Decider` and a scripted die replay the example, and the result is
  compared against the page — re-derived at milestone 0, not copied.
  Where the example knowingly departs from the rules it illustrates, the
  test records the departure with its page cite rather than matching it.
- **The second transcription** of every numeric table, inside the `rules`
  tests. See Authority; it is what the font trap requires.
- **Regeneration round-trip:** each golden reproduced from its own
  recorded seed and inputs. This is what stands in for a `replay`
  subcommand, and it runs on every `task` against records this repository
  controls, rather than shipping a verifier for records it did not write.
- Schema validation of the goldens and of both documented examples; render
  goldens for the sheet and for the transcript; property tests on the dice
  engine; table-lift tests asserting a malformed data file fails loudly.
  What the compiler proves is not also tested.
- **The documents are held to the code, both directions.** Every erratum
  id stamped in code resolves to an ERRATA.md heading and every heading is
  reachable by some path; every POLICY.md row names a `Decider` method and
  every method has a row; every COVERAGE.md row names a test that exists.
  Each gate is verified by breaking it: a gate never seen to fail has not
  been shown to hold.
- **Coverage ratchets on uncovered statements per package, not a
  percentage.** A percentage holds still while a guarded branch adds one
  covered statement and one uncovered; an integer count trips on the
  first.
- **The gate is `task`** — formatting, `go vet`, golangci-lint, NilAway,
  `go test -race` — and CI runs exactly `task`. **The toolchain is
  deliberately unpinned**: the gate is meant to fail when a tool moves
  rather than drift behind it, so a red gate on untouched code is the
  signal working. Answer the finding; do not pin a tool to silence it.
- **A new invariant is not done until a deliberate mutation has been shown
  to kill it.** Invert a target, drop a case from a switch, halve a loop —
  then read the failure. If it does not name what was broken, the check is
  not holding what it appears to. Regenerate the goldens under the
  mutation first, or they fail for the wrong reason and hide it; and check
  that the fixture can express the bug at all, since a mutation with no
  instance in the record reads exactly like a surviving check.
- **What none of this catches** is a reading applied with neither an
  ERRATA entry nor a stamp: absent from the code and the documents both,
  it is invisible to every comparison between them. Only reading the held
  page finds it, which is why milestone 0 comes first.

## Documents

- `PRD.md` — this contract; milestones live here.
- `COVERAGE.md` — every step and per-service rule of pp. 4–25 mapped to its
  page cite, implementation, and test.
- `ERRATA.md` — the recorded readings, each with its page cite.
- `POLICY.md` — the auto-mode decision table.
- `CLAUDE.md` — points here for the authority model and adds only what an
  agent needs and a human reader does not.
- `character.schema.json` plus a minimal and a complete example.

## Milestones

0. **The documents that govern the code**: ERRATA.md, with every reading
   this document names decided against the held page and carrying its cite
   and its stamping condition; POLICY.md; CLAUDE.md. All three precede the
   first line of Go — a guard written after the code it guards is not a
   guard.
1. `traveller` and `dice`: the domain types, the seeded stream, the
   `Target` throw. Then `rules`, and with it **every table of pp. 4–25**,
   transcribed in one visual reading pass with its second transcription in
   the `rules` tests written in the same sitting — the font trap makes a
   second pass over the same pages expensive for nothing, and a second
   copy retyped from the first checks nothing. Then a walking skeleton
   through one service (Other — no commissions, no ranks, the shortest
   path through the table) with the event log wired in from the start,
   JSON out, and a rendered sheet.
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
