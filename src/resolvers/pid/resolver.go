// Package pid resolves a Linux PID to human-readable process information by
// reading /proc/<pid>/ files: comm, cmdline, status, cwd, exe, stat, and fd/.
package pid

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/liranbh7/huh/src/internal/procfs"
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
	CPUPercent float64
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
	r.Process = procfs.ReadComm(pid)

	status := procfs.ReadStatus(pid)
	r.User = procfs.ParseUser(status)
	r.State = procfs.ParseState(status)
	r.MemoryRSS = procfs.ParseVmRSS(status)

	r.Command = procfs.ReadCmdline(pid)
	r.Exe = procfs.ReadExe(pid)
	r.CWD = procfs.ReadCWD(pid)
	r.FDCount = countFDs(pid)

	statFields, _ := procfs.ParseStatFields(pid)
	uptimeSec, _ := procfs.ReadUptime()
	r.CPUPercent = procfs.StatCPUPercent(statFields, uptimeSec)
	r.StartedAgo = procfs.StatStartedAgo(statFields, uptimeSec)

	r.Summary = fmt.Sprintf("pid %d — %s", pid, r.Process)
	return r, nil
}

func countFDs(pid int) int {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return -1
	}
	return len(entries)
}

// readState, readStartedAgo, readCPUPercent are kept for test compatibility.

func readState(pid int) string {
	return procfs.ParseState(procfs.ReadStatus(pid))
}

func readStartedAgo(pid int) time.Duration {
	fields, _ := procfs.ParseStatFields(pid)
	uptimeSec, _ := procfs.ReadUptime()
	return procfs.StatStartedAgo(fields, uptimeSec)
}

func readCPUPercent(pid int) float64 {
	fields, _ := procfs.ParseStatFields(pid)
	uptimeSec, _ := procfs.ReadUptime()
	return procfs.StatCPUPercent(fields, uptimeSec)
}
