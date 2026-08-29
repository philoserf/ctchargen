# POLICY

`policy_version`: **3**

The auto policy is the fixed decision table applied when `--auto` decides
the choice points. It is **total** — it can decide every valid choice
point the engine can present — and **deterministic**, tie-breaking by
first-listed order in Book 1. Every record stamps the `policy_version`
that generated it; replay never consults the policy (recorded choices are
reapplied), so records made under any policy replay under any other.

Changing any row below is a `policy_version` bump.

| Choice point         | Decision                                                                                                                                            |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `service`            | The first-listed service, in Book 1's order (p. 5): Navy, Marines, Army, Scouts, Merchants, Other — so Navy.                                        |
| `submit-to-draft`    | Yes — a rejected character submits to the draft (p. 5).                                                                                             |
| `commission-attempt` | Yes — attempt whenever eligible (p. 5).                                                                                                             |
| `promotion-attempt`  | Yes — attempt whenever eligible (p. 6).                                                                                                             |
| `skill-table`        | `service_skills`, every eligibility.                                                                                                                |
| `weapon`             | The first-listed weapon of the category, in the order of the pp. 12–13 lists (Dagger; Body Pistol).                                                 |
| `reenlist-intent`    | Yes — reenlist while the rules allow it. The 7-term cap and the 12-exactly rule are the rules' (pp. 6–7), not policy.                               |
| `muster-table`       | Cash (Table 2) while the three-roll cap allows, then material benefits (Table 1) (pp. 7, 9).                                                        |
| `benefit-dm`         | Yes — the rank 5–6 +1 on Table 1 is always taken (p. 9).                                                                                            |
| `cash-dm`            | Yes — the gambling +1 on Table 2 is always taken (p. 9).                                                                                            |
| `muster-weapon`      | +1 expertise in the first-listed already-received benefit weapon when the option exists (p. 22); otherwise the first-listed weapon of the category. |
| `assume-title`       | Yes — an eligible hereditary title is assumed (p. 4; Book 3 p. 22).                                                                                 |

## History

- **3** (2026-08-29): milestone 3 — `muster-table`, `benefit-dm`,
  `cash-dm`, `muster-weapon`, and `assume-title` rows added.
- **2** (2026-08-29): all six services available — `service` now picks
  Navy; `commission-attempt` and `promotion-attempt` rows added.
- **1** (2026-08-29): milestone 1 — Other the only service.
