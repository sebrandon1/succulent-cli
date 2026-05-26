# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
make build          # Build binary with version from git tags
make test           # Run all tests with verbose output
make lint           # Run golangci-lint (requires golangci-lint installed)
make vet            # Run go vet
make clean          # Remove built binary

# Run a single test
go test ./lib -run TestGetInfoPlan -v

# Run tests for a specific package
go test ./cmd -v
go test ./lib -v
```

## Architecture

This is a Cobra-based CLI with a two-layer design mirroring [go-quay](https://github.com/sebrandon1/go-quay):

**`lib/`** — API client layer. `Client` struct in `client.go` provides `getRaw()` (returns `*http.Response` for streaming/HTML) and `postForm()` (URL-encoded POST) helpers. Each domain file (`log.go`, `info.go`, `reprovision.go`, `sno.go`, `delete.go`) adds methods to `*Client`. All request/response types live in `structs.go`.

**`cmd/`** — Cobra command layer. Each file defines commands that parse flags, create a `lib.Client`, call lib methods, and output results. `root.go` wires the command tree and defines persistent flags (`--url`, `--env`, `--verify-ssl`). `helpers.go` provides `printJSON()` and `markFlagRequired()`.

**Command tree:**
```
succulent-cli
├── get log          # Streams raw Ansible log text (lib.StreamLog → io.Copy to stdout)
├── get info         # Parses HTML table → JSON (lib.GetInfoPlan, uses golang.org/x/net/html)
├── reprovision      # MNO cluster POST form (lib.Reprovision)
├── sno provision    # SNO cluster POST form (lib.ProvisionSNO)
├── sno kubeconfig   # Downloads SNO kubeconfig (lib.GetSNOKubeconfig)
├── delete           # Deletes environment, requires --confirm (lib.DeleteEnvironment)
└── kubeconfig fetch # Composite: scrapes installer IP from infoplan, then shells out to scp
```

**Key patterns:**
- `lib.NewClient(baseURL, insecureSkipVerify)` — no auth token; SSL verification off by default for self-signed certs
- The `/infoplan/{env}` endpoint returns HTML, not JSON — `info.go` walks the DOM tree to extract table rows
- `kubeconfig.go` has standalone functions (`RemoveSSHHostKey`, `FetchKubeconfig`) that shell out via `os/exec` — these are not `Client` methods since they don't use HTTP
- `WaitForClusterReady` is a `Client` method because it polls `GetInfoPlan` in a loop
- Tests use `httptest.NewServer` with handler funcs that validate request path/method/form values
- Version is embedded at build time via `-ldflags "-X main.version=$(VERSION)"`

## Adding a New Endpoint

1. Add request/response structs to `lib/structs.go`
2. Add `Client` method(s) to `lib/<domain>.go`
3. Add tests to `lib/<domain>_test.go` using `httptest.NewServer`
4. Add Cobra command in `cmd/<domain>.go`, register it in `cmd/root.go` `init()`
