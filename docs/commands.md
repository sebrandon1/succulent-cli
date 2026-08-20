# Command Reference

`--env` is required except on `list`, `health`, `version`, `config`, and `completion`. `--owner` and `--email` fall back to `default_owner` / `default_email` in config. On an interactive TTY, missing `--owner`, `--email`, and `--ocp-tag` (where that flag applies) are prompted instead of a hard error. All `SUCCULENT_*` environment variables are listed in [Configuration](configuration.md).

## Global Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--url` | `SUCCULENT_URL` | `https://succulent.eng.redhat.com` | Succulent base URL |
| `--env` | `SUCCULENT_ENV` | — | Environment name |
| `--verify-ssl` | `SUCCULENT_VERIFY_SSL` | `false` | Enable TLS certificate verification |
| `--ca-cert` | `SUCCULENT_CA_CERT` | — | Path to a CA certificate bundle (PEM) |
| `--output`, `-o` | — | `table` | `json` or `table` |
| `--timeout` | — | `60` | HTTP request timeout in seconds |
| `--verbose`, `-v` | `SUCCULENT_VERBOSE` | `false` | Debug logging to stderr (method, URL, status, duration) |
| `--quiet` | `SUCCULENT_QUIET` | `false` | Log errors only. Cannot be combined with `--verbose` |

## list

```bash
succulent-cli list
succulent-cli list --filter status=active
succulent-cli list --no-detail
```

| Flag | Default | Description |
|------|---------|-------------|
| `--no-detail` | `false` | Skip per-environment info fetches |
| `--no-cache` | `false` | Bypass the info cache (60s TTL) |
| `--concurrency` | `10` | Parallel info fetches |
| `--sort` | `name` | `name`, `status`, `group`, or `nodes-up` |
| `--filter` | — | `key=value` (`name`, `status`, `group`, `owner`). Status is `active`, `partial`, or `empty`. |

## status

```bash
succulent-cli status --env myenv
```

No command-specific flags. Uses `--output json` for JSON.

## get info

```bash
succulent-cli get info --env myenv
succulent-cli get info --env myenv --no-cache
```

| Flag | Default | Description |
|------|---------|-------------|
| `--no-cache` | `false` | Bypass the info cache |

## get log

```bash
succulent-cli get log --env myenv
```

No command-specific flags. Streams Ansible log text to stdout.

## watch

```bash
succulent-cli watch --env myenv
succulent-cli watch --env myenv --control-plane-only
```

| Flag | Default | Description |
|------|---------|-------------|
| `--max-wait` | `60` | Maximum minutes to wait |
| `--poll-interval` | `30` | Seconds between status checks |
| `--control-plane-only` | `false` | Ready when installer and masters are up |

## health

```bash
succulent-cli health
succulent-cli health --verify-ssl
```

No command-specific flags. Checks TLS and HTTP reachability of `--url`.

## reprovision

```bash
succulent-cli reprovision --env myenv --email user@example.com --owner myuser --ocp-tag 4.17 --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | — | Notification email |
| `--owner` | — | Username |
| `--ocp-tag` | — | OCP version tag, e.g. `4.17` (required; prompted on a TTY if omitted) |
| `--release-type` | `nightly` | `nightly` or `ci` |
| `--openshift-image` | — | Full OpenShift image URL |
| `--disk-size` | `50` | Disk size in GB |
| `--virtual-workers` | `true` | Enable virtual workers |
| `--additional-workers` | `false` | `false` to disable, or comma-separated hostnames |
| `--end-date` | — | End date for the environment |
| `--kcli-params` | — | Extra kcli parameters (`key:value`) |
| `--confirm` | `false` | Confirm reprovisioning (required unless `--dry-run`) |
| `--dry-run` | `false` | Print the request without submitting |

## sno provision

```bash
succulent-cli sno provision --env myenv --owner myuser --email user@example.com --ocp-tag 4.17 --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username |
| `--email` | — | Notification email |
| `--ocp-tag` | — | OCP tag (e.g. `4.17`); prompted on a TTY if no version flag is set |
| `--release-type` | `nightly` | `nightly` or `ci` |
| `--full-ocp-tag` | — | Full OCP tag; overrides `--ocp-tag` and `--release-type` |
| `--full-image` | — | Full image reference; overrides tag flags |
| `--confirm` | `false` | Confirm provisioning (required unless `--dry-run`) |
| `--dry-run` | `false` | Print the request without submitting |

## sno kubeconfig

```bash
succulent-cli sno kubeconfig --env myenv
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dest` | `~/Downloads/succulent/{env}/sno-kubeconfig` | Local destination path |

