# Configuration

```bash
succulent-cli config init     # Create ~/.config/succulent-cli/config.yaml
succulent-cli config show     # Show resolved configuration
succulent-cli config path     # Print config file path
succulent-cli config set url https://succulent.example.com
succulent-cli config edit     # Open the file in $EDITOR
succulent-cli config cache status
succulent-cli config cache show
succulent-cli config cache clear
```

## Precedence

CLI flags > environment variables > config file > defaults.

Flag names use hyphens; config keys and env vars use underscores: `--verify-ssl` → `verify_ssl` → `SUCCULENT_VERIFY_SSL`.

## Environment Variables

Viper reads `SUCCULENT_` plus the uppercase config key. These are the variables the CLI actually uses:

| Variable | Config key | Flag | Default | Description |
|----------|------------|------|---------|-------------|
| `SUCCULENT_URL` | `url` | `--url` | `https://succulent.eng.redhat.com` | Succulent base URL |
| `SUCCULENT_ENV` | `env` | `--env` | — | Environment name |
| `SUCCULENT_VERIFY_SSL` | `verify_ssl` | `--verify-ssl` | `false` | Enable TLS certificate verification |
| `SUCCULENT_CA_CERT` | `ca_cert` | `--ca-cert` | — | Path to a CA certificate bundle (PEM) |
| `SUCCULENT_STRICT_SSH` | `strict_ssh` | `--strict-ssh` | `false` | Enable SSH host key checking on `kubeconfig fetch` |
| `SUCCULENT_REMOTE_USER` | `remote_user` | `--user` | `root` | SSH user for `kubeconfig fetch` |
| `SUCCULENT_REMOTE_PATH` | `remote_path` | `--path` | `/root/ocp/auth/kubeconfig` | Remote kubeconfig path |
| `SUCCULENT_REMOTE_PASSWORD` | — | `--password` | — | SSH password for `kubeconfig fetch` (requires `sshpass`) |
| `SUCCULENT_DEFAULT_EMAIL` | `default_email` | — | — | Fallback for `--email` |
| `SUCCULENT_DEFAULT_OWNER` | `default_owner` | — | — | Fallback for `--owner` |
| `SUCCULENT_VERBOSE` | `verbose` | `--verbose`, `-v` | `false` | Debug logging to stderr |
| `SUCCULENT_QUIET` | `quiet` | `--quiet` | `false` | Log errors only |

`--output` and `--timeout` are flags only; they have no environment variables. `--verbose` and `--quiet` are flags/env only; they are not stored by `config set`. They cannot be set together.

`config set` / `config show` / `config init` cover `url`, `env`, `verify_ssl`, `strict_ssh`, `remote_user`, `remote_path`, `default_email`, and `default_owner`. Prefer `SUCCULENT_REMOTE_PASSWORD` or `--password` over storing a password in the config file.

## Example Config

`~/.config/succulent-cli/config.yaml`:

```yaml
url: "https://succulent.example.com"
env: "myenv"
verify_ssl: false
strict_ssh: false
remote_user: "root"
remote_path: "/root/ocp/auth/kubeconfig"
default_email: "user@example.com"
default_owner: "myuser"
```

## Shell Completion

```bash
succulent-cli completion install          # Auto-detect bash, zsh, or fish
succulent-cli completion install bash
succulent-cli completion install --dry-run
```

This writes a script under `~/.config/succulent-cli/completions/` and adds a source line to `~/.bashrc`, `~/.zshrc`, or `~/.config/fish/config.fish`.
