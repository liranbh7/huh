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

func TestWellKnownService(t *testing.T) {
	cases := []struct {
		port int
		want string
	}{
		{22, "ssh"},
		{80, "http"},
		{443, "https"},
		{53, "dns"},
		{21, "ftp"},
		{25, "smtp"},
		{3306, "mysql"},
		{5432, "postgresql"},
		{6379, "redis"},
		{9999, ""},  // unknown port returns empty string
		{0, ""},     // zero not in map
		{65535, ""}, // max port not in map
	}
	for _, tc := range cases {
		got := wellKnownService(tc.port)
		if got != tc.want {
			t.Errorf("wellKnownService(%d) = %q, want %q", tc.port, got, tc.want)
		}
	}
}

// TestParseNetTCPUDPFormat verifies that parseNetTCP handles UDP /proc/net/udp
// data, which uses the same column layout as /proc/net/tcp.
func TestParseNetTCPUDPFormat(t *testing.T) {
	sampleNetUDP := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000   101        0 55555 2 0000000000000000 0
   1: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 77777 2 0000000000000000 0`
	tests := []struct {
		hexPort   string
		wantInode uint64
		wantErr   bool
	}{
		{"0035", 55555, false}, // port 53 (dns)
		{"14E9", 77777, false}, // port 5353
		{"0050", 0, true},      // port 80, not present
	}
	for _, tt := range tests {
		inode, err := parseNetTCP(strings.NewReader(sampleNetUDP), tt.hexPort)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseNetTCP(%q) error = %v, wantErr %v", tt.hexPort, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && inode != tt.wantInode {
			t.Errorf("parseNetTCP(%q) inode = %d, want %d", tt.hexPort, inode, tt.wantInode)
		}
	}
}
