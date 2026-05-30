package ip

import (
	"net"
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
