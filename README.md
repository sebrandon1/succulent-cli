# succulent-cli

[![CI](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml/badge.svg)](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sebrandon1/succulent-cli)](https://golang.org/)
[![License](https://img.shields.io/github/license/sebrandon1/succulent-cli)](LICENSE)

A Go CLI for interacting with the succulent ZTP lab cluster management service.

## Prerequisites

- VPN connection to the succulent service network
- Valid credentials for authenticated endpoints

## Installation

### From Source

```bash
git clone https://github.com/sebrandon1/succulent-cli.git
cd succulent-cli
make build
sudo mv succulent-cli /usr/local/bin/
```

### Go Install

```bash
go install github.com/sebrandon1/succulent-cli@latest
```

### Prebuilt Binaries

Download from the [Releases](https://github.com/sebrandon1/succulent-cli/releases) page.

## Quick Start

```bash
# Get cluster node info as JSON
succulent-cli get info --env myenv

# Stream the Ansible install log
succulent-cli get log --env myenv

# Reprovision an MNO cluster
succulent-cli reprovision --env myenv --email user@example.com --owner username --tag 4.17

# Provision an SNO cluster
succulent-cli sno provision --env myenv --owner username --email user@example.com --ocp-tag 4.17

# Download SNO kubeconfig
succulent-cli sno kubeconfig --env myenv

# Fetch kubeconfig via SCP from installer node
succulent-cli kubeconfig fetch --env myenv

# Delete an environment
succulent-cli delete --env myenv --confirm
```

## Commands

| Command | Description |
|---------|-------------|
| `get info` | Parse cluster VM info from the infoplan page and output as JSON |
| `get log` | Stream raw Ansible playbook log to stdout |
| `reprovision` | Submit an MNO cluster reprovisioning request |
| `sno provision` | Submit an SNO cluster provisioning request |
| `sno kubeconfig` | Download the SNO kubeconfig file |
| `kubeconfig fetch` | SCP the kubeconfig from the installer node |
| `delete` | Delete an environment (requires `--confirm`) |

## Global Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--url` | `SUCCULENT_URL` | See `lib.DefaultSucculentURL` | Succulent base URL |
| `--env` | `SUCCULENT_ENV` | — | Environment name (required) |
| `--verify-ssl` | — | `false` | Enable SSL certificate verification |

## MNO Reprovision Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | — | Email address (required) |
| `--owner` | — | Username (required) |
| `--tag` | — | OCP version tag, e.g. 4.17 (required) |
| `--version` | `nightly` | Release version (nightly, ci) |
| `--openshift-image` | — | Full OpenShift image URL |
| `--disk-size` | `50` | Disk size in GB |
| `--virtual-workers` | `true` | Enable virtual workers |
| `--additional-workers` | — | Comma-separated extra baremetal worker names |
| `--end-date` | — | End date for the environment |
| `--kcli-params` | — | Additional kcli parameters (key:value format) |

## SNO Provision Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username (required) |
| `--email` | — | Email address (required) |
| `--ocp-tag` | — | OCP tag (e.g. 4.17) |
| `--release-type` | `nightly` | Release type (nightly, ci) |
| `--full-ocp-tag` | — | Full OCP tag (e.g. 4.14.0-0.nightly-2023-12-14-072431) |
| `--full-image` | — | Full image name for installation |

## Kubeconfig Fetch Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--user` | `root` | Remote SSH user |
| `--path` | `/root/ocp/auth/kubeconfig` | Remote kubeconfig path |
| `--dest` | `~/Downloads/{env}-kubeconfig` | Local destination path |
| `--wait` | `false` | Wait for all nodes to be up first |
| `--max-wait` | `60` | Maximum wait time in minutes |
| `--poll-interval` | `30` | Seconds between status checks |

## Development

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make vet      # Run go vet
make clean    # Remove binary
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
