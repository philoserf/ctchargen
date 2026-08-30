# POLICY

`policy_version`: **4**

The auto policy is the decision table applied when `--auto` decides the
choice points. It is **total** — it can decide every valid choice point the
engine can present — and **deterministic**: the same inputs always give the
same character, with ties broken by first-listed order in Book 1.

Three rows are **selectable**, by the flags in the Strategies section
below. The rest are fixed. Every record stamps the `policy_version` that
generated it, and a record generated under a non-default strategy also
records which, in its `inputs` block — `policy_version` names this
document, `inputs` names the selection within it.

Replay never consults the policy at all: recorded choices are reapplied, so
records made under any policy or strategy replay under any other.

Changing any row or strategy below is a `policy_version` bump.

## Decision table

The default decision at every choice point. A selectable row shows its
default here and its alternatives under Strategies.

| Choice point         | Decision                                                                                                                                            |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `service`            | The first-listed service, in Book 1's order (p. 5): Navy, Marines, Army, Scouts, Merchants, Other — so Navy.                                        |
| `submit-to-draft`    | Yes — a rejected character submits to the draft (p. 5).                                                                                             |
| `commission-attempt` | Yes — attempt whenever eligible (p. 5).                                                                                                             |
| `promotion-attempt`  | Yes — attempt whenever eligible (p. 6).                                                                                                             |
| `skill-table`        | **Selectable** (`--skills`). Default: `service_skills`, every eligibility.                                                                          |
| `weapon`             | The first-listed weapon of the category, in the order of the pp. 12–13 lists (Dagger; Body Pistol).                                                 |
| `reenlist-intent`    | **Selectable** (`--career`). Default: yes — reenlist while the rules allow it. The 7-term cap and the 12-exactly rule are the rules' (pp. 6–7).     |
| `muster-table`       | **Selectable** (`--muster`). Default: cash (Table 2) while the three-roll cap allows, then material benefits (Table 1) (pp. 7, 9).                  |
| `benefit-dm`         | Yes — the rank 5–6 +1 on Table 1 is always taken (p. 9).                                                                                            |
| `cash-dm`            | Yes — the gambling +1 on Table 2 is always taken (p. 9).                                                                                            |
| `muster-weapon`      | +1 expertise in the first-listed already-received benefit weapon when the option exists (p. 22); otherwise the first-listed weapon of the category. |
| `assume-title`       | Yes — an eligible hereditary title is assumed (p. 5; Book 3 p. 22).                                                                                 |

## Strategies

Each strategy is a pure function of the choice it is handed: the step
carries the term, and the options carry what the rules currently allow. No
strategy consults the dice or carries state between choices, which is what
keeps the policy deterministic.

### `--skills` — the `skill-table` row

The default suppresses two whole tables. Because Service Skills is always
chosen, a character never rolls on Personal Development, so never gains a
characteristic in service; and never rolls on Advanced Education, so
Medical, Navigation, Computer, Leader and Administration are unreachable
except through the p. 23 rank grants. It compounds — the fourth table needs
Education 8+, and Education rises only on Personal Development.

| Strategy   | Decision                                                                                                                                                                                                   |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `service`  | Default. Every eligibility to the service's Service Skills table.                                                                                                                                          |
| `personal` | Every eligibility to Personal Development, the first table (p. 11).                                                                                                                                        |
| `advanced` | The most advanced table on offer — the Education 8+ table where the character has opened it, otherwise Advanced Education.                                                                                 |
| `rounded`  | One term on each of the first three tables, in the book's order, cycling: a term improving himself, a term learning the trade, a term specialising. Every eligibility of a term goes to that term's table. |

### `--muster` — the `muster-table` row

| Strategy   | Decision                                                                                                                   |
| ---------- | -------------------------------------------------------------------------------------------------------------------------- |
| `cash`     | Default. Cash while the three-roll cap allows (p. 9), then material benefits.                                              |
| `benefits` | Table 1 only. Cash is never rolled, so the character musters out with no money — a real choice, and rarely the useful one. |

### `--career` — the `reenlist-intent` row

`max` is the default: reenlist while the rules allow. Otherwise a term
number from 1 to 7, after which the character intends to leave.

**Intent, not outcome.** The reenlistment throw is still required in every
term (p. 6), a 12 exactly still forces another term regardless of desires
(pp. 6–7), and survival still governs. `--career 4` produces a character
who _tries_ to leave after four terms, not one who serves exactly four.

Its values are a term number rather than a fixed set, so unlike the other
two it carries no strategy names.

## History

- **4** (2026-08-30): `skill-table`, `muster-table`, and `reenlist-intent`
  became selectable via `--skills`, `--muster`, and `--career`. Every
  default is unchanged, so a record generated without those flags differs
  from a version 3 record only in its version stamps.
- **3** (2026-08-29): milestone 3 — `muster-table`, `benefit-dm`,
  `cash-dm`, `muster-weapon`, and `assume-title` rows added.
- **2** (2026-08-29): all six services available — `service` now picks
  Navy; `commission-attempt` and `promotion-attempt` rows added.
- **1** (2026-08-29): milestone 1 — Other the only service.
