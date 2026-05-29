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

// Classify returns all Types that could match input. Numeric inputs may match
// both PID and Port when the number is an active process and within port range.
func Classify(input string) []Type {
	if n, err := strconv.Atoi(input); err == nil {
		var types []Type
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", n)); err == nil {
			types = append(types, PID)
		}
		if n >= 1 && n <= 65535 {
			types = append(types, Port)
		}
		if len(types) > 0 {
			return types
		}
	}

	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") {
		return []Type{Path}
	}

	if _, err := exec.LookPath(input); err == nil {
		return []Type{Binary}
	}

	if isRunningProcess(input) {
		return []Type{ProcessName}
	}

	return []Type{Unknown}
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
