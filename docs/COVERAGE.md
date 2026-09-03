# COVERAGE: the procedure, mapped to the code

2026-09-02. Milestone 1. Companion to `PRD.md`.

Every rule of Book 1 pp. 4–25 that this tool implements, with the page that
governs it, the code that carries it, and the test that holds it. A rule with
no row here is a rule that is not implemented; a row with no test is a defect.

The tables of pp. 4–25 are all lifted, whether or not the engine consults
them yet, and each is transcribed twice — once in `rules/data`, once in the
`rules` tests, from the same visual reading of the page. Those rows say
"lifted, not yet consulted" where the engine has no path to them.

## What the engine walks

Milestone 1's engine runs the two services that print no rank column: **Other
and Scouts**. P. 10 makes ranks, commissions and promotions one fact — "Ranks,
commissions, and promotions are non-existent in the scout and other services"
— so a service with any of them has all three, and those arrive at milestone
2 together with the Jamison reproduction.

## Characteristics and the record

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| Six 2D rolls, in order | 1:4 | `chargen.run.rollProfile` | `chargen.TestGoldens` |
| Values 2–12 initially; never above 15 | 1:4 | `traveller.Profile.Alter` | `traveller.TestAlterHoldsTheOrdinaryFloorAndTheCeiling` |
| Floor of 1 for every ordinary alteration | 1:4 (E010) | `traveller.Profile.Alter` | `traveller.TestTheTwoFloorsDiffer` |
| Floor of 0 for an aging reduction | 1:4, 7–9 (E010) | `traveller.Profile.AgeReduce` | `traveller.TestAgeReduceFloorsAtZero` |
| All characters begin at 18 | 1:4 | `chargen.startingAge` | `chargen.TestGoldens` |
| UPP as hexadecimal, in rolled order | 1:8 | `traveller.Profile.UPP` | `traveller.TestUPP` |
| Die roll conventions: `N+`, `N−`, exact | 1:2–3 | `traveller.Target` | `traveller.TestParseTarget`, `TestTargetSatisfied` |
| Two-die throws except the draft | 1:10 | `chargen.roll`, `dice.Stream` | `dice.TestTwoDiceIsTwoDiceInOrder` |

## Enlistment and the draft

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| One enlistment attempt, against the Prior Service Table | 1:5, 10 | `chargen.run.enlist` | `chargen.TestGoldens` |
| Cumulative characteristic DMs | 1:10 | `rules.Throw.Modifier` | `rules.TestModifierIsCumulative` |
| The draft is offered, not compelled | 1:5 (E001) | `chargen.run.draft` | golden `scouts-civilian` |
| One die, entering the service with that draft number | 1:5 | `rules.Rules.Draft` | `rules.TestDraft` |
| Declining ends generation with a civilian | 1:5 (E001) | `chargen.run.draft` | golden `scouts-civilian` |
| Draftees are not eligible for commission in the first term | 1:5 | *not yet: needs a service with ranks* | — |

## Terms of service

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| A term is four years | 1:5 | `traveller.Years` | `chargen.TestACareerPastTheTablesLastColumn` |
| Per-term order is the exposition's | 1:5–6 (E002) | `chargen.run.term` | golden transcripts |
| Survival throw; failure is death | 1:5 | `chargen.run.survive` | golden `other-death` |
| The dead character's age and term | 1:5, 9 (E004) | `chargen.run.term` | golden `other-death` |
| Reenlistment throw every term, no DMs | 1:6 | `chargen.run.reenlist` | golden transcripts |
| A 12 exactly forces another term | 1:6 | `chargen.run.reenlist` | `chargen.TestACareerPastTheTablesLastColumn` |
| A 12 recurs past the seventh term | 1:6–7, 21 (E003) | `chargen.run.reenlist` | `chargen.TestACareerPastTheTablesLastColumn` |
| Seven terms of voluntary service | 1:7 | `chargen.lastVoluntaryTerm` | `chargen.TestVoluntaryServiceStopsAtSeven` |
| Leaving at the end of term 5 or later is retirement | 1:21 | `chargen.run.depart` | `chargen.TestVoluntaryServiceStopsAtSeven` |
| Commission and promotion throws | 1:6, 10 | *not yet: milestone 2* | — |
| No promotion at the top of the ranks table | 1:6, 10 (E013) | *not yet: milestone 2* | — |

