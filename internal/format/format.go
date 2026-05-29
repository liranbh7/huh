// Package format provides a consistent terminal output renderer for huh resolvers.
package format

import (
	"fmt"
	"strings"
	"time"
)

// Row is a single label/value pair in the output table.
type Row struct {
	Label string
	Value string
}

// Print writes a title line followed by aligned label : value rows.
// Rows with an empty Value are skipped.
func Print(title string, rows []Row) {
	fmt.Println(title)

	width := 0
	for _, r := range rows {
		if r.Value != "" && len(r.Label) > width {
			width = len(r.Label)
		}
	}

	for _, r := range rows {
		if r.Value == "" {
			continue
		}
		pad := strings.Repeat(" ", width-len(r.Label))
		fmt.Printf("  %s%s : %s\n", r.Label, pad, r.Value)
	}
}

// Duration formats d as "Xd Xh", "Xh Xm", "Xm Xs", or "Xs", omitting
// leading zero units.
func Duration(d time.Duration) string {
	d = d.Round(time.Second)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

// Bytes returns a human-readable string for a byte count.
func Bytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// Memory returns a human-readable string for a kilobyte value.
func Memory(kb int64) string {
	switch {
	case kb >= 1024*1024:
		return fmt.Sprintf("%.1f GB", float64(kb)/1024/1024)
	case kb >= 1024:
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	default:
		return fmt.Sprintf("%d KB", kb)
	}
}
