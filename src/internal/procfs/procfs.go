// Package procfs provides shared helpers for reading Linux /proc and /etc files.
// All functions return zero/empty values on any read error rather than propagating
// errors, matching the silent-failure convention used across the resolver packages.
package procfs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const clkTck = 100 // USER_HZ, standard on x86 Linux

// LookupIDFile finds a numeric ID in field 2 of a colon-separated file (e.g.
// /etc/passwd or /etc/group) and returns the name in field 0. Falls back to
// returning id unchanged when the file cannot be read or the ID is absent.
func LookupIDFile(path, id string) string {
	f, err := os.Open(path)
	if err != nil {
		return id
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 4)
		if len(parts) >= 3 && parts[2] == id {
			return parts[0]
		}
	}
	return id
}

// ReadComm returns the process name from /proc/<pid>/comm.
func ReadComm(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// ReadStatus reads /proc/<pid>/status once and returns a map of field name to
// raw trimmed value (e.g. "Uid" → "1000 1000 1000 1000"). Use ParseUser,
// ParseState, and ParseVmRSS to extract specific fields without re-reading the
// file.
func ReadStatus(pid int) map[string]string {
	m := make(map[string]string)
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(data), "\n") {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		m[line[:colon]] = strings.TrimSpace(line[colon+1:])
	}
	return m
}

// ParseUser extracts the username from a status map returned by ReadStatus.
func ParseUser(status map[string]string) string {
	fields := strings.Fields(status["Uid"])
	if len(fields) < 1 {
		return ""
	}
	return LookupIDFile("/etc/passwd", fields[0])
}

// ParseState extracts the human-readable state name (e.g. "sleeping") from a
// status map returned by ReadStatus.
func ParseState(status map[string]string) string {
	fields := strings.Fields(status["State"])
	if len(fields) < 2 {
		return ""
	}
	return strings.Trim(strings.Join(fields[1:], " "), "()")
}

// ParseVmRSS extracts the resident set size in kilobytes from a status map.
func ParseVmRSS(status map[string]string) int64 {
	fields := strings.Fields(status["VmRSS"])
	if len(fields) < 1 {
		return 0
	}
	kb, _ := strconv.ParseInt(fields[0], 10, 64)
	return kb
}

// User is a single-call convenience wrapper that reads /proc/<pid>/status and
// returns the username. When you also need State or VmRSS, call ReadStatus once
// and use the Parse* functions instead.
func User(pid int) string {
	return ParseUser(ReadStatus(pid))
}

// ReadCmdline returns the command line from /proc/<pid>/cmdline, with NUL bytes
// replaced by spaces.
func ReadCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
}

// ReadExe returns the executable path from /proc/<pid>/exe.
func ReadExe(pid int) string {
	link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return link
}

// ReadCWD returns the working directory from /proc/<pid>/cwd, replacing the
// home directory prefix with "~".
func ReadCWD(pid int) string {
	link, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(link, home) {
		return "~" + link[len(home):]
	}
	return link
}

// ParseStatFields reads /proc/<pid>/stat and returns the whitespace-separated
// fields after the process name "(comm)". Indices: [0]=state, [11]=utime,
// [12]=stime, [19]=starttime (ticks since boot).
func ParseStatFields(pid int) ([]string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return nil, err
	}
	end := strings.LastIndex(string(data), ")")
	if end < 0 {
		return nil, fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	return strings.Fields(string(data)[end+1:]), nil
}

// ReadUptime returns the system uptime in seconds from /proc/uptime.
func ReadUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("malformed /proc/uptime")
	}
	return strconv.ParseFloat(fields[0], 64)
}

// StatStartedAgo computes how long ago a process started from ParseStatFields
// output and a ReadUptime value.
func StatStartedAgo(fields []string, uptimeSec float64) time.Duration {
	if len(fields) < 20 {
		return 0
	}
	ticks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0
	}
	startedAgo := uptimeSec - float64(ticks)/clkTck
	if startedAgo < 0 {
		return 0
	}
	return time.Duration(startedAgo * float64(time.Second))
}

// StatCPUPercent computes the lifetime average CPU percentage from
// ParseStatFields output and a ReadUptime value.
func StatCPUPercent(fields []string, uptimeSec float64) float64 {
	if len(fields) < 20 {
		return 0
	}
	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0
	}
	starttime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0
	}
	elapsed := uptimeSec - float64(starttime)/clkTck
	if elapsed <= 0 {
		return 0
	}
	return float64(utime+stime) / clkTck / elapsed * 100
}

// InodeToPID walks /proc/*/fd to find the PID that owns the given socket inode.
func InodeToPID(inode uint64) (int, error) {
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
