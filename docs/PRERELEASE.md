# PRERELEASE: the review before the tag

2026-09-03. Companion to `PRD.md`, and the document that governs whether
`v1.0.0-alpha.3` ships.

Every milestone of `PRD.md` is complete. What the PRD and `CLAUDE.md` both
say remains is a review of the whole tool against the four documents. This
file records what each pass checked and what it found. A finding is written
here **before** it is fixed: a finding fixed in passing is a finding nobody
can audit.

Findings are numbered by pass. **Status** is `open`, `fixed` (with the PR),
or `won't fix` (with the reason).

## The shipping bar

No finding of severity **high** may be open when the tag is cut. Medium and
low findings may ship open if this file says why.

---

## Pass 1 — the page against the code

The only pass that can find a rule that was never implemented: `docsgate`
checks that every test `COVERAGE.md` cites exists, not that the map is
complete, so a rule with no row is invisible to every automated check.

### What was read

Book 1 pp. 4–25 in full (PDF pp. 10–31), visually, per the Authority
section. Book 2 pp. 18–19 and Book 3 p. 22, the only pages Book 1 points at.

### What was checked and found correct

Every table was compared cell by cell against the embedded data:

| Table                                     | Page    | Result                                                          |
| ----------------------------------------- | ------- | --------------------------------------------------------------- |
| Prior Service Table (6 rows × 6 services) | 1:10    | exact, DMs included                                             |
| Table of Ranks                            | 1:10    | exact; Merchants stop at rank 5, rank 6 printed `—` (E013)      |
| Acquired Skills Table (4 tables × 6 × 6)  | 1:11    | exact, including the printed duplicates and Other's `−1 Social` |
| Mustering Out Table 1                     | 1:9     | exact, including all three `—` cells                            |
| Mustering Out Table 2                     | 1:9     | exact                                                           |
| Aging Table                               | 1:9     | exact, including the three-band reading                         |
| Rank and Service Skills                   | 1:23    | exact; `Rifl3-1` correctly read as Rifle-1 (E012)               |
| Annual Retirement Pay                     | 1:21    | exact, Scouts and Other excluded                                |
| Blades and Polearms; Guns                 | 1:12–13 | exact, column-major per the recorded reading                    |
| Nobility                                  | 3:22    | exact, five ranks 11–15                                         |
| Scout/Courier Type S; Free Trader Type A  | 2:18–19 | exact, 100 and 200 tons                                         |

E012's normalization map was checked against the skill-description headings
of pp. 12–20 and covers every abbreviation the tables print.

Two candidate findings dissolved on checking, and are recorded so they are
not raised again:

- **Whether a dead character musters out.** P. 7 makes benefits follow
  leaving the service "for any reason", and the engine skips the dead. Not a
  silent reading: `ERRATA.md`'s _Checked and found determinate_ section
  already settles it from p. 5's "a new character must be generated."
- **Book 1 p. 5's Sir/Baron against Book 3's five ranks.** P. 5 gives Sir at
  Social Standing 11 and Baron at 12; Book 3 p. 22 gives knight/dame at 11
  and baron/baroness at 12. They agree, and p. 5 defers to Book 3 for the
  full range.

### Findings

#### P1-1 — The Aging Table's two exemptions have no `COVERAGE.md` row

**Severity.** Low. **Cite.** Book 1 p. 9. **Status.** fixed, this pass.

P. 9 prints two rules the map does not carry: _Education_ and _Social
Standing_ are `unaffected by aging` across every column, and _Intelligence_
has `no effect before age 66`.

Both are implemented — the aging bands carry no Education or Social Standing
effect at any term, and Intelligence appears only in the third band — and
both are tested: `rules.TestAgingTable` asserts the Education and Social
Standing exemption at every term from 4 to 40, and its exact-list comparison
holds Intelligence to the third band alone.

So this is a gap in the map, not in the code. It matters because
`COVERAGE.md`'s own preamble makes an unmapped rule indistinguishable from an
unimplemented one: "A rule with no row here is a rule that is not
implemented."

#### P1-2 — `COVERAGE.md`'s header is stale

**Severity.** Low. **Cite.** `docs/COVERAGE.md` line 3. **Status.** fixed, this pass.

It reads `2026-09-03. Milestone 2.` Every milestone is complete; the date is
current but the milestone is four PRs behind.

#### P1-3 — A batched PDF read misreads table cells

**Severity.** Medium (method, not code). **Cite.** `CLAUDE.md`, Authority.
**Status.** fixed, this pass.

Reading six PDF pages in one `Read` call rendered the Acquired Skills Table
at a fidelity where Merchants row 3 of the Service Skills Table read as
`Electronic`. The page says `Blade Cbt`, which is what the data has always
had. Re-reading the same page alone gave the correct value immediately, as
did every other single-page read in this pass.

The whole authority model rests on visual reads being accurate.
`CLAUDE.md` sanctions the visual read but says nothing about how many pages
may be read at once, so the trap it exists to avoid is reachable through the
tool it recommends.

#### P1-4 — The font trap reaches the rendered image, not only the extracted text

**Severity.** Medium (method, not code). **Cite.** Book 1 p. 23;
`CLAUDE.md`, Authority. **Status.** fixed, this pass.

`CLAUDE.md` and the PRD both frame the substituted glyphs as an artifact of
`pdftotext`, and prescribe the visual read as the remedy. But the Rank and
Service Skills box on p. 23 renders as `Rifl3-1` **in the page image as
well**. A visual read alone does not defeat the trap here.

What actually caught it was a semantic cross-check: no skill or weapon named
`Rifl3` appears anywhere in the three books, and Rifle is on the p. 13 gun
list. E012 records exactly this reasoning, so the tool is correct — but the
guidance describes a defence weaker than the one that worked.

