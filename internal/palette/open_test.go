package palette

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenBinaryStaysInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.penpot"), []byte("PK\x03\x04"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := OpenBinary(root, "a.penpot", 100); err != nil || string(got) != "PK\x03\x04" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := OpenBinary(root, "a.penpot", 2); err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Errorf("a file over the limit: %v", err)
	}
	if _, err := OpenBinary(root, "missing.penpot", 100); err == nil {
		t.Error("a missing file should fail")
	}
	outside := filepath.Join(t.TempDir(), "b.penpot")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenBinary(root, outside, 100); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("a file outside the root: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir.penpot"), 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenBinary(root, "dir.penpot", 100); err == nil {
		t.Error("a directory should fail")
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenBinary(root, ".env", 100); err == nil {
		t.Error(".env must never be read")
	}
	if _, err := Open(t.TempDir()+"/nope", "a"); err == nil {
		t.Error("a missing root should fail")
	}
}
