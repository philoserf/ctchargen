# COVERAGE

Maps every step and per-service rule of Book 1 pp. 4–25 to its page cite,
implementation, and test. Completeness over pp. 4–25 is milestone 2's exit
criterion; through milestone 1 the map covers the walking skeleton (the
Other service) and lists the rest as pending.

## Implemented (milestone 1)

| Rule                                                                                        | Page  | Implementation                                                      | Test                                    |
| ------------------------------------------------------------------------------------------- | ----- | ------------------------------------------------------------------- | --------------------------------------- |
| Six characteristics, 2D each, in rolled order; start age 18                                 | 4     | `chargen/engine.go` `characteristics`                               | golden fixtures                         |
| Values 1–15 through play, never above 15                                                    | 4     | `chargen/record.go` `Apply`                                         | `TestApplyClamps`                       |
| UPP hexadecimal notation, 10–15 as A–F                                                      | 8     | `chargen/record.go` `UPP`                                           | `TestUPP` (against the p. 25 example)   |
| One enlistment attempt; throw with cumulative DMs                                           | 5, 10 | `chargen/engine.go` `enlistment`                                    | golden fixtures                         |
| Other: enlistment 3+, draft 6, survival 5+ (+2 Intel 9+), reenlist 5+, no commissions/ranks | 10    | `service/data/other.json`                                           | `TestOtherDefinition`                   |
| Draft on rejection; may decline (E001) → civilian record                                    | 5     | `chargen/engine.go` `enlistment`                                    | `civilian-declined-draft` fixture       |
| Terms of 4 years; age advances per term                                                     | 5     | `chargen/engine.go` `term`                                          | golden fixtures                         |
| Survival throw; failure is death, recorded (E003)                                           | 5     | `chargen/engine.go` `term`                                          | `death-in-service` fixture              |
| Skill eligibility: 2 initial term, 1 per subsequent                                         | 6     | `chargen/engine.go` `skills`                                        | golden fixtures                         |
| Four skills tables; fourth gated on Education 8+; table declared before the die             | 11    | `chargen/engine.go` `skills` + `service/data/other.json`            | golden fixtures                         |
| Characteristic alterations applied immediately                                              | 12    | `chargen/engine.go` `applySkillResult`                              | golden fixtures                         |
| Weapon expertise: specific weapon chosen immediately                                        | 11–13 | `chargen/engine.go` `applySkillResult`, `service/data/weapons.json` | `TestWeaponsListsInBookOrder`, fixtures |
| Skills accumulate Skill-1, Skill-2, … with no cap                                           | 13    | `chargen/record.go` `AddSkill`                                      | `TestAddSkill`                          |
| Reenlistment thrown every term; failure forces out; 12 exactly forces stay; no DMs          | 6–7   | `chargen/engine.go` `reenlistment`, `service/service.go` validation | golden fixtures                         |
| Voluntary service caps at 7 terms                                                           | 7     | `chargen/engine.go` `reenlistment`                                  | golden fixtures                         |
| Die roll conventions (2D, DMs, N+/N−/exact targets)                                         | 2–3   | `dice/dice.go`                                                      | `dice` package tests                    |

## Pending

| Rule                                                    | Page        | Arrives with |
| ------------------------------------------------------- | ----------- | ------------ |
| The other five services (enlistment DMs, tables, ranks) | 10–11       | milestone 2  |
| Commissions and promotions; draftees' first-term bar    | 5–6, 10     | milestone 2  |
| Rank and Service Skills box                             | 23          | milestone 2  |
| Aging table, per-term crossings from 34; medical crisis | 7–9         | milestone 3  |
| Retirement and retirement pay                           | 7, 21       | milestone 3  |
| Mustering out: rolls, tables, benefits, ships           | 7, 9, 21–23 | milestone 3  |
| Titles (Social 11+; Book 3 nobility)                    | 4; B3 22    | milestone 3  |
| Interactive mode, batch, replay subcommand              | —           | milestone 4  |
