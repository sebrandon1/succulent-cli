# succulent-cli

[![CI](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml/badge.svg)](https://github.com/sebrandon1/succulent-cli/actions/workflows/pre-main.yaml)

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

# List all available environments
succulent-cli list

# Watch provisioning until all nodes are up
succulent-cli watch --env myenv

# Provision a ZTP hub and spoke cluster
succulent-cli ztp provision --env myenv --owner username --email user@example.com --sno-tag 4.17 --spoke-tag 4.17

# Provision a Hypershift hosted cluster
succulent-cli hypershift provision --env myenv --owner username --email user@example.com --sno-tag 4.17 --hcp-tag 4.17

# Delete an environment
succulent-cli delete --env myenv --confirm
```

## Commands

| Command | Description |
|---------|-------------|
| `list` | List all available environments with status details |
| `get info` | Parse cluster VM info from the infoplan page and output as JSON or table |
| `get log` | Stream raw Ansible playbook log to stdout |
| `watch` | Monitor provisioning until all nodes are up, then print installer IP |
| `reprovision` | Submit an MNO cluster reprovisioning request |
| `sno provision` | Submit an SNO cluster provisioning request |
| `sno kubeconfig` | Download the SNO kubeconfig file |
| `ztp provision` | Provision a ZTP hub and spoke cluster |
| `ztp kubeconfig` | Download the ZTP management or spoke kubeconfig |
| `hypershift provision` | Provision a Hypershift hosted cluster |
| `hypershift kubeconfig` | Download the Hypershift management or hosted kubeconfig |
| `kubeconfig fetch` | SCP the kubeconfig from the installer node |
| `delete` | Delete an environment (requires `--confirm`) |
| `config init` | Create a default config file |
| `config show` | Show the resolved configuration |
| `config path` | Print the config file path |

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

## List Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output`, `-o` | `table` | Output format (`json` or `table`) |
| `--no-detail` | `false` | Skip fetching per-environment info (fast mode) |
| `--no-cache` | `false` | Bypass the info cache |
| `--concurrency` | `10` | Number of parallel info fetches |

## Watch Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--max-wait` | `60` | Maximum minutes to wait |
| `--poll-interval` | `30` | Seconds between status checks |

## SNO Provision Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username (required) |
| `--email` | — | Email address (required) |
| `--ocp-tag` | — | OCP tag (e.g. 4.17) |
| `--release-type` | `nightly` | Release type (nightly, ci) |
| `--full-ocp-tag` | — | Full OCP tag (e.g. 4.14.0-0.nightly-2023-12-14-072431) |
| `--full-image` | — | Full image name for installation |

## ZTP Provision Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username (required) |
| `--email` | — | Email address (required) |
| `--sno-tag` | — | Hub (SNO) cluster OCP tag (e.g. 4.17) |
| `--sno-release` | `nightly` | Hub cluster release type |
| `--sno-full-tag` | — | Hub cluster full OCP tag |
| `--spoke-tag` | — | Spoke cluster OCP tag (e.g. 4.17) |
| `--spoke-release` | `nightly` | Spoke cluster release type |
| `--spoke-full-tag` | — | Spoke cluster full OCP tag |
| `--type` | `sno` | ZTP type: `sno` or `mno` |
| `--stop-before-deployment` | `false` | Stop before spoke deployment for manual GitOps changes |
| `--vm-masters` | `3` | Number of VM masters (MNO only) |
| `--bm-masters` | — | Comma-separated baremetal master hosts (MNO only) |
| `--bm-workers` | — | Comma-separated baremetal worker hosts |
| `--vm-workers` | `1` | Number of VM workers |

## ZTP / Hypershift Kubeconfig Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--choice` | — | Kubeconfig type (required): `management`, `spoke`, or `hosted` |
| `--dest` | `~/Downloads/{env}-{type}-kubeconfig` | Local destination path |

## Hypershift Provision Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username (required) |
| `--email` | — | Email address (required) |
| `--sno-tag` | — | Management cluster OCP tag (e.g. 4.17) |
| `--sno-release` | `nightly` | Management cluster release type |
| `--sno-full-tag` | — | Management cluster full OCP tag |
| `--hcp-tag` | — | Hosted cluster OCP tag (e.g. 4.17) |
| `--hcp-release` | `nightly` | Hosted cluster release type |
| `--hcp-full-tag` | — | Hosted cluster full OCP tag |
| `--vm-workers` | `0` | Number of VM workers for hosted cluster |
| `--image-override` | — | Hypershift operator image override |

## Kubeconfig Fetch Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--user` | `root` | Remote SSH user |
| `--path` | `/root/ocp/auth/kubeconfig` | Remote kubeconfig path |
| `--dest` | `~/Downloads/{env}-kubeconfig` | Local destination path |
| `--wait` | `false` | Wait for all nodes to be up first |
| `--max-wait` | `60` | Maximum wait time in minutes |
| `--poll-interval` | `30` | Seconds between status checks |

## Configuration

Create a config file to avoid passing `--env` and `--url` on every command:

```bash
succulent-cli config init    # Creates ~/.config/succulent-cli/config.yaml
succulent-cli config show    # Show resolved configuration
succulent-cli config path    # Print config file path
```

**Example config file** (`~/.config/succulent-cli/config.yaml`):

```yaml
url: "https://succulent.example.com"
env: "myenv"
verify_ssl: false
remote_user: "root"
remote_path: "/root/ocp/auth/kubeconfig"
default_email: "user@example.com"
default_owner: "myuser"
```

**Precedence:** CLI flags > environment variables (`SUCCULENT_` prefix) > config file > defaults.

## Shell Completion

```bash
# Bash (add to ~/.bashrc)
source <(succulent-cli completion bash)

# Zsh (add to ~/.zshrc)
source <(succulent-cli completion zsh)

# Fish
succulent-cli completion fish | source
```

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
