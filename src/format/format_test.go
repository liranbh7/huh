package format

import (
	"testing"
	"time"
)

func TestMemory(t *testing.T) {
	tests := []struct {
		kb   int64
		want string
	}{
		{512, "512 KB"},
		{1024, "1.0 MB"},
		{2048, "2.0 MB"},
		{1536, "1.5 MB"},
		{1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 2, "2.0 GB"},
	}
	for _, tt := range tests {
		got := Memory(tt.kb)
		if got != tt.want {
			t.Errorf("Memory(%d) = %q, want %q", tt.kb, got, tt.want)
		}
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{90 * time.Minute, "1h 30m"},
		{50 * time.Hour, "2d 2h"},
	}
	for _, tt := range tests {
		got := Duration(tt.d)
		if got != tt.want {
			t.Errorf("Duration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
