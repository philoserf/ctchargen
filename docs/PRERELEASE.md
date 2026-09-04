# PRERELEASE: what each tag ships with open

Companion to `PRD.md`. One section per tag, newest last: the review that
preceded it where there was one, and what was still open when it was cut. A
finding is written here **before** it is fixed: a finding fixed in passing is
a finding nobody can audit.

Findings are numbered by pass. **Status** is `open`, `fixed` (with the PR),
or `won't fix` (with the reason).

---

# v1.0.0-alpha.3 — 2026-09-03

The tag the `PRD.md` milestones were finished for. Every milestone is
complete; what the PRD and `CLAUDE.md` both say remains is a review of the
whole tool against the four documents, and this section records what each
pass checked and what it found.

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

### Every gate, broken on purpose

`CLAUDE.md`: "a gate never seen to fail has not been shown to hold." Each was
broken deliberately and the message it produced recorded. All thirteen fail,
and each names what was broken.

| Gate                                      | Break                                      | What it said                                                                           |
| ----------------------------------------- | ------------------------------------------ | -------------------------------------------------------------------------------------- |
| `TestErrataMatchTheDocument`              | E015's heading renamed to E016             | reported **both** directions: E015 missing, E016 unknown                               |
| `TestPolicyRowsMatchTheDecider`           | the `AssumeTitle` row's heading broken     | "AssumeTitle is in the Decider interface but has no entry"                             |
| `TestChoicePointsMatchTheDecider`         | `ChoiceMusterWeapon` dropped from the list | "MusterWeapon … does not exist in the ChoicePoint enum"                                |
| `TestCoverageCitesTestsThatExist`         | a cited test misspelled                    | named the citation and that no test file declares it                                   |
| `TestCoverageCitesGoldensThatExist`       | a cited golden misspelled                  | named the citation and that it is not in `chargen/testdata`                            |
| `TestGoldens`                             | a golden record hand-edited                | named the file and said to regenerate and read the diff                                |
| `TestGoldensRegenerate`                   | a stray die drawn on every second run      | named the fixture that did not reproduce from its own seed                             |
| `TestEveryGoldenMatchesTheSchema`         | `terms` capped at 1                        | named each golden that no longer matches                                               |
| `TestTheDocumentedExamplesMatchTheSchema` | `character.complete.json`'s UPP broken     | named the example                                                                      |
| `TestTheWorkedExampleReproduces`          | the Merchants' survival target 5+ → 9+     | "the example narrates 17 throws and 14 dice the engine never asked for"                |
| `rules.TestRankAndServiceSkills`          | the Scouts' service-wide grant removed     | caught by the second transcription, and by six goldens besides                         |
| the ratchet, rising                       | an uncovered function added                | "coverage fell — these packages gained uncovered statements"                           |
| the ratchet, falling                      | any pass that covers more                  | "coverage improved — lock it in, a stale number is a ratchet that has stopped holding" |

Two of the breaks were **invalid on the first attempt**, and both are worth
recording because they look exactly like a passing gate:

- Altering the **Navy's** survival target left the worked example untouched,
  because Jamison is a Merchant. Aimed at the right column, it failed at once.
- Making generation nondeterministic with `time.Now().UnixNano() % 2` never
  varied: macOS reports microsecond granularity, so the nanosecond parity is
  always even and the mutation was really "always draw", which
  `TestGoldensRegenerate` correctly does not catch because both runs shift
  alike. A counter that alternates between calls killed it immediately.

A mutation that does not apply is indistinguishable from an invariant that
does not hold. Every surviving mutant in this pass was the mutation's fault,
and each was re-aimed until it either killed or was understood.

### Findings

#### P3-1 — The command validated the strategies and the engine trusted them

**Severity.** Medium. **Cite.** `PRD.md` FR9; `POLICY.md`. **Status.** fixed,
this pass.

`Policy.Validate()` was called from `cmd/ctchargen` and from nowhere else.
`chargen.Generate` never checked the decider it was handed, so

