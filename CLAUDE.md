# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

A Go CLI wrapper around the succulent ZTP lab cluster management service
(`https://succulent.eng.redhat.com`). Provides both a reusable library (`lib/`)
and a CLI (`cmd/`) built with Cobra.

## Common Commands

    make build          # Build the succulent-cli binary
    make test           # Run all unit tests
    make lint           # Run golangci-lint
    make vet            # Run go vet

## Architecture

### Two-layer design: lib -> cmd

Every API domain follows the same pattern:

1. `lib/<domain>.go` -- API client methods on `*Client`.
2. `lib/structs.go` -- All request/response types.
3. `cmd/<domain>.go` -- Cobra commands that parse flags, create a client, call lib, print output.

### Key patterns

- `lib.NewClient(baseURL, insecureSkipVerify)` creates the HTTP client.
- No authentication token -- the service is VPN-only.
- SSL verification is disabled by default (self-signed certs).
- HTML parsing uses `golang.org/x/net/html` for the infoplan endpoint.
- The kubeconfig fetch command shells out to `ssh-keygen` and `scp`.
- CLI commands use persistent flags (`--url`, `--env`, `--verify-ssl`) on the root command.

### Adding a new endpoint

1. Add request/response structs to `lib/structs.go`
2. Add client method(s) to `lib/<domain>.go`
3. Add unit tests to `lib/<domain>_test.go` using `httptest.NewServer`
4. Add Cobra command in `cmd/<domain>.go` and register it in `cmd/root.go`

## Requirements

- Go 1.26+
- golangci-lint (for `make lint`)
- VPN connection to Red Hat internal network
