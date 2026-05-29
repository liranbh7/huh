package device

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveNonExistent(t *testing.T) {
	_, err := Resolve("/nonexistent/path/xyz/abc")
	if err == nil {
		t.Error("Resolve of non-existent path: expected error, got nil")
	}
}

func TestResolveTempDir(t *testing.T) {
	dir := t.TempDir()
	r, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve(%q) unexpected error: %v", dir, err)
	}
	if r.FileType != "directory" {
		t.Errorf("FileType = %q, want %q", r.FileType, "directory")
	}
	if r.Path != dir {
		t.Errorf("Path = %q, want %q", r.Path, dir)
	}
	if r.Summary == "" {
		t.Error("Summary is empty")
	}
	if r.Mode == "" {
		t.Error("Mode is empty")
	}
}

func TestResolveTempFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "huh-test-*")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("hello")
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	r, err := Resolve(f.Name())
	if err != nil {
		t.Fatalf("Resolve(%q) unexpected error: %v", f.Name(), err)
	}
	if r.FileType != "file" {
		t.Errorf("FileType = %q, want %q", r.FileType, "file")
	}
	if r.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", r.Size, len(content))
	}
	if r.Summary == "" {
		t.Error("Summary is empty")
	}
}

func TestResolveSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	r, err := Resolve(link)
	if err != nil {
		t.Fatalf("Resolve(%q) unexpected error: %v", link, err)
	}
	if r.FileType != "symlink" {
		t.Errorf("FileType = %q, want %q", r.FileType, "symlink")
	}
	if r.Symlink != target {
		t.Errorf("Symlink = %q, want %q", r.Symlink, target)
	}
}

func TestParseLsblkLine(t *testing.T) {
	tests := []struct {
		line string
		want map[string]string
	}{
		{
			line: `NAME="sda" SIZE="500G" FSTYPE="" MOUNTPOINT="" MODEL="Samsung SSD 870"`,
			want: map[string]string{
				"NAME": "sda", "SIZE": "500G", "FSTYPE": "", "MOUNTPOINT": "", "MODEL": "Samsung SSD 870",
			},
		},
		{
			line: `NAME="sda1" SIZE="512M" FSTYPE="vfat" MOUNTPOINT="/boot/efi" MODEL=""`,
			want: map[string]string{
				"NAME": "sda1", "SIZE": "512M", "FSTYPE": "vfat", "MOUNTPOINT": "/boot/efi", "MODEL": "",
			},
		},
	}
	for _, tt := range tests {
		got := parseLsblkLine(tt.line)
		for k, wantVal := range tt.want {
			if got[k] != wantVal {
				t.Errorf("parseLsblkLine[%q] = %q, want %q", k, got[k], wantVal)
			}
		}
	}
}