```go
chargen.Generate(inputs, chargen.Policy{Career: "dawdle", Skills: "osmosis", Muster: "gold"})
```

generated a complete seven-term character and wrote those three words into
the record's `inputs`, where no row of `POLICY.md` answers to any of them.
Nothing failed, because an unrecognized strategy matches no branch and
silently behaves as whichever one the conditionals fall through to.

This is the same shape as the gap #22 found — the engine applying a
caller-supplied value that no page allows — and it is now refused in the same
spirit: `chargen.validate` asks any decider that can be misconfigured whether
it was built out of things that exist, before generation starts. A decider
with no `Validate` is not asked, so the scripted test deciders are unaffected.

#### P3-2 — Nothing validated what the command writes

**Severity.** Medium. **Cite.** `PRD.md`, JSON conventions. **Status.** fixed,
this pass.

`render`'s schema test validated the goldens and the two documented examples
— what the _engine_ produces. What the _command_ writes was validated by
nothing, and the two are not the same document: the command stamps `build`
and fills `name`, neither of which any golden carries, and a batch member and
a guided run are further paths again.

This is why #22's `--career dawdle` record could be schema-invalid with CI
green. `cmd/ctchargen.TestWhatTheCommandWritesMatchesTheSchema` now validates
an automatic run, a named character, a death, and a guided run;
`TestEveryBatchMemberMatchesTheSchema` validates every line of a batch.

Shown to hold by deleting `build` from the schema: the golden test stays
green — no golden carries the field — and the new test fails. That is
precisely the gap it was written for.

### Decided, not a finding

- **`render` trusts the record it reads.** It accepts `terms: -5`, a UPP of
  `"zzz"`, a Strength of 99 and a missing service, and renders a sheet for
  each. This is deliberate and `decode` says so: nothing is rebuilt into a
  domain type, because the record is a projection of the domain values and a
  sheet is a projection of the record. The behaviour was undocumented outside
  that comment, so the CLI sketch now states it — a record is something this
  tool wrote, and a sheet of a hand-edited one is the answer the operator
  asked for.
- **The `Decider` methods' error propagations remain uncovered.** Reaching
  each takes an input that ends at that exact question. They are one-line
  returns, and the ratchet holds them at a count rather than pretending
  otherwise.

---

# v1.0.0-alpha.4 — 2026-09-04

The first tag cut for a referee's finding rather than for a milestone.

## What changed since alpha.3

- **#69, closing #29** — every command lists its own flags. `new --help` and
  `batch -h` printed `usage: flag: help requested` and exited 1; `--help` at
  top level read as a mistyped subcommand; `new extra-arg` ignored the word
  and started an interactive generation on a drawn seed. Reviewed at
  `/code-review high`; the findings, one of which was a test that did not test
  what it claimed, were applied in `860dd9e`.
- **This PR, closing #30 and #31** — the README gained an install section and
  the flags it never documented, and the rejection for an unknown strategy
  stopped naming a repository file at a reader who has only a binary.

No new pass was run. The three passes above reviewed the engine and the
documents against the page; nothing since has touched either. What changed is
the command surface and the prose, both of which the gate covers.

## What ships open, and why

