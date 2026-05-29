package processname

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestResolveNotFound(t *testing.T) {
	_, err := Resolve("this-process-does-not-exist-zzz")
	if err == nil {
		t.Error("expected error for non-existent process, got nil")
	}
}

func TestResolveEmptyInput(t *testing.T) {
	_, err := Resolve("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestResolveCurrentProcess(t *testing.T) {
	pid := os.Getpid()
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		t.Skip("cannot read /proc comm")
	}
	name := strings.TrimSpace(string(comm))

	r, err := Resolve(name)
	if err != nil {
		t.Fatalf("Resolve(%q) unexpected error: %v", name, err)
	}

	found := false
	for _, inst := range r.Instances {
		if inst.PID == pid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("current PID %d not found in instances %v", pid, r.Instances)
	}
	if r.Name != name {
		t.Errorf("Name = %q, want %q", r.Name, name)
	}
	if r.Summary == "" {
		t.Error("Summary is empty")
	}
	if len(r.Instances) == 0 {
		t.Error("Instances is empty")
	}
}

func TestResolveInstanceFields(t *testing.T) {
	pid := os.Getpid()
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		t.Skip("cannot read /proc comm")
	}
	name := strings.TrimSpace(string(comm))

	r, err := Resolve(name)
	if err != nil {
		t.Fatalf("Resolve(%q) unexpected error: %v", name, err)
	}

	var inst *Instance
	for i := range r.Instances {
		if r.Instances[i].PID == pid {
			inst = &r.Instances[i]
			break
		}
	}
	if inst == nil {
		t.Fatalf("current PID %d not found in result", pid)
	}
	if inst.State == "" {
		t.Error("Instance.State is empty for current process")
	}
	if inst.Exe == "" {
		t.Error("Instance.Exe is empty for current process")
	}
}

func TestParseListening(t *testing.T) {
	// Simulate a /proc/net/tcp snippet with one LISTEN (0A) and one ESTABLISHED (01) row.
	const input = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 99999 1 0000000000000000 100 0 0 10 0
   1: 0100007F:1F91 0100007F:C350 01 00000000:00000000 00:00000000 00000000  1000        0 88888 1 0000000000000000 20 4 24 10 -1
`
	m := map[uint64]int{}
	parseListening(strings.NewReader(input), m)

	if port, ok := m[99999]; !ok || port != 0x1F90 {
		t.Errorf("expected inode 99999 → port %d, got %d (ok=%v)", 0x1F90, port, ok)
	}
	if _, ok := m[88888]; ok {
		t.Error("ESTABLISHED socket should not appear in listening map")
	}
}
