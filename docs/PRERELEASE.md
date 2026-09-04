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

Not yet run. Its checklist: `PRD.md` sentence by sentence (FR1–FR11, Goals,
Non-goals, the CLI sketch, Determinism, JSON conventions, Architecture
notes, Testing); all 15 `ERRATA.md` readings, each **Stamped on.** condition
re-checked by arithmetic against the code that stamps it; all 12 `POLICY.md`
rows against the `Decider` method's behaviour; `README.md` and `CLAUDE.md`
against the tool as built; and the interactive menu's third wording of the
p. 22 alternative (`+1 expertise in the Dagger` against the record's
`expertise in Dagger`).

## Pass 3 — the invariants against themselves

Not yet run. Its checklist: break every gate deliberately and record the
message it produced — the six `internal/docsgate` tests, the ratchet in both
directions, the golden comparison, the schema validation, the Jamison
replay; sweep for values validated in one place and trusted in another; and
extend schema validation past the goldens to records the CLI actually
writes.
