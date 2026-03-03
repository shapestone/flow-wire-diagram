# Contributing to flow-wire-diagram

Thank you for your interest in contributing.

## Reporting Bugs

Open a GitHub issue using the **Bug Report** template. Include:
- The Markdown input that triggered the defect
- The output you got vs. what you expected
- Go version and OS

## Suggesting Features

Open a GitHub issue using the **Feature Request** template. Describe the use case before proposing a solution.

## Pull Requests

1. Fork the repository and create a branch from `main`
2. Make your changes
3. Ensure all tests pass: `make test`
4. Ensure `go vet` is clean: `make lint`
5. Add or update tests for any changed behaviour
6. Open a PR against `main` — fill in the PR template

## Development Setup

```bash
git clone https://github.com/shapestone/flow-wire-diagram.git
cd flow-wire-diagram
go mod download
make build   # builds bin/wire-fix
make test    # runs all tests
make lint    # runs go vet
```

## Code Style

- Standard Go formatting (`gofmt`)
- No external dependencies beyond `github.com/mattn/go-runewidth`
- Hexagonal architecture: domain types in `domain/`, use-cases in `application/`, I/O in `infrastructure/`
- New public behaviour must have a test in `repair_test.go`, `parse_test.go`, or `wirediagram_test.go`

## Testing

```bash
make test                          # all tests
go test ./... -run TestRepair      # specific tests
go test -fuzz=FuzzRepair ./internal/diagram/infrastructure  # fuzz
```

Property invariants enforced by `TestRepairProperties`:
1. Text content is never altered — only structural characters move
2. `VerifyDiagram` reports no defects after repair
3. Repair is idempotent: `repair(repair(x)) == repair(x)`

## Commit Messages

Use the imperative mood and keep the first line under 72 characters.
Reference issues with `Fixes #N` in the body when applicable.
