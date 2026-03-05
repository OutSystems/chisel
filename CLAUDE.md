# CLAUDE.md

Chisel is a TCP/UDP tunneling tool transported over HTTP and secured via SSH. A single Go binary acts as both client and server. This is an OutSystems fork of [jpillora/chisel](https://github.com/jpillora/chisel).

## Key Commands

```sh
make all              # Build for current platform (uses goreleaser)
make test             # Unit tests with race detection and coverage (excludes test/)
make lint             # go fmt + go vet (excludes test/)
go test ./...         # All tests including e2e
go test ./test/e2e/   # Only e2e tests
make release          # Lint + test + goreleaser release
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for full build, test, and release details.

## Directory Overview

| Directory | Contents |
|---|---|
| `main.go` | CLI entrypoint -- parses `server`/`client` subcommand, delegates |
| `client/` | Client: WebSocket connection, SSH setup, reconnection with backoff |
| `server/` | Server: HTTP/TLS listener, WebSocket upgrade, SSH auth, user permissions |
| `share/` | Shared libraries used by both client and server |
| `share/tunnel/` | Core tunnel engine -- proxy, SSH channels, keepalive, UDP mux |
| `share/settings/` | Config parsing, remote format, user/auth, `CHISEL_*` env vars |
| `share/ccrypto/` | ECDSA key generation, SSH fingerprinting |
| `share/cnet/` | WebSocket-to-net.Conn adapter, HTTP server with graceful shutdown |
| `share/cio/` | Bidirectional pipe, logging, stdio |
| `share/cos/` | OS signals (SIGUSR2 stats, SIGHUP reconnect), context helpers |
| `test/e2e/` | End-to-end tests (auth, TLS, SOCKS, UDP, proxy) |
| `test/bench/` | Benchmarking tool |
| `example/` | Example configs (users.json, Fly.io deployment) |

See [ARCHITECTURE.md](./ARCHITECTURE.md) for component responsibilities and data flow diagrams.

## Important Patterns

### Module path and import aliases

The Go module path is `github.com/jpillora/chisel` (upstream path, kept for compatibility). Standard import aliases used throughout:

```go
chclient "github.com/jpillora/chisel/client"
chserver "github.com/jpillora/chisel/server"
chshare  "github.com/jpillora/chisel/share"
```

### Backwards compatibility layer

`share/compat.go` re-exports types and functions from sub-packages under the `chshare` package. This exists for backwards compatibility with code that imports `chshare` directly. When adding new public APIs to `share/` sub-packages, do NOT add new aliases to `compat.go` unless maintaining an existing external API contract.

### Build version injection

`share.BuildVersion` is set at compile time via `-ldflags`. Default is `"0.0.0-src"`. The protocol version (`"chisel-v3"`) is in `share/version.go` -- changing it breaks client/server compatibility.

### Environment variables

All chisel-specific env vars use the `CHISEL_` prefix, accessed via `settings.Env("SUFFIX")` (which reads `CHISEL_SUFFIX`). Helper functions `EnvInt`, `EnvDuration`, `EnvBool` are in `share/settings/env.go`.

### Test structure

- Unit tests live alongside source files (e.g., `client/client_test.go`)
- E2e tests are in `test/e2e/` using `package e2e_test`
- E2e tests share a `testLayout` setup helper defined in `test/e2e/setup_test.go`
- `make test` excludes the `test/` directory; use `go test ./...` to include e2e

### CGO and cross-compilation

Most build targets use `CGO_ENABLED=0` except Linux and Windows which use `CGO_ENABLED=1`. All builds use `-trimpath` for reproducibility.

### Context and shutdown

Both client and server use `cos.InterruptContext()` for graceful shutdown on SIGINT/SIGTERM. Always propagate `context.Context` through new code paths.

### go.mod replace directive

`go.mod` contains `replace github.com/jpillora/chisel => ../chisel`. This is a local development override -- do not remove it, but be aware it means `go mod tidy` expects a sibling directory.

## Domain Glossary

| Term | Meaning |
|---|---|
| **Remote** | A tunnel endpoint specification in the format `local:remote` (e.g., `3000:google.com:80`). Parsed by `settings.DecodeRemote`. |
| **Forward tunnel** | Client listens locally, traffic flows through server to the target. |
| **Reverse tunnel** | Server listens, traffic flows back through client. Remotes prefixed with `R:`. |
| **Fingerprint** | SHA256 hash of the server's ECDSA public key, base64-encoded (44 chars). Used for host-key verification. |
| **SOCKS remote** | A special remote using `socks` as the target, routing through the built-in SOCKS5 proxy. |
| **stdio remote** | A special remote using `stdio` as the local host, connecting stdin/stdout to the tunnel (useful with SSH ProxyCommand). |
| **Config handshake** | JSON payload (`settings.Config`) sent from client to server over SSH after connection, containing version and requested remotes. |
| **UserIndex** | Server-side auth structure loaded from `users.json`. Maps `user:pass` to regex-based address ACLs. Hot-reloads via `fsnotify`. |
| **Keepalive** | SSH-level ping sent every 25s (default) to prevent proxy idle timeouts. |
| **Proxy (tunnel)** | A `tunnel.Proxy` instance that listens on a local address and pipes traffic through an SSH channel. |
