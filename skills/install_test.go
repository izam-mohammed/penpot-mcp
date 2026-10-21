package skills

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestInstallWritesEverySkill(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder
	if err := Install(FS, dir, "v1", false, &out); err != nil {
		t.Fatal(err)
	}
	for _, name := range skillNames(t) {
		if _, err := os.Stat(filepath.Join(dir, name, "SKILL.md")); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, name, Marker)); string(got) != "v1\n" {
			t.Errorf("%s marker = %q", name, got)
		}
		if !strings.Contains(out.String(), filepath.Join(dir, name)) {
			t.Errorf("output does not name %s:\n%s", name, out.String())
		}
	}
	// A second install replaces its own, dropping files no longer shipped.
	stale := filepath.Join(dir, "penpot-mcp", "old.md")
	if err := os.WriteFile(stale, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(FS, dir, "v2", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("a file from the earlier install lingered")
	}
}

func TestInstallRefusesForeignDir(t *testing.T) {
	dir := t.TempDir()
	mine := filepath.Join(dir, "penpot-mcp", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(mine), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mine, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Install(FS, dir, "v1", false, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "not written by penpot-mcp") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if got, _ := os.ReadFile(mine); string(got) != "mine" {
		t.Error("the user's skill was overwritten")
	}
	// The other skills still land.
	if _, err := os.Stat(filepath.Join(dir, "penpot-mcp-draw", "SKILL.md")); err != nil {
		t.Error(err)
	}
	if err := Install(FS, dir, "v1", true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(mine); string(got) == "mine" {
		t.Error("--force did not replace it")
	}
}

func TestInstallRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	checkout := t.TempDir()
	link := filepath.Join(dir, "penpot-mcp")
	if err := os.Symlink(checkout, link); err != nil {
		t.Fatal(err)
	}
	err := Install(FS, dir, "v1", false, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("err = %v, want a symlink refusal", err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced")
	}
	if entries, _ := os.ReadDir(checkout); len(entries) != 0 {
		t.Error("the install wrote through the symlink")
	}
}

func TestInstallNamesAreSafe(t *testing.T) {
	for _, name := range []string{"..", "other", "penpot-mcp-../x", "penpot-mcp-Draw"} {
		fsys := fstest.MapFS{name + "/SKILL.md": {Data: []byte("x")}}
		if err := Install(fsys, t.TempDir(), "v1", true, io.Discard); err == nil {
			t.Errorf("%q: installed, want refused", name)
		}
	}
}
