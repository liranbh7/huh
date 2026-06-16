# huh

A Linux terminal command that tells you what any given thing actually is.

No flags, no manuals needed — just `huh <thing>` and get a human-readable summary.

> For when you stare at a port number, a process name, or some mystery binary and think... *huh?*
> Type it. `huh` figures out what it is, who owns it, and what it's doing — in under 500ms.

## Installation

**Requirements:** Go 1.22+

```bash
git clone https://github.com/liranbh7/huh
cd huh
make build
sudo mv bin/huh /usr/local/bin/
```

## Usage

```
huh <input>
```

`<input>` can be:

| Input        | Example           |
| ------------ | ----------------- |
| Port number  | `huh 80`          |
| PID          | `huh 1234`        |
| Process name | `huh nginx`       |
| Device path  | `huh /dev/sda1`   |
| Binary name  | `huh man`         |
| IP address   | `huh 192.168.1.1` |


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

**IP address**
```
$ huh 8.8.8.8

IP 8.8.8.8
  Kind      : public
  Version   : IPv4
  Hostname  : dns.google
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

| Input type    | Detection method                      | Info sources                             |
| ------------- | ------------------------------------- | ---------------------------------------- |
| Port number   | Numeric, 1–65535                      | `/proc/net/tcp`, `/proc/net/udp`                   |
| PID           | Numeric, matches `/proc/<n>`          | `/proc/<pid>/status`, `cmdline`, `stat`, `fd`      |
| Process name  | String matching running process names | `pgrep` (falls back to `/proc/*/comm`), `systemctl` |
| File / device | Path exists on filesystem             | `stat`, `lsblk`, `findmnt`, `smartctl`              |
| Binary        | Found in `$PATH`                      | `which`, `ldd`, `whatis`                            |
| IP address    | Parses as IPv4 or IPv6                | `net.LookupAddr`, `/proc/net`, `/proc/net/route`   |

## Goals

- Zero flags — input type is auto-detected
- Fast — results in under 500ms
- No external dependencies at runtime — single static binary
- Human-readable output, not raw dump

## Linux compatibility

`huh` works on any Linux distro that provides `/proc`. Some resolvers depend on external tools that may not be present on every system:

| Tool               | Used for                           | Availability                                                                              |
| ------------------ | ---------------------------------- | ----------------------------------------------------------------------------------------- |
| `pgrep`            | Process name → PID lookup (fast path) | Part of `procps`; present on virtually all distros; falls back to `/proc` walk if absent |
| `lsblk`, `findmnt` | Device resolver                   | Part of `util-linux`; present on most mainstream distros, may be absent on Alpine/BusyBox |
| `smartctl`         | Device health (SMART)              | From `smartmontools`; often not installed by default                                      |
| `systemctl`        | Process → service lookup           | Systemd only; absent on Alpine, Void, Gentoo/OpenRC, etc.                                 |
| `whatis`           | Binary man page summary            | From `man-db`; may be missing on minimal installs                                         |
| `ldd`              | Binary linked libraries            | Part of glibc; present on virtually all distros                                           |

Missing tools are detected at runtime via `PATH` lookup — the affected field is skipped rather than causing an error. Core functionality (port, PID, process name, binary path) works on any standard Linux system.

## Tech stack

- **Language**: Go
- **Build**: single static binary via `go build`
- **Install**: drop into `/usr/local/bin`
