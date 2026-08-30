# Brewfile — development toolchain for ctchargen.
# Install with `task deps` (or `brew bundle`).

brew "go"             # Go compiler/toolchain (see go.mod for required version)
brew "go-task"        # `task` runner — this project's build gate
brew "gofumpt"        # stricter gofmt; enforced via `task fmt`
brew "golangci-lint"  # meta-linter; enforced via `task lint`
brew "prettier"       # JSON and Markdown formatter; enforced via `task fmt:check:docs`

# Not everything the gate needs is a brew package: NilAway has no formula
# and no tagged release, so `task deps` installs it with `go install` at
# @latest — its current tip commit. See the deps task.
#
# Nothing here is version-pinned, and CI installs the same tools unpinned:
# the gate is meant to fail when the tooling moves rather than drift
# behind it. See the comment in .github/workflows/ci.yml.
