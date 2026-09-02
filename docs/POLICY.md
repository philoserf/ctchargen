# POLICY: the auto-mode decision table

2026-09-02. Milestone 0. Companion to `PRD.md`.

`--auto` answers every question the procedure asks. This document is the
whole of what it answers, one row per `Decider` method. The gate holds the
two to each other in both directions: every row here names a method that
exists, and every method has a row (PRD, Testing and the gate).

## The interface

`Decider` has **one method per choice point**, each typed to its own
alphabet. That is what closes the set: adding a choice point breaks every
implementation at compile time. Interactive mode, auto mode, and the tests
each implement all of it.

```go
type Decider interface {
    // Career
    Service(from []EnlistmentOffer) (ServiceName, error)
    SubmitToDraft() (bool, error)
    AttemptCommission() (bool, error)
    AttemptPromotion() (bool, error)
    ReenlistIntent(from []Intent) (Intent, error)
    AssumeTitle(t Title) (bool, error)

    // Skills
    SkillTable(from []SkillTable) (SkillTable, error)
    Weapon(cat WeaponCategory, from []WeaponName) (WeaponName, error)

    // Mustering out
    MusterTable(from []MusterTable) (MusterTable, error)
    MusterTable1DM() (bool, error)
    MusterTable2DM() (bool, error)
    MusterWeapon(cat WeaponCategory, from []WeaponName,
        received []WeaponName) (WeaponBenefit, error)
}
```

Twelve methods, and the PRD's FR9 sketch elides six of them behind its
ellipsis. Two notes on where this differs from what FR9 prints:

- **`Service` is offered `[]EnlistmentOffer`, not `[]ServiceName`.** FR9
  sketches `Service(from []ServiceName)`. The PRD also says the auto policy
  tie-breaks "by first-listed order in Book 1", and a tie-break presupposes a
  ranking; the only ranking a service has is its enlistment throw and the
  cumulative characteristic DMs it earns (p. 10). An `EnlistmentOffer`
  carries exactly that — the `ServiceName`, the service's printed enlistment
  `Target`, and the DM this character earns against it — so the ranking stays
  a pure function of the offered set and the policy never reaches into the
  character. The shape `from []X` survives; only the element type changes.
  Recorded here rather than applied silently.
- **The remaining five signatures are new**, not sketched. They are fixed
  here so milestone 1 writes them once. Four alphabets come with them that
  the PRD's Domain model does not yet list — `Intent`, `MusterTable`,
  `WeaponBenefit`, `Title` — plus the `EnlistmentOffer` above.

**Every strategy is a pure function of the arguments it is handed** — no
hidden state, no randomness, no I/O, no reach into the character. Where a
strategy needs to know that an option is legal, the engine supplies that by
what it puts in the offered set, never by the strategy inspecting the
character. This is what makes each row testable on its own.

**The offered set is never empty**, and a method with only one legal answer
is never called — a question with one answer is not a choice, and calling it
would put a choice event in the log that no reader could have decided
differently. Where that rule bites, the row says so.

## Selecting strategies

Three flags, each choosing a named strategy over one group of rows:

| Flag       | Governs                                                                                              | Values                            | Default    |
| ---------- | ---------------------------------------------------------------------------------------------------- | --------------------------------- | ---------- |
| `--career` | `Service`, `SubmitToDraft`, `AttemptCommission`, `AttemptPromotion`, `ReenlistIntent`, `AssumeTitle` | `serve`, `retire`, `oneterm`      | `serve`    |
| `--skills` | `SkillTable`, `Weapon`                                                                               | `advanced`, `service`, `personal` | `advanced` |
| `--muster` | `MusterTable`, `MusterTable1DM`, `MusterTable2DM`, `MusterWeapon`                                    | `cash`, `goods`, `spartan`        | `cash`     |

A record generated under any non-default strategy names it in the record's
inputs (PRD, CLI sketch). The default triple — `serve`/`advanced`/`cash` —
is what a bare `--auto` run applies, and a record generated under it names
that too, so no record is silent about the policy that made it.

Ranked preferences below are written best-first. A strategy picks the
first of its ranked options that appears in the offered set; where its
ranking is silent, the tie-break is **first-listed order in Book 1**, which
for services is the Prior Service Table's column order (p. 10) and for skill
tables is the Acquired Skills Table's block order (p. 11).

---

## Career rows — `--career`

### `Service(from []EnlistmentOffer) (ServiceName, error)`

**Asked when** no `--service` flag was given. With `--service`, the
enlistment attempt is forced to the named service and the method is not
called — the throw is still made, and a failed throw still goes to the draft
(PRD, CLI sketch).

**Alphabet.** One `EnlistmentOffer` per service of the Prior Service Table
(p. 10), in book order — Navy, Marines, Army, Scouts, Merchants, Other —
each carrying the service's printed enlistment target and the cumulative DM
this character earns against it.

