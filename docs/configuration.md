# Configuration

Create a config file to avoid passing `--env` and `--url` on every command:

```bash
succulent-cli config init    # Creates ~/.config/succulent-cli/config.yaml
succulent-cli config show    # Show resolved configuration
succulent-cli config path    # Print config file path
```

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

## Precedence

CLI flags > environment variables (`SUCCULENT_` prefix) > config file > defaults.

## Shell Completion

```bash
# Bash (add to ~/.bashrc)
source <(succulent-cli completion bash)

# Zsh (add to ~/.zshrc)
source <(succulent-cli completion zsh)

# Fish
succulent-cli completion fish | source
```
