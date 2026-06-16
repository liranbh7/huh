// Package port resolves a TCP port number (1–65535) to the process that owns it.
// It reads /proc/net/tcp and /proc/net/tcp6 to locate the socket inode, then
// walks /proc/*/fd to find the owning PID.
package port

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Result holds information about the process listening on a TCP or UDP port.
type Result struct {
	Summary     string
	Port        int
	Protocol    string // "tcp" or "udp"
	ServiceName string // well-known name, e.g. "http", "ssh"
	PID         int
	Process     string
	User        string
	Command     string
	CWD         string
	StartedAgo  time.Duration
}

// Resolve looks up the process owning the given TCP or UDP port.
func Resolve(input string) (*Result, error) {
	port, err := strconv.Atoi(input)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("port: invalid input %q", input)
	}

	inode, proto, err := findSocketInode(port)
	if err != nil {
		return nil, err
	}

	pid, err := inodeToPID(inode)
	if err != nil {
		return nil, fmt.Errorf("port %d: %w", port, err)
	}

	r := &Result{Port: port, PID: pid, Protocol: proto}
	r.ServiceName = wellKnownService(port)
	r.Process = readComm(pid)
	r.User = readUser(pid)
	r.Command = strings.TrimSpace(readCmdline(pid))
	r.CWD = readCWD(pid)
	r.StartedAgo = readStartedAgo(pid)
	r.Summary = fmt.Sprintf("port %d/%s — %s (pid %d)", port, proto, r.Process, pid)
	return r, nil
}

// findSocketInode searches /proc/net/tcp[6] then /proc/net/udp[6] for a socket
// bound to port and returns its inode and protocol ("tcp" or "udp").
func findSocketInode(port int) (uint64, string, error) {
	hexPort := fmt.Sprintf("%04X", port)
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		inode, err := parseNetTCP(f, hexPort)
		f.Close()
		if err == nil {
			return inode, "tcp", nil
		}
	}
	for _, path := range []string{"/proc/net/udp", "/proc/net/udp6"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		inode, err := parseNetTCP(f, hexPort)
		f.Close()
		if err == nil {
			return inode, "udp", nil
		}
	}
	return 0, "", fmt.Errorf("port %d: not listening", port)
}

// wellKnownService returns the conventional name for common port numbers.
func wellKnownService(port int) string {
	services := map[int]string{
		20:    "ftp-data",
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		465:   "smtps",
		587:   "smtp-submission",
		993:   "imaps",
		995:   "pop3s",
		3306:  "mysql",
		5432:  "postgresql",
		6379:  "redis",
		8080:  "http-alt",
		8443:  "https-alt",
		27017: "mongodb",
	}
	return services[port]
}

// parseNetTCP scans a /proc/net/tcp[6] reader for a local address whose port
// matches hexPort (e.g. "1F90") and returns the socket inode.
func parseNetTCP(r io.Reader, hexPort string) (uint64, error) {
	scanner := bufio.NewScanner(r)
	scanner.Scan() // skip header line
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		// fields[1] = "XXXXXXXX:PPPP" (local_address)
		localAddr := fields[1]
		colon := strings.LastIndex(localAddr, ":")
		if colon < 0 {
			continue
		}
		if strings.EqualFold(localAddr[colon+1:], hexPort) {
			return strconv.ParseUint(fields[9], 10, 64)
		}
	}
	return 0, fmt.Errorf("not found")
}

// inodeToPID finds the PID whose open file descriptors include the given socket inode.
func inodeToPID(inode uint64) (int, error) {
	target := fmt.Sprintf("socket:[%d]", inode)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err == nil && link == target {
				return pid, nil
			}
		}
	}
	return 0, fmt.Errorf("no process owns inode %d", inode)
}

func readComm(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

func readUser(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return uidToName(fields[1])
			}
		}
	}
	return ""
}

// uidToName resolves a numeric UID string to a username via /etc/passwd.
func uidToName(uid string) string {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return uid
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 4)
		if len(parts) >= 3 && parts[2] == uid {
			return parts[0]
		}
	}
	return uid
}

func readCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(data), "\x00", " ")
}

func readCWD(pid int) string {
	link, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(link, home) {
		return "~" + link[len(home):]
	}
	return link
}

// readStartedAgo returns how long ago the process started using /proc/<pid>/stat
// and /proc/uptime. Returns 0 on any parse error.
func readStartedAgo(pid int) time.Duration {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	// Skip past the comm field "(name)" which may contain spaces and parens.
	end := strings.LastIndex(string(data), ")")
	if end < 0 {
		return 0
	}
	fields := strings.Fields(string(data)[end+1:])
	// field 19 (0-based after ')') is starttime in clock ticks since boot.
	if len(fields) < 20 {
		return 0
	}
	ticks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0
	}

	upData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	uptimeFields := strings.Fields(string(upData))
	if len(uptimeFields) < 1 {
		return 0
	}
	uptimeSec, err := strconv.ParseFloat(uptimeFields[0], 64)
	if err != nil {
		return 0
	}

	const clkTck = 100 // USER_HZ, standard on x86 Linux
	startedAgo := uptimeSec - float64(ticks)/clkTck
	if startedAgo < 0 {
		return 0
	}
	return time.Duration(startedAgo * float64(time.Second))
}
