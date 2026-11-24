# Installation

## From Source

Requires Go 1.27+ (the version in `go.mod`).

```bash
git clone https://github.com/sebrandon1/succulent-cli.git
cd succulent-cli
make build
sudo mv succulent-cli /usr/local/bin/
```

`make install` copies the binary to `$GOPATH/bin` (or `~/go/bin`) instead.

## Go Install

```bash
go install github.com/sebrandon1/succulent-cli@latest
```

`@latest` is the newest GitHub release tag. For unreleased `main`, build from source.

## Prebuilt Binaries

Download from the [Releases](https://github.com/sebrandon1/succulent-cli/releases) page.

## Container Image

The repository includes a `Dockerfile` that builds a static binary on a
`scratch` base image:

```bash
docker build -t succulent-cli .
docker run --rm succulent-cli list
```

The container runs as the `root` user, so mount a config directory if you rely
on `~/.config/succulent-cli`:

```bash
docker run --rm -v ~/.config/succulent-cli:/root/.config/succulent-cli succulent-cli list
```
