# Command Reference

## Global Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--url` | `SUCCULENT_URL` | See `lib.DefaultSucculentURL` | Succulent base URL |
| `--env` | `SUCCULENT_ENV` | — | Environment name (required) |
| `--verify-ssl` | — | `false` | Enable SSL certificate verification |
| `--ca-cert` | — | — | Path to custom CA certificate |
| `--output`, `-o` | — | `table` | Output format (`json` or `table`) |

## MNO Reprovision

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | — | Email address (required) |
| `--owner` | — | Username (required) |
| `--ocp-tag` | — | OCP version tag, e.g. 4.17 (required) |
| `--release-type` | `nightly` | Release type (nightly, ci) |
| `--openshift-image` | — | Full OpenShift image URL |
| `--disk-size` | `50` | Disk size in GB |
| `--virtual-workers` | `true` | Enable virtual workers |
| `--additional-workers` | — | Comma-separated extra baremetal worker names |
| `--end-date` | — | End date for the environment |
| `--kcli-params` | — | Additional kcli parameters (key:value format) |
| `--confirm` | `false` | Confirm reprovisioning (required) |

## List

| Flag | Default | Description |
|------|---------|-------------|
| `--no-detail` | `false` | Skip fetching per-environment info (fast mode) |
| `--no-cache` | `false` | Bypass the info cache |
| `--concurrency` | `10` | Number of parallel info fetches |
| `--sort` | `name` | Sort by field: name, status, group, nodes-up |
| `--filter` | — | Filter environments: key=value (e.g., status=active) |

## Watch

| Flag | Default | Description |
|------|---------|-------------|
| `--max-wait` | `60` | Maximum minutes to wait |
| `--poll-interval` | `30` | Seconds between status checks |
| `--control-plane-only` | `false` | Report ready when installer and masters are up |

## SNO Provision

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | — | Username (required) |
| `--email` | — | Email address (required) |
| `--ocp-tag` | — | OCP tag (e.g. 4.17) |
| `--release-type` | `nightly` | Release type (nightly, ci) |
| `--full-ocp-tag` | — | Full OCP tag (e.g. 4.14.0-0.nightly-2023-12-14-072431) |
| `--full-image` | — | Full image name for installation |
| `--confirm` | `false` | Confirm provisioning (required) |

## ZTP Provision

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
| `--confirm` | `false` | Confirm provisioning (required) |

## ZTP / Hypershift Kubeconfig

| Flag | Default | Description |
|------|---------|-------------|
| `--choice` | — | Kubeconfig type (required): `management`, `spoke`, or `hosted` |
| `--dest` | `~/Downloads/{env}-{type}-kubeconfig` | Local destination path |

## Hypershift Provision

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
| `--confirm` | `false` | Confirm provisioning (required) |

## Kubeconfig Fetch

| Flag | Default | Description |
|------|---------|-------------|
| `--user` | `root` | Remote SSH user |
| `--path` | `/root/ocp/auth/kubeconfig` | Remote kubeconfig path |
| `--dest` | `~/Downloads/{env}-kubeconfig` | Local destination path |
| `--wait` | `false` | Wait for all nodes to be up first |
| `--max-wait` | `60` | Maximum wait time in minutes |
| `--poll-interval` | `30` | Seconds between status checks |

## Delete

| Flag | Default | Description |
|------|---------|-------------|
| `--confirm` | `false` | Confirm deletion (required) |
