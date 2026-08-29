# ctchargen

A Go CLI that generates rules-accurate Classic Traveller characters per the
Books 1–3 prior-service procedure (© 1977 text): characteristics, service,
terms, skills, aging, mustering out — or death, which is a completed
generation too. Sibling to [t5chargen](https://github.com/philoserf/t5chargen).

[`docs/PRD.md`](docs/PRD.md) is the v1 contract; all four milestones are
implemented. `COVERAGE.md` maps every rule of Book 1 pp. 4–25 to its page
cite, implementation, and test; `ERRATA.md` records every interpretation;
`POLICY.md` is the auto mode's decision table.

## Usage

```sh
ctchargen new --auto                     # one character, policy decides, JSON to stdout
ctchargen new --seed 42 --service navy   # interactive: you answer each choice
ctchargen batch --count 20 --auto -o npcs.jsonl
ctchargen render character.json          # Markdown character sheet
ctchargen render --history character.json# the full generation transcript
ctchargen replay character.json          # verify a record reproduces exactly
ctchargen version
```

Every record carries its seed, versions, inputs, and the complete event
log; `replay` re-runs the engine from the seed and recorded choices and
exits non-zero at the first mismatch. Death is an outcome, not an error:
the dead get complete records too.

## Development

```sh
task deps   # install the toolchain (brew bundle)
task hooks  # install the pre-push gate
task        # check (modernize + fmt + vet + lint) + test
```
