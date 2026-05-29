// Package print renders resolver results to stdout.
package print

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/liranbh7/huh/src/binary"
	"github.com/liranbh7/huh/src/device"
	"github.com/liranbh7/huh/src/format"
	"github.com/liranbh7/huh/src/pid"
	"github.com/liranbh7/huh/src/port"
	"github.com/liranbh7/huh/src/processname"
)

// PID prints a pid.Result to stdout.
func PID(r *pid.Result) {
	rows := []format.Row{
		{Label: "Process", Value: r.Process},
		{Label: "User", Value: r.User},
		{Label: "State", Value: r.State},
		{Label: "Command", Value: r.Command},
		{Label: "Exe", Value: r.Exe},
		{Label: "CWD", Value: r.CWD},
		{Label: "Memory", Value: fmtMemory(r.MemoryRSS)},
		{Label: "FDs", Value: fmtFDs(r.FDCount)},
		{Label: "Started", Value: fmtStarted(r.StartedAgo)},
	}
	format.Print(fmt.Sprintf("PID %d", r.PID), rows)
}

// Port prints a port.Result to stdout.
func Port(r *port.Result) {
	rows := []format.Row{
		{Label: "Process", Value: fmt.Sprintf("%s (pid %d)", r.Process, r.PID)},
		{Label: "User", Value: r.User},
		{Label: "Command", Value: r.Command},
		{Label: "CWD", Value: r.CWD},
		{Label: "Started", Value: fmtStarted(r.StartedAgo)},
	}
	format.Print(fmt.Sprintf("PORT %d", r.Port), rows)
}

// Device prints a device.Result to stdout.
func Device(r *device.Result) {
	title := fmt.Sprintf("%s %s", deviceTitlePrefix(r.FileType), r.Path)

	typeLabel := r.FileType
	if r.Model != "" {
		typeLabel = fmt.Sprintf("%s (%s)", r.FileType, r.Model)
	}

	rows := []format.Row{
		{Label: "Type", Value: typeLabel},
	}
	if r.BlockSize != "" {
		rows = append(rows, format.Row{Label: "Size", Value: r.BlockSize})
	} else if r.Size > 0 {
		rows = append(rows, format.Row{Label: "Size", Value: format.Bytes(r.Size)})
	}
	rows = append(rows,
		format.Row{Label: "Mode", Value: r.Mode},
		format.Row{Label: "Owner", Value: fmtOwner(r.Owner, r.Group)},
		format.Row{Label: "Modified", Value: fmtModified(r.Modified)},
		format.Row{Label: "Target", Value: r.Symlink},
		format.Row{Label: "Mounts", Value: r.Mounts},
		format.Row{Label: "Mount", Value: r.MountPoint},
		format.Row{Label: "Device", Value: r.Device},
		format.Row{Label: "Filesystem", Value: r.Filesystem},
		format.Row{Label: "SMART", Value: r.Smart},
	)
	format.Print(title, rows)
}

// Binary prints a binary.Result to stdout.
func Binary(r *binary.Result) {
	rows := []format.Row{
		{Label: "Path", Value: r.Path},
		{Label: "Description", Value: r.Description},
		{Label: "Size", Value: format.Bytes(r.Size)},
		{Label: "Mode", Value: r.Mode},
		{Label: "Owner", Value: fmtOwner(r.Owner, r.Group)},
		{Label: "Libs", Value: fmtLibs(r.LinkedLibs)},
	}
	format.Print(fmt.Sprintf("BINARY %s", r.Name), rows)
}

// ProcessName prints a processname.Result to stdout.
func ProcessName(r *processname.Result) {
	var pids []string
	for _, inst := range r.Instances {
		pids = append(pids, strconv.Itoa(inst.PID))
	}

	var ports []string
	for _, p := range r.Ports {
		ports = append(ports, fmt.Sprintf(":%d", p))
	}

	title := "PROCESS " + r.Name
	if len(r.Instances) > 1 {
		title = fmt.Sprintf("PROCESS %s (%d instances)", r.Name, len(r.Instances))
	}

	rows := []format.Row{
		{Label: "Exe", Value: r.Exe},
		{Label: "Service", Value: r.Service},
		{Label: "PIDs", Value: strings.Join(pids, ", ")},
		{Label: "Ports", Value: strings.Join(ports, ", ")},
		{Label: "Logs", Value: r.LogsCmd},
	}
	format.Print(title, rows)
}

func fmtLibs(libs []string) string {
	if libs == nil {
		return ""
	}
	if len(libs) == 0 {
		return "static"
	}
	return strings.Join(libs, ", ")
}

func deviceTitlePrefix(fileType string) string {
	switch fileType {
	case "file":
		return "FILE"
	case "directory":
		return "DIR"
	case "block device", "char device":
		return "DEVICE"
	case "symlink":
		return "SYMLINK"
	default:
		return "PATH"
	}
}

func fmtOwner(owner, group string) string {
	if owner == "" {
		return ""
	}
	if group == "" {
		return owner
	}
	return owner + ":" + group
}

func fmtModified(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func fmtStarted(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return format.Duration(d) + " ago"
}

func fmtMemory(kb int64) string {
	if kb <= 0 {
		return ""
	}
	return format.Memory(kb)
}

func fmtFDs(n int) string {
	if n < 0 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}
