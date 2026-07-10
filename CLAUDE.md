# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Requirements

- Go 1.26+

## Build and Development Commands

```bash
make build          # Build binary with version from git tags
make test           # Run all tests with verbose output
make lint           # Run golangci-lint (requires golangci-lint installed)
make vet            # Run go vet
make fmt            # Run gofmt and goimports
make install        # Build and copy binary to $GOPATH/bin
make coverage       # Run tests with coverage report
make coverage-html  # Generate HTML coverage report
make clean          # Remove binary and coverage files

# Run a single test
go test ./lib -run TestGetInfoPlan -v

# Run tests for a specific package
go test ./cmd -v
go test ./lib -v
```

## Architecture

This is a Cobra-based CLI with a two-layer design mirroring [go-quay](https://github.com/sebrandon1/go-quay):

**`lib/`** — API client layer. `Client` struct in `client.go` provides `getRaw()` (returns `*http.Response` for streaming/HTML), `postForm()` (URL-encoded POST), and `postFormRaw()` (returns raw response body) helpers. Each domain file adds methods to `*Client`:
- `info.go` — `GetInfoPlan` (HTML table parsing via `golang.org/x/net/html`)
- `log.go` — `StreamLog` (raw Ansible log streaming)
- `reprovision.go` — `Reprovision` (MNO cluster POST)
- `sno.go` — `ProvisionSNO`, `GetSNOKubeconfig`
- `delete.go` — `DeleteEnvironment`
- `list.go` — `ListEnvironments`, `ListEnvironmentsWithInfo` (concurrent info fetches with optional caching)
- `hypershift.go` — `ProvisionHypershift`, `GetHypershiftKubeconfig`
- `ztp.go` — `ProvisionZTP`, `GetZTPKubeconfig`
- `cache.go` — `Cache` struct with TTL-based JSON file cache (`GetInfo`, `SetInfo`, `SetMultipleInfo`, `Clear`)
- `kubeconfig.go` — standalone functions (`RemoveSSHHostKey`, `FetchKubeconfig`, `ValidateKubeconfig`) that shell out via `os/exec`

All request/response types live in `structs.go`.

**`cmd/`** — Cobra command layer. Each file defines commands that parse flags, use a shared `lib.Client` (created in `PersistentPreRunE`), call lib methods, and output results. `root.go` wires the command tree and defines persistent flags (`--url`, `--env`, `--verify-ssl`, `--ca-cert`, `--output`). `helpers.go` provides `printJSON()`, `printResult()`, `resolveOwnerEmail()`, and `saveKubeconfig()`. `constants.go` centralizes command name strings and defaults.

**Command tree:**
```
succulent-cli
├── list                  # List environments with status table (lib.ListEnvironments/ListEnvironmentsWithInfo)
├── watch                 # Poll until cluster nodes are up (lib.WaitForClusterReady)
├── get log               # Streams raw Ansible log text (lib.StreamLog → io.Copy to stdout)
├── get info              # Parses HTML table → JSON (lib.GetInfoPlan, uses golang.org/x/net/html)
├── reprovision           # MNO cluster POST form (lib.Reprovision)
├── sno provision         # SNO cluster POST form (lib.ProvisionSNO)
├── sno kubeconfig        # Downloads SNO kubeconfig (lib.GetSNOKubeconfig)
├── ztp provision         # ZTP hub+spoke provisioning (lib.ProvisionZTP)
├── ztp kubeconfig        # Downloads ZTP management or spoke kubeconfig (lib.GetZTPKubeconfig)
├── hypershift provision  # Hypershift hosted cluster provisioning (lib.ProvisionHypershift)
├── hypershift kubeconfig # Downloads Hypershift management or hosted kubeconfig (lib.GetHypershiftKubeconfig)
├── kubeconfig fetch      # Composite: scrapes installer IP from infoplan, then shells out to scp
├── delete                # Deletes environment, requires --confirm (lib.DeleteEnvironment)
├── config show           # Show resolved configuration (Viper settings)
├── config init           # Create default config file at ~/.config/succulent-cli/config.yaml
├── config path           # Print the config file path
└── version               # Print CLI version
```

**Key patterns:**
- `lib.NewClient(baseURL, insecureSkipVerify, caCertPath)` — no auth token; SSL verification off by default for self-signed certs; optional CA cert
- The `/infoplan/{env}` endpoint returns HTML, not JSON — `info.go` walks the DOM tree to extract table rows
- `kubeconfig.go` has standalone functions (`RemoveSSHHostKey`, `FetchKubeconfig`) that shell out via `os/exec` — these are not `Client` methods since they don't use HTTP
- `WaitForClusterReady` is a `Client` method because it polls `GetInfoPlan` in a loop
- Tests use `httptest.NewServer` with handler funcs that validate request path/method/form values
- Version is embedded at build time via `-ldflags "-X main.version=$(VERSION)"`
- **Viper config management** — config file at `~/.config/succulent-cli/config.yaml`, env vars with `SUCCULENT_` prefix, CLI flags; all three layers merge via Viper with flag > env > file precedence
- **Caching** — `lib.Cache` stores per-environment `ClusterInfo` as a JSON file with configurable TTL (default 60s); used by `list` command for concurrent info fetches
- **Tabwriter output** — `list` command renders aligned table output; `--output json` switches to JSON; `--filter` and `--sort` control filtering/ordering
- **Owner/email resolution** — `resolveOwnerEmail()` falls back to `default_owner`/`default_email` from config

## Adding a New Endpoint

1. Add request/response structs to `lib/structs.go`
2. Add `Client` method(s) to `lib/<domain>.go`
3. Add tests to `lib/<domain>_test.go` using `httptest.NewServer`
4. Add Cobra command in `cmd/<domain>.go`, register it in `cmd/root.go` `init()`
