# ctchargen

A Go CLI that generates rules-accurate Classic Traveller characters.

Ruleset baseline: **Books 1–3 only** — the FFE reprints of the © 1977 text.
Character generation is Book 1 pp. 4–25; Books 2 and 3 are consulted only
where Book 1 points at them. Every implemented rule carries its printed-page
cite, and every place the text is silent or ambiguous has a recorded reading.

## Status

Milestone 1, in progress. There is a gate, a seeded dice stream, and nothing
yet that generates a character.

## The documents

| File | What it holds |
| --- | --- |
| [docs/PRD.md](docs/PRD.md) | The v1 contract: goals, domain model, requirements, determinism, milestones. |
| [docs/ERRATA.md](docs/ERRATA.md) | Every recorded reading, with its page cite and its stamping condition. |
| [docs/POLICY.md](docs/POLICY.md) | The `--auto` decision table, one row per choice point. |
| [CLAUDE.md](CLAUDE.md) | Authority, source precedence, and the working rules for agents. |

## The gate

```sh
task
```

Formatting, `go vet`, golangci-lint, NilAway, `go test -race`, and a coverage
ratchet that holds each package's count of uncovered statements. CI runs
exactly this. The toolchain is unpinned on purpose.

## Licence

[MIT](LICENSE).
