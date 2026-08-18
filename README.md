# succulent-cli

[![CI](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml/badge.svg)](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml)

CLI for the succulent ZTP lab cluster management service.

## Quick Start

```bash
succulent-cli list
succulent-cli get info --env myenv
succulent-cli reprovision --env myenv --ocp-tag 4.17 --owner user --email u@example.com --confirm
succulent-cli watch --env myenv
succulent-cli sno kubeconfig --env myenv
```

## Commands

| Command | Description |
|---------|-------------|
| `list` | List environments with status |
| `status` | Show environment status in a readable format |
| `get info` | Cluster node info from the infoplan page |
| `get log` | Stream the Ansible playbook log |
| `watch` | Wait until cluster nodes are up |
| `health` | Check connectivity to the succulent server |
| `reprovision` | Submit an MNO reprovision request |
| `sno provision` | Submit an SNO provision request |
| `sno kubeconfig` | Download the SNO kubeconfig |
| `ztp provision` | Provision a ZTP hub and spoke |
| `ztp kubeconfig` | Download the ZTP management or spoke kubeconfig |
| `hypershift provision` | Provision a Hypershift hosted cluster |
| `hypershift kubeconfig` | Download the Hypershift management or hosted kubeconfig |
| `kubeconfig fetch` | SCP the kubeconfig from the installer node |
| `delete` | Delete an environment (requires `--confirm`) |
| `config` | Config and cache (`show`, `set`, `edit`, `init`, `path`, `cache`) |
| `completion install` | Install shell completion (bash, zsh, fish) |
| `version` | Print the CLI version |

## Guides

| Guide | Description |
|-------|-------------|
| [Installation](docs/installation.md) | From source, `go install`, or release binaries |
| [Configuration](docs/configuration.md) | Config file, environment variables, shell completion |
| [Command Reference](docs/commands.md) | Flags and defaults for every command |

## Prerequisites

- Go 1.26+ (to build from source)
- VPN access to the succulent service network

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
