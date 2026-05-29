package print

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liranbh7/huh/src/pid"
	"github.com/liranbh7/huh/src/port"
)

// captureStdout runs f and returns everything written to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = orig

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestPID_Output(t *testing.T) {
	r := &pid.Result{
		PID:        42,
		Process:    "myproc",
		User:       "alice",
		State:      "sleeping",
		Command:    "/usr/bin/myproc --flag",
		Exe:        "/usr/bin/myproc",
		CWD:        "/home/alice",
		MemoryRSS:  2048,
		FDCount:    10,
		StartedAgo: 90 * time.Second,
	}

	out := captureStdout(t, func() { PID(r) })

	checks := []string{
		"PID 42",
		"myproc",
		"alice",
		"sleeping",
		"/usr/bin/myproc --flag",
		"2.0 MB",
		"10",
		"ago",
	}
	for _, s := range checks {
		if !strings.Contains(out, s) {
			t.Errorf("PID output missing %q\nfull output:\n%s", s, out)
		}
	}
}

func TestPID_SkipsEmptyFields(t *testing.T) {
	r := &pid.Result{
		PID:       7,
		Process:   "tiny",
		MemoryRSS: 0,  // should be omitted
		FDCount:   -1, // should be omitted
	}

	out := captureStdout(t, func() { PID(r) })

	if strings.Contains(out, "Memory") {
		t.Errorf("expected Memory row to be absent when MemoryRSS=0, got:\n%s", out)
	}
	if strings.Contains(out, "FDs") {
		t.Errorf("expected FDs row to be absent when FDCount=-1, got:\n%s", out)
	}
}

func TestPort_Output(t *testing.T) {
	r := &port.Result{
		Port:       8080,
		PID:        99,
		Process:    "server",
		User:       "bob",
		Command:    "/usr/bin/server",
		CWD:        "/srv",
		StartedAgo: 2*time.Hour + 5*time.Minute,
	}

	out := captureStdout(t, func() { Port(r) })

	checks := []string{
		"PORT 8080",
		"server (pid 99)",
		"bob",
		"/usr/bin/server",
		"/srv",
		"ago",
	}
	for _, s := range checks {
		if !strings.Contains(out, s) {
			t.Errorf("Port output missing %q\nfull output:\n%s", s, out)
		}
	}
}

func TestFmtMemory(t *testing.T) {
	cases := []struct {
		kb   int64
		want string
	}{
		{0, ""},
		{-1, ""},
		{512, "512 KB"},
		{1024, "1.0 MB"},
		{2048, "2.0 MB"},
		{1024 * 1024, "1.0 GB"},
	}
	for _, c := range cases {
		if got := fmtMemory(c.kb); got != c.want {
			t.Errorf("fmtMemory(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}

func TestFmtFDs(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{-1, ""},
		{0, "0"},
		{42, "42"},
	}
	for _, c := range cases {
		if got := fmtFDs(c.n); got != c.want {
			t.Errorf("fmtFDs(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFmtStarted(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, ""},
		{-time.Second, ""},
		{30 * time.Second, "30s ago"},
		{90 * time.Second, "1m 30s ago"},
	}
	for _, c := range cases {
		if got := fmtStarted(c.d); got != c.want {
			t.Errorf("fmtStarted(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