## Skills and training

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| Two eligibilities for the initial term, one thereafter | 1:6 | `chargen.run.grantEligibility` | `rules.TestEligibility` |
| The table is designated before the die | 1:11 | `chargen.run.trainOnce` | golden transcripts |
| The fourth table needs Education 8+ | 1:11 | `chargen.run.offeredTables` | `rules.TestEligibility` |
| Three kinds of result | 1:12 | `chargen.applyResult` | `traveller.TestTableResultFolds` |
| Blade and gun combat name a weapon at once | 1:11–13 | `chargen.applyResult.WeaponPick` | golden transcripts |
| Skills accumulate as Skill-1, Skill-2, with no cap | 1:12 | `chargen.Character.addSkill` | golden sheets |
| Service-wide rank and service skills, granted once on entering | 1:23 (E005) | `chargen.run.grantsOnEntering` | golden `scouts-retire` |
| Rank-keyed grants | 1:23 | *not yet: milestone 2* | — |
| Names normalized to their description headings | 1:11–23 (E012) | `rules.Rules.Normalize` | `rules.TestNormalization` |

## Aging

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| A round at the end of term 4 and every term after | 1:7, 9 (E006) | `chargen.run.agingRound` | golden `other-serve` |
| Read off the table's term row, not its age row | 1:9 (E006) | `rules.Aging.At` | `rules.TestAgingTable` |
| Saving throws in the table's row order | 1:9 (E007) | `chargen.run.agingRound` | golden transcripts |
| The last column is terminal | 1:9 (E014) | `rules.Aging.At` | `chargen.TestACareerPastTheTablesLastColumn` |
| A characteristic at zero is a medical crisis, resolved inline | 1:7–8 (E007) | `chargen.run.crisis` | golden `other-crisis-survived` |
| Saving throw 8+, with no modifier during generation | 1:7 (E009) | `chargen.run.crisis` | `rules.TestMedicalCrisis` |
| Survival recovers to 1 and adds 1D months | 1:7–8 | `chargen.run.crisis` | golden `other-crisis-survived` |
| A failed crisis throw is death | 1:7–8 (E008) | `chargen.run.crisis` | golden `other-crisis-died` |

## Mustering out

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| One roll per term, plus rank extras | 1:7, 9 | `rules.Muster.Rolls` | `rules.TestMusterRollsAndPassages` |
| The table is designated before the die | 1:9 | `chargen.run.chooseMusterTable` | golden transcripts |
| At most three rolls on Table 2 | 1:9 | `chargen.run.chooseMusterTable` | `rules.TestMusterRollsAndPassages` |
| The +1 at rank 5 or 6 on Table 1 | 1:9 | `chargen.run.musterModifier` | *no golden: needs ranks (milestone 2)* |
| The +1 with gambling on Table 2 | 1:9 | `chargen.run.musterModifier` | golden transcripts |
| The seven kinds of Table 1 row | 1:9, 21–23 | `chargen.applyBenefit` | `traveller.TestBenefitRowFolds` |
| The dash rows deliver nothing | 1:9 | `rules` lift | `rules.TestTheDashCellsAreNothing` |
| Travellers' Aid only once; duplicates wasted | 1:22 | `chargen.applyBenefit.TravellersAid` | `traveller.TestBenefitRowFolds` |
| A repeat weapon may be taken as expertise | 1:22 | `chargen.takeWeapon` | `chargen.TestMusterWeaponStrategies` |
| Free Trader: 40 years of payments, 10 off per repeat | 1:22–23 | `chargen.run.receiveShipAgain` | *no golden: only Merchants award one* |
| Scout ship: duplicates lost | 1:23 | `chargen.run.receiveShipAgain` | *no golden: only Scouts award one* |
| Retirement pay from term 5, not for Scouts or Other | 1:7, 21 | `chargen.run.pension` | `rules.TestRetirementPay` |

## Titles

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| Social Standing 11+ may assume the hereditary title | 1:5; 3:22 | `chargen.run.assessTitle` | golden `other-title` |
| Assessed once, at the end, against the final value | 1:5; 3:22 (E011) | `chargen.run.assessTitle` | golden `other-title` |
| The dead are assessed but not asked | (E011) | `chargen.run.assessTitle` | golden `other-death` |
| The five ranks of nobility | 3:22 | `rules.Rules.TitleFor` | `rules.TestNobility` |

## The record

| Rule | Page | Implementation | Test |
| --- | --- | --- | --- |
| Every step, throw, choice and consequence, in order | FR11 | `chargen.log` | golden transcripts |
| Each choice names who decided and what was offered | FR11 | `chargen.logging` | golden transcripts |
| Every reading that governed a record is named on it | Authority | `chargen.log.stamped` | `internal/docsgate` |
| One seed reproduces one character | Determinism | `dice.Stream` | `chargen.TestGoldensRegenerate` |
