# ctchargen

A Go CLI that generates rules-accurate Classic Traveller characters.

Ruleset baseline: **Books 1–3 only** — the FFE reprints of the © 1977 text.
Character generation is Book 1 pp. 4–25; Books 2 and 3 are consulted only
where Book 1 points at them. Every implemented rule carries its printed-page
cite, and every place the text is silent or ambiguous has a recorded reading.

## Status

All six services generate, the book's own worked character (pp. 23–25)
replays against the engine, the record has a schema, and characters can be
written to files, generated in batches and read back. `new` without
`--auto` walks the procedure a question at a time, showing each throw
between the questions.

```sh
ctchargen new --auto --seed 145 --service merchants --sheet
ctchargen batch --count 20 --auto --seed 145 --service merchants -o characters/
ctchargen render characters/00000000000000000145.json
ctchargen new --seed 145 --sheet    # asks at every choice point
```

Batch members number from zero, so the first of that batch is the character
the first command generated, and the third command shows the same sheet.

## The documents

| File | What it holds |
| --- | --- |
| [docs/PRD.md](docs/PRD.md) | The v1 contract: goals, domain model, requirements, determinism, milestones. |
| [docs/ERRATA.md](docs/ERRATA.md) | Every recorded reading, with its page cite and its stamping condition. |
| [docs/POLICY.md](docs/POLICY.md) | The `--auto` decision table, one row per choice point. |
| [docs/COVERAGE.md](docs/COVERAGE.md) | Every implemented rule mapped to its page cite, its implementation and its test. |
| [docs/character.schema.json](docs/character.schema.json) | What the tool writes, with a minimal and a complete example beside it. |
| [CLAUDE.md](CLAUDE.md) | Authority, source precedence, and the working rules for agents. |

## The gate

```sh
task
```

`go mod tidy -diff`, `go vet`, golangci-lint, NilAway, `go test -race`, and a
coverage ratchet that holds each package's count of uncovered statements. CI
runs exactly this.

The toolchain is unpinned on purpose, and golangci-lint runs with
`default: all`, so a linter added upstream arrives switched on. Formatting is
`gofumpt`, run *inside* golangci-lint rather than beside it, so there is one
definition of formatted rather than two that can disagree.

## Licence

[MIT](LICENSE).
