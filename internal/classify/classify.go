// Package classify determines what kind of input was given to huh and returns
// the appropriate resolver type so the caller can dispatch to the right package.
package classify

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Type identifies the category of input.
type Type int

const (
	Unknown     Type = iota
	Port             // numeric 1–65535, not an active PID
	PID              // numeric matching /proc/<n>
	ProcessName      // string matching a running process comm
	Path             // starts with / or ./ and exists on the filesystem
	Binary           // name found in $PATH
)

func (t Type) String() string {
	switch t {
	case Port:
		return "port"
	case PID:
		return "pid"
	case ProcessName:
		return "process"
	case Path:
		return "path"
	case Binary:
		return "binary"
	default:
		return "unknown"
	}
}

// Classify returns the most likely Type for input.
func Classify(input string) Type {
	if n, err := strconv.Atoi(input); err == nil {
		// A running process with this PID takes priority over a port with the same number.
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", n)); err == nil {
			return PID
		}
		if n >= 1 && n <= 65535 {
			return Port
		}
	}

	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") {
		return Path
	}

	if _, err := exec.LookPath(input); err == nil {
		return Binary
	}

	if isRunningProcess(input) {
		return ProcessName
	}

	return Unknown
}

// isRunningProcess reports whether any /proc/*/comm matches name exactly.
func isRunningProcess(name string) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		comm, err := os.ReadFile(fmt.Sprintf("/proc/%s/comm", entry.Name()))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) == name {
			return true
		}
	}
	return false
}
