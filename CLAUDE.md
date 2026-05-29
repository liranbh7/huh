# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```
make build   # compile to ./huh
make test    # go test ./...
make lint    # run linter
```

## Key Constraints

- **No third-party runtime dependencies** — use only the Go standard library and system interfaces (`/proc`, `lsblk`, `ss`, `ldd`, etc.). Runtime deps are forbidden; dev/test-only deps are fine if they stay out of the binary.
- Results must appear in under 500ms on normal hardware.
- Zero flags — input type is auto-detected by the classifier; no subcommands.

## Architecture

Input → classifier → one resolver → output formatter

Resolver modules (each in its own package):
- **port** — numeric 1–65535, reads `/proc/net/tcp`, resolves owning process
- **process** — PID (matches `/proc/<n>`) or process name (walks `/proc/*/comm`, queries systemd)
- **device** — file/device path (`stat`, `lsblk`, `findmnt`, `smartctl`)
- **binary** — name found in `$PATH` (`which`, `ldd`, man page summary)

## Commit Style

Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`
