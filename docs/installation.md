# Installation

## From Source

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
