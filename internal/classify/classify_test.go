package classify

import (
	"os"
	"strings"
	"testing"
)

func TestTypeString(t *testing.T) {
	cases := []struct {
		t    Type
		want string
	}{
		{Port, "port"},
		{PID, "pid"},
		{ProcessName, "process"},
		{Path, "path"},
		{Binary, "binary"},
		{Unknown, "unknown"},
	}
	for _, c := range cases {
		if got := c.t.String(); got != c.want {
			t.Errorf("Type(%d).String() = %q, want %q", c.t, got, c.want)
		}
	}
}

func TestClassify_Path(t *testing.T) {
	cases := []string{"/etc/hosts", "./foo", "../bar"}
	for _, input := range cases {
		got := Classify(input)
		if len(got) != 1 || got[0] != Path {
			t.Errorf("Classify(%q) = %v, want [path]", input, got)
		}
	}
}

func TestClassify_Binary(t *testing.T) {
	// "sh" is guaranteed to exist on any POSIX system.
	got := Classify("sh")
	if len(got) != 1 || got[0] != Binary {
		t.Errorf("Classify(\"sh\") = %v, want [binary]", got)
	}
}

func TestClassify_Unknown(t *testing.T) {
	got := Classify("thisprobablydoesnotexist_xyzzy")
	if len(got) != 1 || got[0] != Unknown {
		t.Errorf("Classify(\"thisprobablydoesnotexist_xyzzy\") = %v, want [unknown]", got)
	}
}

func TestClassify_NumericPID(t *testing.T) {
	// Use PID 1 which always exists on Linux.
	input := "1"
	types := Classify(input)
	hasPID := false
	for _, ty := range types {
		if ty == PID {
			hasPID = true
		}
	}
	if !hasPID {
		t.Errorf("Classify(%q) = %v, expected PID to be included", input, types)
	}
}

func TestClassify_NumericPortRange(t *testing.T) {
	// 65000 is in port range (1–65535) and also in PID range, so both are returned.
	got := Classify("65000")
	hasPort, hasPID := false, false
	for _, ty := range got {
		if ty == Port {
			hasPort = true
		}
		if ty == PID {
			hasPID = true
		}
	}
	if !hasPort || !hasPID {
		t.Errorf("Classify(\"65000\") = %v, want [port pid]", got)
	}
}

func TestClassify_NumericOutOfPortRange(t *testing.T) {
	// 99999 exceeds MaxPort (65535) but is within MaxPID, so only PID is returned.
	got := Classify("99999")
	if len(got) != 1 || got[0] != PID {
		t.Errorf("Classify(\"99999\") = %v, want [pid]", got)
	}
}

func TestClassify_Zero(t *testing.T) {
	got := Classify("0")
	// 0 is not a valid port (1–65535) and /proc/0 doesn't exist.
	if len(got) != 1 || got[0] != Unknown {
		t.Errorf("Classify(\"0\") = %v, want [unknown]", got)
	}
}

func TestIsRunningProcess(t *testing.T) {
	// "init" or "systemd" runs as PID 1; read its comm to get a real name.
	comm, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		t.Skip("cannot read /proc/1/comm")
	}
	name := strings.TrimSpace(string(comm))
	if !isRunningProcess(name) {
		t.Errorf("isRunningProcess(%q) = false, want true", name)
	}
	if isRunningProcess("thisprocnamedoesnotexist_xyzzy") {
		t.Error("isRunningProcess returned true for a nonexistent process name")
	}
}
