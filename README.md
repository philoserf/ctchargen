# ctchargen

A Go CLI that generates rules-accurate Classic Traveller characters per the
Books 1–3 prior-service procedure (© 1977 text): characteristics, service,
terms, skills, aging, mustering out — or death, which is a completed
generation too. Sibling to [t5chargen](https://github.com/philoserf/t5chargen).

**Status: pre-implementation.** [`docs/PRD.md`](docs/PRD.md) is the v1
contract.

## Development

```sh
task deps   # install the toolchain (brew bundle)
task hooks  # install the pre-push gate
task        # check (modernize + fmt + vet + lint) + test
```
