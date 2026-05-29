# huh

A Linux terminal command that tells you what any given thing actually is.

No flags, no manuals needed — just `huh <thing>` and get a human-readable summary.

## Usage

```
huh <port | pid | process name | file path | device | binary>
```

## Examples

**Port**
```
$ huh 8080

PORT 8080
  Process : node (pid 14231)
  User    : lbh
  Command : node server.js --port 8080
  CWD     : ~/projects/myapp
  Started : 2h 14m ago
```

**Process name**
```
$ huh nginx

PROCESS nginx
  Binary  : /usr/sbin/nginx (nginx/1.24.0)
  Service : nginx.service [active, running]
  PIDs    : 1023 (master), 1024 1025 (workers)
  Ports   : :80, :443
  Logs    : journalctl -u nginx
```

**Device**
```
$ huh /dev/sda

DEVICE /dev/sda
  Type    : disk (ATA Samsung SSD 870)
  Size    : 500G
  Mounts  : /dev/sda1 → /boot, /dev/sda2 → /
  SMART   : OK (last checked 3d ago)
```

**Binary**
```
$ huh rsync

BINARY rsync
  Path    : /usr/bin/rsync
  Version : rsync 3.2.7
  Linked  : libacl.so.1, libpopt.so.0, libc.so.6
  Man     : rsync(1) — a fast, versatile file-copying tool
```

## How it works

`huh` inspects the input and determines what kind of thing it is:

| Input type     | Detection method                      | Info sources                           |
|----------------|---------------------------------------|----------------------------------------|
| Port number    | Numeric, 1–65535                      | `/proc/net/tcp`, `ss`, `lsof`          |
| PID            | Numeric, matches `/proc/<n>`          | `/proc/<pid>/status`, `cmdline`, `fd`  |
| Process name   | String matching running process names | `/proc/*/comm`, `systemctl`            |
| File / device  | Path exists on filesystem             | `stat`, `lsblk`, `findmnt`, `smartctl` |
| Binary         | Found in `$PATH`                      | `which`, `ldd`, `man`, `--version`     |

## Goals

- Zero flags — input type is auto-detected
- Fast — results in under 500ms
- No external dependencies at runtime — single static binary
- Human-readable output, not raw dump

## Implementation plan

1. Input classifier — determine what kind of thing was passed
2. Port resolver — `/proc/net/tcp` + process lookup
3. Process resolver — `/proc` walking + systemd integration
4. Device resolver — `lsblk`, `findmnt`, SMART status
5. Binary resolver — `$PATH` lookup, `ldd`, man page summary
6. Output formatter — consistent, aligned output

## Tech stack

- **Language**: Go
- **Build**: single static binary via `go build`
- **Install**: drop into `/usr/local/bin`
