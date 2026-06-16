// Package binary resolves a name found in $PATH to human-readable metadata.
// Uses exec.LookPath to find the binary, os.Stat for file info, ldd for
// shared library dependencies, and whatis for a one-line description.
package binary

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/liranbh7/huh/src/internal/procfs"
)

// Result holds information about an executable found in $PATH.
type Result struct {
	Summary     string
	Name        string
	Path        string
	Size        int64
	Mode        string
	Owner       string
	Group       string
	Description string   // one-line summary from whatis
	LinkedLibs  []string // shared library names from ldd; nil means static
}

// Resolve looks up input in $PATH and returns executable metadata.
func Resolve(input string) (*Result, error) {
	path, err := exec.LookPath(input)
	if err != nil {
		return nil, fmt.Errorf("binary: %q not found in $PATH", input)
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("binary: %w", err)
	}

	r := &Result{
		Name: input,
		Path: path,
		Size: fi.Size(),
		Mode: fi.Mode().Perm().String(),
	}

	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		r.Owner = procfs.LookupIDFile("/etc/passwd", fmt.Sprintf("%d", stat.Uid))
		r.Group = procfs.LookupIDFile("/etc/group", fmt.Sprintf("%d", stat.Gid))
	}

	r.Description = runWhatis(input)
	r.LinkedLibs = runLdd(path)

	if r.Description != "" {
		r.Summary = fmt.Sprintf("%s — %s", path, r.Description)
	} else {
		r.Summary = path
	}
	return r, nil
}

// runWhatis returns the one-line description for name from the whatis database.
// Returns "" when whatis is unavailable or has no entry.
func runWhatis(name string) string {
	out, err := exec.Command("whatis", name).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	// First line: "ls (1) - list directory contents"
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if i := strings.Index(line, " - "); i >= 0 {
		return strings.TrimSpace(line[i+3:])
	}
	return ""
}

// runLdd returns the list of shared library names for path.
// Returns nil for static binaries or when ldd is unavailable.
func runLdd(path string) []string {
	out, err := exec.Command("ldd", path).Output()
	if err != nil && len(out) == 0 {
		return nil
	}
	var libs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "not a dynamic executable") {
			return nil
		}
		// "libname.so.X => /path/to/lib (0xaddr)"  — keep only the left side
		if i := strings.Index(line, "=>"); i >= 0 {
			name := strings.TrimSpace(line[:i])
			if name != "" {
				libs = append(libs, name)
			}
		}
	}
	return libs
}
