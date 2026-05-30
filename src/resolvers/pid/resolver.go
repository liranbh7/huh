// Package pid resolves a Linux PID to human-readable process information by
// reading /proc/<pid>/ files: comm, cmdline, status, cwd, exe, stat, and fd/.
package pid

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Result holds information about a running process identified by PID.
type Result struct {
	Summary    string
	PID        int
	Process    string
	User       string
	State      string
	Command    string
	Exe        string
	CWD        string
	MemoryRSS  int64 // kilobytes
	FDCount    int
	StartedAgo time.Duration
}

// Resolve looks up a process by its PID.
func Resolve(input string) (*Result, error) {
	pid, err := strconv.Atoi(input)
	if err != nil || pid < 1 {
		return nil, fmt.Errorf("pid: invalid input %q", input)
	}
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
		return nil, fmt.Errorf("pid %d: no such process", pid)
	}

	r := &Result{PID: pid}
	r.Process = readComm(pid)
	r.User = readUser(pid)
	r.State = readState(pid)
	r.Command = readCmdline(pid)
	r.Exe = readExe(pid)
	r.CWD = readCWD(pid)
	r.MemoryRSS = readMemoryRSS(pid)
	r.FDCount = countFDs(pid)
	r.StartedAgo = readStartedAgo(pid)
	r.Summary = fmt.Sprintf("pid %d — %s", pid, r.Process)
	return r, nil
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

func readState(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "State:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// "State:\tS (sleeping)" → "sleeping"
				return strings.Trim(strings.Join(fields[2:], " "), "()")
			}
		}
	}
	return ""
}

func readCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
}

func readExe(pid int) string {
	link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return link
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

func readMemoryRSS(pid int) int64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return kb
				}
			}
		}
	}
	return 0
}

func countFDs(pid int) int {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return -1
	}
	return len(entries)
}

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
	_ = scanner.Err()
	return uid
}
