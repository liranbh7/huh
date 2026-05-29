// Package device resolves a filesystem path (file, directory, block device, or
// symlink) to human-readable metadata. Uses os.Lstat for base info, findmnt
// for mount/filesystem details, lsblk for block-device specifics, and smartctl
// for SMART health status.
package device

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Result holds information about a filesystem path.
type Result struct {
	Summary  string
	Path     string
	FileType string // "file", "directory", "block device", "char device", "symlink", …
	Size     int64  // bytes; non-zero only for regular files
	Mode     string // permission bits, e.g. "rw-r--r--"
	Owner    string
	Group    string
	Modified time.Time
	Symlink  string // link target when FileType == "symlink"
	// Regular file / directory fields (from findmnt)
	Device     string
	MountPoint string
	Filesystem string
	// Block device fields (from lsblk / smartctl)
	BlockSize string // human-readable size string, e.g. "500G"
	Model     string
	Mounts    string // "sda1 → /boot, sda2 → /"
	Smart     string
}

// Resolve inspects input (an absolute or relative path) and returns filesystem
// metadata along with supplemental information from external tools.
func Resolve(input string) (*Result, error) {
	fi, err := os.Lstat(input)
	if err != nil {
		return nil, fmt.Errorf("device: %w", err)
	}

	r := &Result{Path: input}
	r.FileType = fileTypeName(fi.Mode())
	r.Mode = fi.Mode().Perm().String()
	r.Modified = fi.ModTime()

	if fi.Mode().IsRegular() {
		r.Size = fi.Size()
	}

	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		r.Owner = lookupIDFile("/etc/passwd", fmt.Sprintf("%d", stat.Uid))
		r.Group = lookupIDFile("/etc/group", fmt.Sprintf("%d", stat.Gid))
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		r.Symlink, _ = os.Readlink(input)
	}

	isBlock := fi.Mode()&os.ModeDevice != 0 && fi.Mode()&os.ModeCharDevice == 0
	if isBlock {
		r.BlockSize, r.Model, r.Mounts = runLsblk(input)
		r.Smart = runSmartctl(input)
	} else {
		r.MountPoint, r.Filesystem, r.Device = runFindmnt(input)
	}

	r.Summary = fmt.Sprintf("%s — %s", input, r.FileType)
	return r, nil
}

func fileTypeName(m os.FileMode) string {
	switch {
	case m.IsDir():
		return "directory"
	case m.IsRegular():
		return "file"
	case m&os.ModeSymlink != 0:
		return "symlink"
	case m&os.ModeDevice != 0 && m&os.ModeCharDevice == 0:
		return "block device"
	case m&os.ModeCharDevice != 0:
		return "char device"
	case m&os.ModeNamedPipe != 0:
		return "named pipe"
	case m&os.ModeSocket != 0:
		return "socket"
	default:
		return "unknown"
	}
}

// runLsblk returns the human-readable size, model name, and a mount summary
// ("part → mnt, …") for a block device path.
func runLsblk(path string) (size, model, mounts string) {
	out, err := exec.Command("lsblk", "--pairs", "--noheadings",
		"--output", "NAME,SIZE,FSTYPE,MOUNTPOINT,MODEL", path).Output()
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return
	}
	first := parseLsblkLine(lines[0])
	size = first["SIZE"]
	model = strings.TrimSpace(first["MODEL"])

	var parts []string
	for _, line := range lines[1:] {
		row := parseLsblkLine(line)
		if row["MOUNTPOINT"] != "" {
			parts = append(parts, row["NAME"]+" → "+row["MOUNTPOINT"])
		}
	}
	mounts = strings.Join(parts, ", ")
	return
}

// parseLsblkLine parses one line of lsblk --pairs output, e.g.
// NAME="sda" SIZE="500G" FSTYPE="" MOUNTPOINT="" MODEL="Samsung SSD 870"
func parseLsblkLine(line string) map[string]string {
	m := make(map[string]string)
	s := line
	for {
		eq := strings.Index(s, `="`)
		if eq < 0 {
			break
		}
		keyStart := strings.LastIndexByte(s[:eq], ' ') + 1
		key := s[keyStart:eq]
		s = s[eq+2:]
		end := strings.IndexByte(s, '"')
		if end < 0 {
			break
		}
		m[key] = s[:end]
		s = s[end+1:]
	}
	return m
}

// runFindmnt returns the mount target, filesystem type, and source device for
// the filesystem that contains path.
func runFindmnt(path string) (mountPoint, fstype, device string) {
	out, err := exec.Command("findmnt", "--noheadings", "--raw",
		"--output", "TARGET,SOURCE,FSTYPE", "--target", path).Output()
	if err != nil {
		return
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) >= 1 {
		mountPoint = fields[0]
	}
	if len(fields) >= 2 {
		device = fields[1]
	}
	if len(fields) >= 3 {
		fstype = fields[2]
	}
	return
}

// runSmartctl returns the SMART overall-health status for a block device.
// Returns "" when smartctl is unavailable or does not report health.
func runSmartctl(path string) string {
	out, err := exec.Command("smartctl", "-H", path).Output()
	if err != nil && len(out) == 0 {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "SMART overall-health") {
			if i := strings.IndexByte(line, ':'); i >= 0 {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}
	return ""
}

// lookupIDFile looks up a numeric ID in an /etc/passwd- or /etc/group-style
// file and returns the name in field 0. Falls back to returning id unchanged.
func lookupIDFile(path, id string) string {
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
