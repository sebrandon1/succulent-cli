# succulent-cli

[![CI](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml/badge.svg)](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml)

A Go CLI for interacting with the succulent ZTP lab cluster management service.

## Quick Start

```bash
succulent-cli list                                     # List all environments
succulent-cli get info --env myenv                     # Cluster node info
succulent-cli reprovision --env myenv --ocp-tag 4.17 --owner user --email u@example.com --confirm
succulent-cli watch --env myenv                        # Wait for nodes to come up
succulent-cli sno kubeconfig --env myenv               # Download kubeconfig
```

## Commands

| Command | Description |
|---------|-------------|
| `list` | List all available environments with status details |
| `get info` | Parse cluster VM info from the infoplan page |
| `get log` | Stream raw Ansible playbook log to stdout |
| `watch` | Monitor provisioning until all nodes are up |
| `reprovision` | Submit an MNO cluster reprovisioning request |
| `sno provision` | Submit an SNO cluster provisioning request |
| `sno kubeconfig` | Download the SNO kubeconfig file |
| `ztp provision` | Provision a ZTP hub and spoke cluster |
| `ztp kubeconfig` | Download the ZTP management or spoke kubeconfig |
| `hypershift provision` | Provision a Hypershift hosted cluster |
| `hypershift kubeconfig` | Download the Hypershift management or hosted kubeconfig |
| `kubeconfig fetch` | SCP the kubeconfig from the installer node |
| `delete` | Delete an environment (requires `--confirm`) |
| `config show` | Show the resolved configuration |
| `config init` | Create a default config file |
| `config path` | Print the config file path |

## Guides

| Guide | Description |
|-------|-------------|
| [Installation](docs/installation.md) | From source, go install, or prebuilt binaries |
| [Configuration](docs/configuration.md) | Config file, environment variables, shell completion |
| [Command Reference](docs/commands.md) | All flags and defaults for every command |

## Prerequisites

- VPN connection to the succulent service network
- Valid credentials for authenticated endpoints

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
