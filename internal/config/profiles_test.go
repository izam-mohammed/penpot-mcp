package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("PENPOT_WORK_TOKEN", "x")
	dir := t.TempDir()
	write := func(s string) string {
		p := filepath.Join(dir, "p.json")
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	ps, err := Load(write(`[{"name":"work","token_env":"PENPOT_WORK_TOKEN","base_url":"https://penpot.example.com"}]`))
	if err != nil || len(ps) != 1 || ps[0].BaseURL != "https://penpot.example.com" {
		t.Fatalf("load: %+v %v", ps, err)
	}
	for _, c := range []struct{ json, want string }{
		{`[]`, "no profiles"},
		{`{}`, "JSON list"},
		{`[{"name":"Work","token_env":"PENPOT_WORK_TOKEN"}]`, "lowercase"},
		{`[{"name":"w","token_env":"PENPOT_WORK_TOKEN"},{"name":"w","token_env":"PENPOT_WORK_TOKEN"}]`, "twice"},
		{`[{"name":"w","token_env":"abc-123"}]`, "environment variable"},
		{`[{"name":"w","token_env":"PENPOT_UNSET_TOKEN"}]`, "is not set"},
		{`[{"name":"w","token_env":"PENPOT_WORK_TOKEN","password_env":"hunter2"}]`, "environment variable"},
	} {
		if _, err := Load(write(c.json)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.json, err, c.want)
		}
	}
}
