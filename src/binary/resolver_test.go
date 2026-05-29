package binary

import (
	"os/exec"
	"testing"
)

func TestResolve(t *testing.T) {
	bin := "ls"
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("%q not found in PATH, skipping", bin)
	}

	tests := []struct {
		input   string
		wantErr bool
	}{
		{bin, false},
		{"__no_such_binary__", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := Resolve(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Resolve(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if r.Name != tt.input {
				t.Errorf("Name = %q, want %q", r.Name, tt.input)
			}
			if r.Path == "" {
				t.Error("Path is empty")
			}
			if r.Size <= 0 {
				t.Errorf("Size = %d, want > 0", r.Size)
			}
			if r.Mode == "" {
				t.Error("Mode is empty")
			}
			if r.Summary == "" {
				t.Error("Summary is empty")
			}
		})
	}
}

func TestRunWhatis(t *testing.T) {
	if _, err := exec.LookPath("whatis"); err != nil {
		t.Skip("whatis not available")
	}
	// ls is universally documented; result may be empty if whatis DB not built
	_ = runWhatis("ls")
}

func TestRunLdd(t *testing.T) {
	if _, err := exec.LookPath("ldd"); err != nil {
		t.Skip("ldd not available")
	}
	path, err := exec.LookPath("ls")
	if err != nil {
		t.Skip("ls not found")
	}
	libs := runLdd(path)
	// ls is almost universally dynamically linked; just assert no panic
	_ = libs
}
