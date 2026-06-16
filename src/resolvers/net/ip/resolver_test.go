package ip

import (
	"net"
	"os"
	"testing"
)

func TestResolve_invalid(t *testing.T) {
	_, err := Resolve("not-an-ip")
	if err == nil {
		t.Fatal("expected error for invalid input, got nil")
	}
}

func TestResolve_loopbackIPv4(t *testing.T) {
	r, err := Resolve("127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Version != 4 {
		t.Errorf("version: got %d, want 4", r.Version)
	}
	if r.Kind != "loopback" {
		t.Errorf("kind: got %q, want %q", r.Kind, "loopback")
	}
	if r.Summary == "" {
		t.Error("Summary must not be empty")
	}
}

func TestResolve_loopbackIPv6(t *testing.T) {
	r, err := Resolve("::1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Version != 6 {
		t.Errorf("version: got %d, want 6", r.Version)
	}
	if r.Kind != "loopback" {
		t.Errorf("kind: got %q, want %q", r.Kind, "loopback")
	}
}

func TestResolve_private(t *testing.T) {
	cases := []string{"10.0.0.1", "172.16.0.1", "192.168.1.1"}
	for _, input := range cases {
		r, err := Resolve(input)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", input, err)
			continue
		}
		if r.Kind != "private" {
			t.Errorf("%s: kind: got %q, want %q", input, r.Kind, "private")
		}
	}
}

func TestResolve_public(t *testing.T) {
	r, err := Resolve("8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Version != 4 {
		t.Errorf("version: got %d, want 4", r.Version)
	}
	if r.Kind != "public" {
		t.Errorf("kind: got %q, want %q", r.Kind, "public")
	}
}

func TestResolve_linkLocal(t *testing.T) {
	r, err := Resolve("169.254.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != "link-local" {
		t.Errorf("kind: got %q, want %q", r.Kind, "link-local")
	}
}

func TestHexToIP(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"0101A8C0", "192.168.1.1"}, // common private IP, little-endian
		{"0100007F", "127.0.0.1"},   // loopback
		{"00000000", "0.0.0.0"},     // unspecified / default route destination
		{"0101A8C0", "192.168.1.1"}, // repeated to confirm idempotence
	}
	for _, tc := range cases {
		got, err := hexToIP(tc.input)
		if err != nil {
			t.Errorf("hexToIP(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got.String() != tc.want {
			t.Errorf("hexToIP(%q) = %q, want %q", tc.input, got.String(), tc.want)
		}
	}
}

func TestHexToIPInvalid(t *testing.T) {
	_, err := hexToIP("not-hex")
	if err == nil {
		t.Error("hexToIP(invalid) expected error, got nil")
	}
}

func TestFindRoutesIPv6ReturnsNil(t *testing.T) {
	// /proc/net/route only covers IPv4; IPv6 must return nil gracefully.
	ip := net.ParseIP("::1")
	routes := findRoutes(ip)
	if routes != nil {
		t.Errorf("findRoutes(::1) = %v, want nil", routes)
	}
}

func TestFindRoutesPublicIP(t *testing.T) {
	if _, err := os.Stat("/proc/net/route"); err != nil {
		t.Skip("/proc/net/route not available")
	}
	routes := findRoutes(net.ParseIP("8.8.8.8"))
	// Any connected system has at least a default route that covers a public IP.
	if len(routes) == 0 {
		t.Skip("no routes found — isolated environment")
	}
	for _, r := range routes {
		if r.Interface == "" {
			t.Errorf("route has empty Interface: %+v", r)
		}
		if r.Network == "" {
			t.Errorf("route has empty Network: %+v", r)
		}
	}
}

func TestResolve_localIPNoRoutes(t *testing.T) {
	// Local IPs use Interface+Listeners; Routes must be empty.
	r, err := Resolve("127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Routes) != 0 {
		t.Errorf("127.0.0.1 Routes = %v, want empty (local IP uses Interface instead)", r.Routes)
	}
}

func TestResolve_publicIPRoutes(t *testing.T) {
	if _, err := os.Stat("/proc/net/route"); err != nil {
		t.Skip("/proc/net/route not available")
	}
	r, err := Resolve("8.8.8.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Routes field should be populated (at minimum the default route) on a connected host.
	if len(r.Routes) == 0 {
		t.Skip("no routes found — isolated environment")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"0.0.0.0", "unspecified"},
		{"127.0.0.1", "loopback"},
		{"::1", "loopback"},
		{"169.254.1.1", "link-local"},
		{"fe80::1", "link-local"},
		{"224.0.0.1", "multicast"},
		{"10.1.2.3", "private"},
		{"172.31.0.1", "private"},
		{"192.168.0.1", "private"},
		{"fc00::1", "private"},
		{"8.8.8.8", "public"},
	}
	for _, tc := range cases {
		ip := net.ParseIP(tc.input)
		if ip == nil {
			t.Fatalf("could not parse %q as IP", tc.input)
		}
		got := classify(ip)
		if got != tc.want {
			t.Errorf("classify(%q): got %q, want %q", tc.input, got, tc.want)
		}
	}
}
