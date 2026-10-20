package palette

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func names(cs []Color) string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name+"="+c.Hex+"@"+strconv.FormatFloat(c.Opacity, 'f', -1, 64))
	}
	return strings.Join(out, " ")
}

func TestReaders(t *testing.T) {
	for _, c := range []struct {
		file, text string
		alphaFirst bool
		want       string
	}{
		{
			"Theme.swift", "/// The accent.\nstatic let accent = token(\"#5BF8F9\")\nstatic let dim: Color = tone(\"#3D5BF8F9\")", true,
			"accent=#5BF8F9@1 dim=#5BF8F9@0.2392",
		},
		{
			"Colors.kt", "val Dark = Palette(\n  accent = Color(0xFF4DD0E1),\n  veil = Color(0x80000000))", false,
			"accent=#4DD0E1@1 veil=#000000@0.502",
		},
		{
			"design.md", "| Token | Value | Use |\n|---|---|---|\n| `ink` | #101828 | text |\n| term | ANSI | no hex |\n| `sky` | #3B82F6 | links |\n", true,
			"ink=#101828@1 sky=#3B82F6@1",
		},
		{
			"theme.css", ":root { --brand-primary: #2563eb; --muted: #ccc; --overlay: #00000080; }", false,
			"brand-primary=#2563EB@1 muted=#CCCCCC@1 overlay=#000000@0.502",
		},
		{
			"tokens.json", `{"color":{"brand":{"primary":{"$value":"#2563EB","$type":"color"}},"space":{"$value":"8px","$type":"dimension"}}}`, true,
			"color / brand / primary=#2563EB@1",
		},
		{
			"sd.json", `{"brand":{"ink":{"value":"#101828"}},"flat":"#FFFFFF"}`, true,
			"brand / ink=#101828@1 flat=#FFFFFF@1",
		},
		{
			"colors.txt", "#FF0000\nAccent: #00FF00\nnotes here\nbg = #0000FF", true,
			"Color 1=#FF0000@1 Accent=#00FF00@1 bg=#0000FF@1",
		},
	} {
		got, err := Parse(c.file, []byte(c.text), c.alphaFirst)
		if err != nil {
			t.Errorf("%s: %v", c.file, err)
			continue
		}
		if names(got) != c.want {
			t.Errorf("%s:\n got  %s\n want %s", c.file, names(got), c.want)
		}
	}
	if _, err := Parse("empty.txt", []byte("nothing"), true); err == nil {
		t.Error("a file with no colours should be an error")
	}
}

func TestPaletteRefusesOutsideCwd(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	write := func(dir, name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write(root, "ok.css", "--a: #ffffff;")
	secret := write(outside, "secret.txt", "#123456")
	write(root, ".env", "PENPOT_ACCESS_TOKEN=x")
	write(root, "big.txt", strings.Repeat("#", MaxSize+1))
	write(root, "bin.txt", "\xff\xfe\x00")
	if err := os.Symlink(secret, filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(root, "ok.css"); err != nil {
		t.Errorf("a file inside root: %v", err)
	}
	for _, c := range []struct{ path, want string }{
		{secret, "outside the working directory"},
		{"../" + filepath.Base(outside) + "/secret.txt", "outside the working directory"},
		{"link.txt", "outside the working directory"},
		{".env", "not a palette file"},
		{"big.txt", "palette files are read up to"},
		{"bin.txt", "not a text file"},
		{".", "not a regular file"},
	} {
		if _, err := Open(root, c.path); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Open(%q) = %v, want an error with %q", c.path, err, c.want)
		}
	}
}