## ztp provision

```bash
succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17 --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username |
| `--email` | — | Notification email |
| `--sno-tag` | — | Hub (SNO) OCP tag |
| `--sno-release` | `nightly` | Hub release type |
| `--sno-full-tag` | — | Hub full OCP tag |
| `--spoke-tag` | — | Spoke OCP tag |
| `--spoke-release` | `nightly` | Spoke release type |
| `--spoke-full-tag` | — | Spoke full OCP tag |
| `--type` | `sno` | `sno` or `mno` |
| `--stop-before-deployment` | `false` | Stop before spoke deployment |
| `--vm-masters` | `3` | VM masters (MNO only) |
| `--bm-masters` | — | Comma-separated baremetal masters (MNO only) |
| `--bm-workers` | — | Comma-separated baremetal workers |
| `--vm-workers` | `1` | Number of VM workers |
| `--confirm` | `false` | Confirm provisioning (required unless `--dry-run`) |
| `--dry-run` | `false` | Print the request without submitting |

## ztp kubeconfig

```bash
succulent-cli ztp kubeconfig --env myenv --choice management
succulent-cli ztp kubeconfig --env myenv --choice spoke
```

| Flag | Default | Description |
|------|---------|-------------|
| `--choice` | — | Required: `management` or `spoke` |
| `--dest` | `~/Downloads/succulent/{env}/ztp-{choice}-kubeconfig` | Local destination path |

## hypershift provision

```bash
succulent-cli hypershift provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --hcp-tag 4.17 --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username |
| `--email` | — | Notification email |
| `--sno-tag` | — | Management cluster OCP tag |
| `--sno-release` | `nightly` | Management release type |
| `--sno-full-tag` | — | Management full OCP tag |
| `--hcp-tag` | — | Hosted cluster OCP tag |
| `--hcp-release` | `nightly` | Hosted release type |
| `--hcp-full-tag` | — | Hosted full OCP tag |
| `--vm-workers` | `0` | VM workers for the hosted cluster |
| `--image-override` | — | Hypershift operator image override |
| `--confirm` | `false` | Confirm provisioning (required unless `--dry-run`) |
| `--dry-run` | `false` | Print the request without submitting |

## hypershift kubeconfig

```bash
succulent-cli hypershift kubeconfig --env myenv --choice management
succulent-cli hypershift kubeconfig --env myenv --choice hosted
```

| Flag | Default | Description |
|------|---------|-------------|
| `--choice` | — | Required: `management` or `hosted` |
| `--dest` | `~/Downloads/succulent/{env}/hypershift-{choice}-kubeconfig` | Local destination path |

## kubeconfig fetch

```bash
succulent-cli kubeconfig fetch --env myenv
succulent-cli kubeconfig fetch --env myenv --wait --strict-ssh
```

SSH host key checking is off by default (`StrictHostKeyChecking=no`). `--strict-ssh` uses `~/.ssh/known_hosts`.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--user` | `SUCCULENT_REMOTE_USER` | `root` | Remote SSH user |
| `--password` | `SUCCULENT_REMOTE_PASSWORD` | — | SSH password (requires `sshpass`) |
| `--path` | `SUCCULENT_REMOTE_PATH` | `/root/ocp/auth/kubeconfig` | Remote kubeconfig path |
| `--dest` | — | `~/Downloads/succulent/{env}/kubeconfig` | Local destination path |
| `--wait` | — | `false` | Wait for nodes to be up first |
| `--control-plane-only` | — | `false` | With `--wait`, ready when installer and masters are up |
| `--max-wait` | — | `60` | Maximum wait in minutes |
| `--poll-interval` | — | `30` | Seconds between status checks |
| `--strict-ssh` | `SUCCULENT_STRICT_SSH` | `false` | Enable SSH host key checking |

## delete

```bash
succulent-cli delete --env myenv --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--confirm` | `false` | Confirm deletion (required unless `--dry-run`) |
| `--dry-run` | `false` | Print the action without deleting |

## config

```bash
succulent-cli config init
succulent-cli config show
succulent-cli config path
succulent-cli config set url https://succulent.example.com
succulent-cli config edit
succulent-cli config cache status
succulent-cli config cache show
succulent-cli config cache clear
```

See [Configuration](configuration.md).

## completion install

```bash
succulent-cli completion install
succulent-cli completion install bash
succulent-cli completion install --dry-run
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Print the planned changes |
| `--shell` | auto | `bash`, `zsh`, or `fish` |

## version

```bash
succulent-cli version
```

No command-specific flags.
