package pid

import (
	"os"
	"strconv"
	"testing"
)

func TestResolveInvalidInput(t *testing.T) {
	tests := []struct{ input string }{
		{"0"},
		{"-1"},
		{"abc"},
		{""},
		{"1a"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := Resolve(tt.input)
			if err == nil {
				t.Errorf("Resolve(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestResolveNonExistentPID(t *testing.T) {
	// PID 2147483647 is virtually certain to not exist.
	_, err := Resolve("2147483647")
	if err == nil {
		t.Error("Resolve(very large PID) expected error, got nil")
	}
}

func TestResolveCurrentProcess(t *testing.T) {
	pid := os.Getpid()
	r, err := Resolve(strconv.Itoa(pid))
	if err != nil {
		t.Fatalf("Resolve(%d) unexpected error: %v", pid, err)
	}
	if r.PID != pid {
		t.Errorf("PID = %d, want %d", r.PID, pid)
	}
	if r.Process == "" || r.Process == "unknown" {
		t.Errorf("Process name is empty or unknown")
	}
	if r.Summary == "" {
		t.Error("Summary is empty")
	}
}

func TestStartedAgoPID1(t *testing.T) {
	// PID 1 has been running since boot — StartedAgo must be non-zero.
	if _, err := os.Stat("/proc/1"); err != nil {
		t.Skip("cannot access /proc/1")
	}
	d := readStartedAgo(1)
	if d == 0 {
		t.Error("readStartedAgo(1) returned 0; expected a positive uptime")
	}
}

func TestReadState(t *testing.T) {
	// Current process must have a valid state.
	state := readState(os.Getpid())
	if state == "" {
		t.Error("readState returned empty string for current process")
	}
}
