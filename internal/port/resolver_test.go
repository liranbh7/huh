package port

import (
	"strings"
	"testing"
)

var sampleNetTCP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 67890 1 0000000000000000 100 0 0 10 0
   2: 0F02000A:C4B2 0202000A:0050 01 00000000:00000000 00:00000000 00000000  1000        0 99999 1 0000000000000000 20 4 24 10 -1`

func TestParseNetTCP(t *testing.T) {
	tests := []struct {
		name      string
		hexPort   string
		wantInode uint64
		wantErr   bool
	}{
		{"port 8080 (0x1F90)", "1F90", 12345, false},
		{"port 22 (0x0016)", "0016", 67890, false},
		{"lowercase hex", "1f90", 12345, false},
		{"port not present", "0050", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inode, err := parseNetTCP(strings.NewReader(sampleNetTCP), tt.hexPort)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseNetTCP(%q) error = %v, wantErr %v", tt.hexPort, err, tt.wantErr)
			}
			if !tt.wantErr && inode != tt.wantInode {
				t.Errorf("inode = %d, want %d", inode, tt.wantInode)
			}
		})
	}
}

func TestResolveInvalidInput(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"0"},
		{"65536"},
		{"-1"},
		{"abc"},
		{""},
		{"80a"},
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

func TestResolvePortNotListening(t *testing.T) {
	// Port 1 is almost certainly not listening; expect an error.
	_, err := Resolve("1")
	if err == nil {
		t.Skip("port 1 happened to be in use; skipping")
	}
}
