# POLICY

`policy_version`: **1**

The auto policy is the fixed decision table applied when `--auto` decides
the choice points. It is **total** — it can decide every valid choice
point the engine can present — and **deterministic**, tie-breaking by
first-listed order in Book 1. Every record stamps the `policy_version`
that generated it; replay never consults the policy (recorded choices are
reapplied), so records made under any policy replay under any other.

Changing any row below is a `policy_version` bump.

| Choice point      | Decision                                                                                                                                                  |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `service`         | The first-listed available service, in Book 1's order (p. 5): Navy, Marines, Army, Scouts, Merchants, Other. Through milestone 1 only Other is available. |
| `submit-to-draft` | Yes — a rejected character submits to the draft (p. 5).                                                                                                   |
| `skill-table`     | `service_skills`, every eligibility.                                                                                                                      |
| `weapon`          | The first-listed weapon of the category, in the order of the pp. 12–13 lists (Dagger; Body Pistol).                                                       |
| `reenlist-intent` | Yes — reenlist while the rules allow it. The 7-term cap and the 12-exactly rule are the rules' (pp. 6–7), not policy.                                     |
