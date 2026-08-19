# CLAUDE.md

Guidance for working in this repository.

## Requirements

- Go 1.26+

## Build

```bash
make build            # Binary with version from git tags
make test             # All tests, verbose
make lint             # golangci-lint
make vet
make fmt              # gofmt + goimports
make install          # Build and copy to $GOPATH/bin
make coverage
make coverage-html
make clean

go test ./lib -run TestGetInfoPlan -v
go test ./cmd -v
go test ./lib -v
```

## Architecture

Cobra CLI with a two-layer split like [go-quay](https://github.com/sebrandon1/go-quay).

**`lib/`** — HTTP client. `Client` in `client.go` exposes `NewClient`, `NewClientWithTimeout`, `getRaw`, `postForm`, and `postFormRaw`. Response bodies are capped at `MaxResponseSize` (10MB) via `readLimited`, `copyLimited`, and `limitedBody`. Types live in `structs.go`.

| File | Role |
|------|------|
| `info.go` | `GetInfoPlan` — HTML table parse (`golang.org/x/net/html`) |
| `log.go` | `StreamLog` |
| `reprovision.go` | `Reprovision` |
| `sno.go` | `ProvisionSNO`, `GetSNOKubeconfig` |
| `delete.go` | `DeleteEnvironment` |
| `list.go` | `ListEnvironments`, `ListEnvironmentsWithInfo` |
| `hypershift.go` | `ProvisionHypershift`, `GetHypershiftKubeconfig` |
| `ztp.go` | `ProvisionZTP`, `GetZTPKubeconfig` |
| `cache.go` | TTL JSON file cache |
| `kubeconfig.go` | `RemoveSSHHostKey`, `FetchKubeconfig`, `ValidateKubeconfig` (`os/exec`); `WaitForClusterReady` polls `GetInfoPlan` |

**`cmd/`** — Cobra. Most commands register themselves in their own `init()` with `rootCmd.AddCommand`. `root.go` defines persistent flags (`--url`, `--env`, `--verify-ssl`, `--ca-cert`, `--output`, `--timeout`) and creates the shared `lib.Client` in `PersistentPreRunE`. `helpers.go` has `printJSON`, `printResult`, `resolveOwnerEmail`, and `saveKubeconfig`. `constants.go` holds command names and defaults.

```text
succulent-cli
├── list | watch | status | health
├── get info | get log
├── reprovision | delete
├── sno provision | sno kubeconfig
├── ztp provision | ztp kubeconfig
├── hypershift provision | hypershift kubeconfig
├── kubeconfig fetch
├── config show | path | init | set | edit | cache
├── completion install
└── version
```

**Patterns:**

- No auth token. TLS verification is off by default for lab self-signed certs; `--verify-ssl` and `--ca-cert` enable it.
- `/infoplan/{env}` returns HTML, not JSON.
- `FetchKubeconfig` is not a `Client` method; it shells out to `scp`.
- Config: flag > `SUCCULENT_*` env > `~/.config/succulent-cli/config.yaml` > defaults.
- `list` caches `ClusterInfo` as JSON (60s TTL). `--output json` switches table commands to JSON; `--filter` / `--sort` apply to `list`.
- Tests use `httptest.NewServer`.
- Version: `-ldflags "-X main.version=$(VERSION)"`.
- Construct API URLs with `c.endpointURL(...)`, not string concat or `fmt.Sprintf` on `BaseURL`.

## Adding a command

1. Request/response structs in `lib/structs.go`
2. `Client` method in `lib/<domain>.go`
3. Tests in `lib/<domain>_test.go` with `httptest.NewServer`
4. Cobra command in `cmd/<domain>.go`; register it in that file's `init()` via `rootCmd.AddCommand`