| Strategy          | Answer                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `serve` (default) | The offer whose throw is likeliest to succeed — the 2D probability of meeting its target after its DM. Ties to book order. |
| `retire`          | Same as `serve`.                                                                                                          |
| `oneterm`         | Same as `serve`.                                                                                                          |

The probability is computed from the target and DM the offer carries; it is
not a table of its own and it is not a rule. Two offers of equal probability
are separated by the order the engine listed them in, which is the Prior
Service Table's column order and nothing else.

### `SubmitToDraft() (bool, error)`

**Asked when** the enlistment throw failed. The offer exists at all because
of [E001](ERRATA.md#e001--the-draft-is-may-not-must) — the page prints both
_may_ and _must_, and _may_ governs.

| Strategy          | Answer                                                |
| ----------------- | ----------------------------------------------------- |
| `serve` (default) | yes                                                   |
| `retire`          | yes                                                   |
| `oneterm`         | **no** — generation ends with an 18-year-old civilian |

`oneterm` is the strategy that reaches the civilian record at all, and the
only one whose golden fixture can.

### `AttemptCommission() (bool, error)`

**Asked when** the service has commissions (not Scouts, not Other; p. 6,
p. 10), the character is not yet commissioned, and it is not a draftee's
first term (p. 5).

| Strategy  | Answer |
| --------- | ------ |
| all three | yes    |

Nothing in the rules costs a character anything for a failed commission
throw, and a commission carries a skill eligibility (p. 6). A strategy that
declined would be modelling a preference the book does not price.

### `AttemptPromotion() (bool, error)`

**Asked when** the character is commissioned and holds a rank below the
highest his service prints
([E013](ERRATA.md#e013--no-promotion-throw-at-the-top-of-the-table-of-ranks)).
One attempt per term (p. 6).

| Strategy  | Answer |
| --------- | ------ |
| all three | yes    |

### `ReenlistIntent(from []Intent) (Intent, error)`

**Asked when** the reenlistment throw has been made, was not a 12, and
succeeded, **and** more than one intent is legal. The engine does not ask
otherwise: a 12 forces another term "regardless of his desires" (p. 6), a
failed throw ends the service, and at the end of term 7 or later the only
remaining intent is to leave
([E003](ERRATA.md#e003--a-12-on-the-reenlistment-throw-recurs-past-term-7)).

**Alphabet.** `Continue` | `Discharge` | `Retire`. The engine offers:

| Term just ended | Offered                             |
| --------------- | ----------------------------------- |
| 1–4             | `Continue`, `Discharge`             |
| 5–6             | `Continue`, `Retire`                |
| 7 or later      | _not asked_ — departure is `Retire` |

`Discharge` and `Retire` never coexist: p. 21 makes leaving at the end of the
5th term or later retirement by definition ("is considered to have retired"),
and p. 7 caps voluntary service at seven terms. A `Retire` from the Scouts or
Other is a retirement that pays nothing (p. 21).

| Strategy          | Ranked preference                   |
| ----------------- | ----------------------------------- |
| `serve` (default) | `Continue` > `Retire` > `Discharge` |
| `retire`          | `Retire` > `Continue` > `Discharge` |
| `oneterm`         | `Discharge` > `Retire` > `Continue` |

`serve` takes the full seven terms whenever the throws allow. `retire` leaves
the moment leaving counts as retirement — the end of term 5 (p. 21) — whether
or not that service pays a pension. `oneterm` leaves at the first
opportunity.

### `AssumeTitle(t Title) (bool, error)`

**Asked when** the character survived generation and his final Social
Standing is 11 or greater — once, at the end of generation
([E011](ERRATA.md#e011--when-title-eligibility-is-assessed)). Eligibility is
assessed for the dead too, but they are not asked. `t` is the Book 3 p. 22
title the value confers.

| Strategy  | Answer |
| --------- | ------ |
| all three | yes    |

---

## Skills rows — `--skills`

### `SkillTable(from []SkillTable) (SkillTable, error)`

**Asked once per skill eligibility**, before the die is rolled (p. 11:
"must specify the table being consulted prior to the die throw").

**Alphabet.** Personal Development, Service Skills, Advanced Education,
Advanced Education (Education 8+) — the fourth offered only while the
character's Education stands at 8 or greater at the moment of the
designation.

| Strategy             | Ranked preference                                                                    |
| -------------------- | ------------------------------------------------------------------------------------ |
| `advanced` (default) | Advanced Education (8+) > Advanced Education > Service Skills > Personal Development |
| `service`            | Service Skills > Advanced Education (8+) > Advanced Education > Personal Development |
| `personal`           | Personal Development > Service Skills > Advanced Education (8+) > Advanced Education |

`advanced` is the default because it is the one ranking that makes the
Education 8+ gate visible in a default run: it takes the fourth table the
instant it opens and stops taking it if Education ever falls below 8.
`personal` is the ranking that reaches the procedure's one negative result,
Other's −1 Social (p. 11).

### `Weapon(cat WeaponCategory, from []WeaponName) (WeaponName, error)`

**Asked when** a skills-table result is Blade Cbt or Gun Cbt, immediately
(p. 11, p. 12, p. 13).

**Alphabet.** The blades and polearms list (p. 12) or the guns list (p. 13),
whichever the category names, in printed order.

| Strategy  | Answer                                                                                         |
| --------- | ---------------------------------------------------------------------------------------------- |
| all three | The first name in the printed list for the category — Dagger for blades, Body Pistol for guns. |

Repeat receipts take the same weapon again, which raises its level (p. 12:
"Additional acquisitions of expertise in the same weapon increase the present
level by one"). Concentrating rather than diversifying is deliberate: it is
the branch that produces a level above 1 in a default run, and the diversified
branch is reached through `--muster spartan` below.

---

## Mustering-out rows — `--muster`

### `MusterTable(from []MusterTable) (MusterTable, error)`

**Asked once per benefit roll**, before the die (p. 9: "The player must
designate the table before the die is rolled"). The engine offers Table 2
only while fewer than three rolls have been taken on it (p. 9: "A maximum of
three rolls on table 2 are allowed per character"); once the cap is reached
Table 1 is the only option and the method is not called.

| Strategy         | Ranked preference |
| ---------------- | ----------------- |
| `cash` (default) | Table 2 > Table 1 |
| `goods`          | Table 1 > Table 2 |
| `spartan`        | Table 1 > Table 2 |

`cash` reaches the three-roll cap in any run with three or more rolls, which
is the only way the cap is exercised. `goods` is what reaches the ship
benefits, Travellers' Aid, and Table 1's "—" rows.

### `MusterTable1DM() (bool, error)`

**Asked when** the character's final rank is 5 or 6 (p. 9: "Characters with
rank 5 or 6 may add +1 to their rolls on this table") and the roll is on
Table 1.

| Strategy         | Answer |
| ---------------- | ------ |
| `cash` (default) | yes    |
| `goods`          | yes    |
| `spartan`        | **no** |

### `MusterTable2DM() (bool, error)`

**Asked when** the character has Gambling skill (p. 9: "Individuals with
gambling expertise are allowed a DM of +1 on table 2") and the roll is on
Table 2.

| Strategy         | Answer |
| ---------------- | ------ |
| `cash` (default) | yes    |
| `goods`          | yes    |
| `spartan`        | **no** |

`spartan` exists so that the declined branch of both DMs is reachable by a
generated golden. A branch no fixture exercises is a branch nothing tests
(PRD, Testing and the gate).

### `MusterWeapon(cat, from, received) (WeaponBenefit, error)`

**Asked when** a Table 1 roll is a weapon benefit (p. 9's Blade and Gun
rows). `from` is the category's printed list; `received` is the weapons this
character has already taken **as benefits** — p. 22: "Expertise may only be
taken in a weapon received as a benefit", which is narrower than the weapons
he holds from skills-table results.

**Alphabet.** `TakeWeapon(name)` — a weapon from the list, whether or not he
already has one — or `TakeExpertise(name)`, +1 in a weapon in `received`.
The expertise option is offered only when `received` is non-empty (p. 22:
"in lieu of receiving a second or subsequent weapon of exactly the same
type").

| Strategy         | Answer                                                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cash` (default) | `TakeExpertise` on the first name in `received` when offered; otherwise `TakeWeapon` on the first name in the printed list.                       |
| `goods`          | Same as `cash`.                                                                                                                                   |
| `spartan`        | `TakeWeapon` on the first name in the printed list not already in `received`; if every name is in `received`, the first name in the printed list. |

`cash` and `goods` convert repeats into expertise, which is the branch p. 22's
worked sentence describes. `spartan` diversifies, which is the branch that
fills a character's benefit list with distinct weapons.

---

## Totality

Twelve methods; every one has a row, and every row answers for every offered
set the engine can construct. The policy never returns an error: an offered
set it cannot answer from would mean the engine offered something the rules
do not contain, which is an engine defect and fails there rather than being
absorbed here.

## What this document does not decide

- **Anything a flag on the command line already decides.** `--service`
  forces the enlistment attempt and suppresses `Service`; `--name` supplies
  the name, which is not a choice point (PRD, Decisions).
- **Anything after generation.** Selling a passage at 90% (p. 21), buying
  Travellers' Aid membership for CR 1,000,000 (p. 22), and flying the ship
  are in-play decisions; the record carries the assets and stops.
- **Interactive mode.** It implements the same interface and asks the
  player. The rows above describe only what `--auto` answers.
