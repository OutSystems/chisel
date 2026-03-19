# Contributing to Chisel

## Prerequisites

- **Go 1.25.1+** (see `go.mod` for exact version)
- **goreleaser** (for builds and releases)
- **gocover-cobertura** (for coverage reports)

Install build dependencies:

```sh
make dep
```

## Project Structure

```
main.go          # CLI entrypoint (server + client subcommands)
client/          # Client package
server/          # Server package
share/           # Shared libraries
  ccrypto/       #   Cryptographic utilities (key generation, SSH keys)
  cio/           #   I/O helpers (logging, pipes, stdio)
  cnet/          #   Networking (WebSocket connections, HTTP server, metering)
  cos/           #   OS utilities (signals, pprof)
  settings/      #   Configuration (remotes, users, environment)
  tunnel/        #   Tunnel implementation (TCP/UDP proxying)
test/
  e2e/           # End-to-end tests (auth, TLS, SOCKS, UDP, proxy)
  bench/         # Benchmarks
example/         # Example config files (users.json, deployment configs)
```

## Building

Build for the current platform:

```sh
make all
```

Cross-compile for a specific OS:

```sh
make linux
make darwin
make windows
make freebsd
```

Build a Docker image:

```sh
make docker
```

## Running Tests

Unit tests (excludes the `test/` package):

```sh
make test
```

This runs tests with race detection, generates a coverage report in `./build/`, and produces a Cobertura XML for CI.

Run all tests directly (including e2e):

```sh
go test ./...
```

Run only e2e tests:

```sh
go test ./test/e2e/
```

## Linting

Format and vet the code:

```sh
make lint
```

This runs `go fmt` and `go vet` across all packages (excluding `test/`).

## Code Conventions

- Standard Go formatting (`go fmt`)
- No external linting tools beyond `go vet`
- Tests use the `_test` package suffix for e2e tests (e.g., `package e2e_test`)
- Unit tests live alongside their source files (e.g., `client/client_test.go`)
- E2e tests use a shared `testLayout` setup helper in `test/e2e/setup_test.go`

## Release Process

Releases are handled via goreleaser. The configuration lives in two places:

- `.github/goreleaser.yml` -- upstream multi-platform release config
- `goreleaser.yml` -- OutSystems-specific config (Linux only, with Docker image to `ghcr.io`)

To do a dry-run release:

```sh
goreleaser release --config .github/goreleaser.yml --clean --snapshot
```

To create an actual release:

```sh
make release
```

This runs lint, tests, and then `goreleaser release`.

## Pull Request Guidelines

1. Fork or branch from `master`
2. Keep changes focused -- one concern per PR
3. Run `make lint` and `make test` before pushing
4. Ensure your changes compile across platforms (`CGO_ENABLED=0` is used for most builds)
5. Add or update tests for new functionality, especially in `test/e2e/` for tunnel behavior changes
6. Dependabot manages dependency updates on a monthly cycle -- don't batch unrelated dep bumps with feature work
