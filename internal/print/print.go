// Package print renders resolver results to stdout.
package print

import (
	"fmt"
	"time"

	"github.com/liranbh7/huh/internal/format"
	"github.com/liranbh7/huh/internal/pid"
	"github.com/liranbh7/huh/internal/port"
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
