# COVERAGE

Maps every step and per-service rule of Book 1 pp. 4–25 to its page cite,
implementation, and test. Every rule is mapped; the Pending section's
rows carry their milestone instead of an implementation.

## Implemented

| Rule                                                                                                          | Page    | Implementation                                                             | Test                                                                      |
| ------------------------------------------------------------------------------------------------------------- | ------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| Six characteristics, 2D each, in rolled order; start age 18                                                   | 4       | `chargen/engine.go` `characteristics`                                      | golden fixtures                                                           |
| Values 1–15 through play, never above 15                                                                      | 4       | `chargen/record.go` `Apply`                                                | `TestApplyClamps`                                                         |
| UPP hexadecimal notation, 10–15 as A–F                                                                        | 8       | `chargen/record.go` `UPP`                                                  | `TestUPP` (against the p. 25 example)                                     |
| One enlistment attempt; throw with cumulative DMs                                                             | 5, 10   | `chargen/engine.go` `enlistment`                                           | golden fixtures                                                           |
| Prior Service Table: all six services' enlistment/draft/survival/commission/promotion/reenlist throws and DMs | 10      | `service/data/*.json`                                                      | `service` tests + golden fixtures                                         |
| Table of Ranks (Merchants stop at Captain, rank 5)                                                            | 10      | `service/data/*.json` `ranks`                                              | ranked golden fixtures                                                    |
| Commission: once per term until achieved; not draftees' first term; not Scouts/Other; rank 1                  | 5–6, 10 | `chargen/engine.go` `commission`                                           | `marines-careerist`, `draftee` fixtures                                   |
| Promotion: one per term, commissioned only, from the commission term on; next higher rank                     | 6, 10   | `chargen/engine.go` `promotion`                                            | `marines-careerist`, `army-careerist` fixtures                            |
| +1 skill eligibility each for commission and promotion                                                        | 6       | `chargen/engine.go` `commissionAndPromotion`                               | `marines-careerist` fixture (term 1: 2+1 rolls)                           |
| Rank and Service Skills box (timing reading E004)                                                             | 23      | `chargen/engine.go` `grantAutoSkills`, `service/data/*.json` `auto_skills` | `marines-careerist` (Cutlass-1, Revolver-1), `scouts-careerist` (Pilot-1) |
| `--service` forces the enlistment attempt only; a failed throw still drafts anywhere                          | 5       | `cmd/ctchargen/main.go`, `chargen/engine.go` `enlistment`                  | per-service fixtures, `draftee` fixture                                   |
| Draft on rejection; may decline (E001) → civilian record                                                      | 5       | `chargen/engine.go` `enlistment`                                           | `civilian-declined-draft` fixture                                         |
| Terms of 4 years; age advances per term                                                                       | 5       | `chargen/engine.go` `term`                                                 | golden fixtures                                                           |
| Survival throw; failure is death, recorded (E003)                                                             | 5       | `chargen/engine.go` `term`                                                 | `death-in-service` fixture                                                |
| Skill eligibility: 2 initial term, 1 per subsequent                                                           | 6       | `chargen/engine.go` `skills`                                               | golden fixtures                                                           |
| Four skills tables; fourth gated on Education 8+; table declared before the die                               | 11      | `chargen/engine.go` `skills` + `service/data/other.json`                   | golden fixtures                                                           |
| Characteristic alterations applied immediately                                                                | 12      | `chargen/engine.go` `applySkillResult`                                     | golden fixtures                                                           |
| Weapon expertise: specific weapon chosen immediately                                                          | 11–13   | `chargen/engine.go` `applySkillResult`, `service/data/weapons.json`        | `TestWeaponsListsInBookOrder`, fixtures                                   |
| Skills accumulate Skill-1, Skill-2, … with no cap                                                             | 13      | `chargen/record.go` `AddSkill`                                             | `TestAddSkill`                                                            |
| Reenlistment thrown every term; failure forces out; 12 exactly forces stay; no DMs                            | 6–7     | `chargen/engine.go` `reenlistment`, `service/service.go` validation        | golden fixtures                                                           |
| Voluntary service caps at 7 terms                                                                             | 7       | `chargen/engine.go` `reenlistment`                                         | golden fixtures                                                           |
| Die roll conventions (2D, DMs, N+/N−/exact targets)                                                           | 2–3     | `dice/dice.go`                                                             | `dice` package tests                                                      |

## Pending

| Rule                                                    | Page        | Arrives with |
| ------------------------------------------------------- | ----------- | ------------ |
| Aging table, per-term crossings from 34; medical crisis | 7–9         | milestone 3  |
| Retirement and retirement pay                           | 7, 21       | milestone 3  |
| Mustering out: rolls, tables, benefits, ships           | 7, 9, 21–23 | milestone 3  |
| Titles (Social 11+; Book 3 nobility)                    | 4; B3 22    | milestone 3  |
| Interactive mode, batch, replay subcommand              | —           | milestone 4  |