---

## Pass 2 — the documents against the code

The class both of #22's shipped bugs came from: the document being right and
the code not having re-read it. Each sentence checked against what the code
does, not against memory of it.

### What was checked and found correct

- **All 15 `ERRATA.md` stamping conditions**, re-checked against the errata
  every golden actually carries. They agree, including the two that turn on
  a distinction rather than a count: E006/E007 are absent from `other-death`,
  which served four terms but died at the fourth survival throw and never
  reached the round; and E004 is absent from `other-crisis-died`, which died
  of a crisis rather than a survival throw.
- **`Age` normalization.** Months carry into years at twelve, and
  `traveller.TestAgeReads` holds the 0–11 invariant across addition,
  subtraction and multiple crises, so a record cannot violate the schema's
  `months` bound.
- **The CLI sketch**, command by command: `--name`, `-o`, `--force`,
  `--history`, the batch filenames (`00000000000000000145.json`), members
  numbering from zero, `--service` forcing only the attempt, and the build
  stamp read from `debug.ReadBuildInfo`.
- **The README's gate list** against `Taskfile.yml`'s `default` — tidy, vet,
  lint, nilaway, test, ratchet, in that order.
- **FR8's record fields**, against goldens that carry each: rank and
  `rankTitle`, `annualRetirementPay` (CR 8,000 at seven terms, absent for
  the Scouts), the ship with its age and remaining payments, and the dead
  character's term and cause.

### Findings

#### P2-1 — The passage prices are lifted, tested, and applied to nothing

**Severity.** Low. **Cite.** `PRD.md` FR6; Book 1 pp. 21–22. **Status.**
fixed, this pass.

FR6 named the passage prices and the 90% resale rate as part of the
requirement. They are lifted by `rules.liftPassages`, reachable through
`Rules.Passage()` and `Muster.ResalePercent`, and asserted in
`rules.TestTheMusterNotes` — and no engine path, render or record field
reads either. The record reports which passages a character holds and never
what one is worth.

FR6 now says so: the prices are reference data v1 applies to nothing.
Pricing a passage on the sheet, as the book's own summary does for Jamison
(p. 25), is a candidate for after the tag rather than a v1 promise.

#### P2-2 — The PRD said age is whole years

**Severity.** Low. **Cite.** `PRD.md`, JSON conventions. **Status.** fixed,
this pass.

"age in whole years with terms served" — but a medical-crisis recovery adds
1D months (FR5, pp. 7–8), the schema carries a `months` property bounded 1
to 11, and `other-crisis-survived` records `{"years": 46, "months": 2}`. The
schema was precise where the PRD was not.

#### P2-3 — `COVERAGE.md` cited a golden that cannot show the rule

**Severity.** Medium. **Cite.** `COVERAGE.md`, Titles; E011. **Status.**
fixed, this pass.

The row "The dead are assessed but not asked" cited golden `other-death`.
That character's final Social Standing is 5: he is not eligible, carries no
`title` block and no E011 stamp, so the golden cannot demonstrate the rule
it was cited for. Every other death in the roster also ended below 11, so
**nothing exercised the branch** — `assessTitle`'s `if r.dead` arm, which
sets eligibility and declines to ask.

This is the third time a `COVERAGE.md` citation has named a fixture that
does not carry its rule, and the `docsgate` gate cannot catch it: it checks
that a cited golden exists, not that it exercises anything.

Fixed by adding golden `died-a-noble` — seed 39 through the Navy, killed by
a survival throw holding Social Standing 12, recorded `{"eligible": true,
"rank": "baron/baroness", "assumed": false}` and stamped E011 — and citing
it instead.

#### P2-4 — `POLICY.md` claimed a branch was golden-reachable that no strategy can reach

**Severity.** Medium. **Cite.** `POLICY.md`, the muster DMs; Book 1 p. 9.
**Status.** fixed, this pass.

`POLICY.md` said "`spartan` exists so that the declined branch of both DMs
is reachable by a generated golden," and named the standard it was meeting:
"a branch no fixture exercises is a branch nothing tests." Across all
eighteen goldens both DMs were answered **yes** every time.

Worse for Table 2, the claim cannot be made true by any fixture: `spartan`
is the only strategy that declines a DM and also the only one that never
rolls on Table 2, because it prefers Table 1 and Table 1 is always offered.
The declined branch is unreachable from the auto policy entirely.

Fixed in two parts. Table 1's declined branch is now carried by a generated
golden, `navy-spartan-declines` — seed 4 through the Navy under `spartan`,
which reaches rank 5 and refuses the +1 nine times. Table 2's is held by
`ctchargen.TestInteractiveOffersTheGamblingModifier`, where a person is
asked and answers no; `POLICY.md` now says which branch is reached how, and
that one of them is reachable only from outside the policy.

#### P2-5 — The README's document table omitted `PRERELEASE.md`

**Severity.** Low. **Cite.** `README.md`. **Status.** fixed, this pass.

Pass 1 added the row to `CLAUDE.md`'s table and not to the README's.

### Decided, not a finding

- **The interactive menu's third wording of p. 22's alternative.** The menu
  offers `+1 expertise in the Dagger` where the record says `expertise in
  Dagger`. They are kept apart deliberately — one addresses a person reading
  a numbered list, the other a record — and nothing compares them, because
  the engine's gate folds the benefit rather than matching the string. Now
  said so in `interactive.go`, so a later pass does not unify them by
  mistake.

## Pass 3 — the invariants against themselves

Not yet run. Its checklist: break every gate deliberately and record the
message it produced — the six `internal/docsgate` tests, the ratchet in both
directions, the golden comparison, the schema validation, the Jamison
replay; sweep for values validated in one place and trusted in another; and
extend schema validation past the goldens to records the CLI actually
writes.
