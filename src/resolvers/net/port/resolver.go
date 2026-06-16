// Package port resolves a TCP port number (1–65535) to the process that owns it.
// It reads /proc/net/tcp and /proc/net/tcp6 to locate the socket inode, then
// walks /proc/*/fd to find the owning PID.
package port

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/liranbh7/huh/src/internal/procfs"
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

	pid, err := procfs.InodeToPID(inode)
	if err != nil {
		return nil, fmt.Errorf("port %d: %w", port, err)
	}

	r := &Result{Port: port, PID: pid, Protocol: proto}
	r.ServiceName = wellKnownService(port)
	r.Process = procfs.ReadComm(pid)
	r.User = procfs.User(pid)
	r.Command = procfs.ReadCmdline(pid)
	r.CWD = procfs.ReadCWD(pid)

	statFields, _ := procfs.ParseStatFields(pid)
	uptimeSec, _ := procfs.ReadUptime()
	r.StartedAgo = procfs.StatStartedAgo(statFields, uptimeSec)

	r.Summary = fmt.Sprintf("port %d/%s — %s (pid %d)", port, proto, r.Process, pid)
	return r, nil
}

var wellKnownServices = map[int]string{
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

func wellKnownService(port int) string {
	return wellKnownServices[port]
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