**This tag departs from the shipping bar above.** That bar — no finding of
severity high open when the tag is cut — was written for the review of the v1
contract, and it held for alpha.3. Three high findings from the code audit
(#27) are open here:

|     |                                                     | Why it ships open                                                                                                         |
| --- | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| #41 | `Policy` strategy names are strings                 | Structural. The runtime validation that compensates is tested and holds.                                                  |
| #42 | The record carries a sum _and_ the flag it replaced | Structural. Nothing today writes them into disagreement.                                                                  |
| #43 | Where a printed number lives                        | Structural, and the one number with teeth — `lastPrintedTerm` — is E014's stamping condition, which is stamped correctly. |

None of the three changes a character the tool generates. They are findings
about the shape of the code, raised by a review that came after alpha.3 was
tagged, and an alpha is where a structural finding ships open with a reason
rather than blocking a fix a referee asked for.

Also open, and named so the release notes do not have to restate them:

- **The rest of the referee's report** (#26) — the seed absent from the sheet
  (#32), `batch` silent about what it generated (#33), and #34–#40, #67, #68.
- **#70 — `ctchargen version junk` is now an error.** Shipping this tag with
  the refusal in it is **not** a decision on that issue, which was opened to
  settle it separately.
- Medium and low audit findings #44–#61, and the five open questions #62–#66.

---

# v1.0.0-alpha.5 — 2026-09-04

The tag that closes the referee's report. Every finding of #26 is answered and
the `2 · The referee's table` milestone is empty.

## What changed since alpha.4

- **#72, closing #32** — the sheet prints the seed, and the footer stops
  offering to regenerate a character the seed cannot bring back.
- **#78, answering #62–#66, #70, #73 and #74** — the eight open questions. Four
  answers became standing rules in `CLAUDE.md`: what `v1.0.0` means, that the
  record freezes at that tag, where the data/constant line falls, and that a
  stamping condition must be machine-checked rather than merely reachable. Two
  became issues instead of answers (#76, #77).
- **#79, closing #35, #36 and #37** — the headline says whose service it is,
  and names the service that refused him where the draft placed him elsewhere;
  errata come off the sheet and stay in the record and the transcript; a
  one-term character no longer prints "1 terms".
- **#80, closing #38, #39 and #67** — interactive event numbers no longer skip,
  the two Advanced Education menu entries are told apart, and a non-numeric
  answer is complained about by name instead of silently re-printing the menu.
- **#85, closing #81** — the regenerate line quotes what it pastes. A record is
  a file people share, and its fields were reaching the reader's shell
  unquoted; the markdown fence now widens to hold a value carrying backticks of
  its own.
- **#86, closing #40 and #82** — a session that stops before the end prints the
  seed and the answers given, so `--answers` walks back to the question it
  stopped on; and a scanner failure is no longer reported as "the input ended".
- **#87, closing #68** — a release carries binaries, and the workflow that
  attaches them asserts that what it built knows which release it is.

## The review that preceded it

**No whole-tool pass.** alpha.3's three passes read the engine and the
documents against the page, and nothing since has touched either: every change
above is to the command surface, the rendering, or the release plumbing, all of
which the gate covers.

What ran instead was a review per PR, and that is worth recording because of
what it caught. Three consecutive PRs contained a regression introduced while
fixing something else — a headline that ignored the choice event it was
reading, a quoting fix that dropped control-character escaping, and four
`--answers` interaction bugs. Each was found before merge and fixed. Four test
assertions were also found to be passing for the wrong reason, one of which had
never passed in the sense it claimed.

That is the pattern this tag ships knowing about. It is the argument for the
per-PR review continuing rather than being folded into a pass at the next
milestone: each of those regressions was in new surface written minutes
earlier, which a milestone pass reads last and least.

## What ships open, and why

**This tag departs from the shipping bar**, as alpha.4 did and for the same
reason: #41, #42 and #43 are still open. They are structural findings from the
prerelease code audit (#27), none of them changes a character the tool
generates, and answering them is milestone 3's first work.

Also open, by milestone:

- **3 · Structure the engine** — #41–#43, #45, #47–#49, #61, and #77, the gate
  that would hold an erratum to the records it is supposed to govern.
- **4 · Batch and the record's shape** — #33, #34, #44, #46, #50, and #76. That
  last one matters more than its number suggests: the record does not name its
  own shape, so the freeze promised at `v1.0.0` has nothing to attach to.
- **5 · Housekeeping** — #51–#60, #83 and #84.

## One thing a reader of these notes should know

**This is the first tag whose binaries were attached by CI.** alpha.4's six
were built on a laptop and uploaded by hand, which its notes say; these come
from `release.yml`, checking out the tag itself. The workflow was rehearsed
locally against the runner's own checkout shape and reproduced those six byte
for byte — but a rehearsal is not a run, and this tag is its first.
