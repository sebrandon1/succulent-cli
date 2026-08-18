# Contributing to succulent-cli

Requires Go 1.26+.

1. Clone the repository
2. `make build`
3. `make test`
4. `make lint`

## Pull Requests

- Keep changes focused
- Add tests for new functionality
- `make lint` and `make test` must pass
- API logic in `lib/`, CLI in `cmd/` (see [CLAUDE.md](CLAUDE.md))
