package main

import (
	"fmt"
	"os"
	"time"

	"github.com/liranbh7/huh/internal/classify"
	"github.com/liranbh7/huh/internal/port"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: huh <port | pid | process | path | binary>\n")
		os.Exit(1)
	}
	input := os.Args[1]

	switch classify.Classify(input) {
	case classify.Port:
		r, err := port.Resolve(input)
		if err != nil {
			fatal(err)
		}
		printPort(r)
	case classify.PID:
		fatal(fmt.Errorf("pid resolver not yet implemented"))
	case classify.ProcessName:
		fatal(fmt.Errorf("process resolver not yet implemented"))
	case classify.Path:
		fatal(fmt.Errorf("device/path resolver not yet implemented"))
	case classify.Binary:
		fatal(fmt.Errorf("binary resolver not yet implemented"))
	default:
		fatal(fmt.Errorf("huh: don't know what %q is", input))
	}
}

func printPort(r *port.Result) {
	fmt.Printf("PORT %d\n", r.Port)
	fmt.Printf("  Process : %s (pid %d)\n", r.Process, r.PID)
	if r.User != "" {
		fmt.Printf("  User    : %s\n", r.User)
	}
	if r.Command != "" {
		fmt.Printf("  Command : %s\n", r.Command)
	}
	if r.CWD != "" {
		fmt.Printf("  CWD     : %s\n", r.CWD)
	}
	if r.StartedAgo > 0 {
		fmt.Printf("  Started : %s ago\n", fmtDuration(r.StartedAgo))
	}
}

// fmtDuration formats a duration as "Xd Xh Xm Xs", omitting leading zero units.
func fmtDuration(d time.Duration) string {
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

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "huh: %s\n", err)
	os.Exit(1)
}
