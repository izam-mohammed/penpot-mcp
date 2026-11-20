package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// clearEnv blanks every variable run reads, so the host's own do not leak in.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"PENPOT_BASE_URL", "PENPOT_ACCESS_TOKEN", "PENPOT_EMAIL", "PENPOT_PASSWORD",
		"PENPOT_MCP_URL", "PENPOT_PROFILES", "PENPOT_MCP_HTTP_TOKEN", "CLAUDE_CONFIG_DIR",
	} {
		t.Setenv(k, "")
	}
}

// fakePenpot answers get-profile with the given id.
func fakePenpot(t *testing.T, id string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/get-profile") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + id + `","fullname":"Test User"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRunVersion(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-version"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "penpot-mcp ") {
		t.Fatalf("output %q", out.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	if err := run([]string{"-nope"}, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error for an unknown flag")
	}
}

func TestRunWithoutToken(t *testing.T) {
	clearEnv(t)
	err := run(nil, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "PENPOT_ACCESS_TOKEN is not set") {
		t.Fatalf("got %v", err)
	}
}

func TestRunTokenRejected(t *testing.T) {
	clearEnv(t)
	srv := fakePenpot(t, "00000000-0000-0000-0000-000000000000")
	t.Setenv("PENPOT_BASE_URL", srv.URL)
	t.Setenv("PENPOT_ACCESS_TOKEN", "test-token-not-real")
	err := run(nil, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "was not accepted") {
		t.Fatalf("got %v", err)
	}
}

func TestRunBadBaseURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("PENPOT_BASE_URL", "ftp://example.invalid")
	t.Setenv("PENPOT_ACCESS_TOKEN", "test-token-not-real")
	if err := run(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error for a non-http base url")
	}
}

func TestRunBadProfilesFile(t *testing.T) {
	clearEnv(t)
	t.Setenv("PENPOT_PROFILES", filepath.Join(t.TempDir(), "missing.json"))
	if err := run(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error for a missing profiles file")
	}
}

func TestRunProfileTokenRejected(t *testing.T) {
	clearEnv(t)
	srv := fakePenpot(t, "00000000-0000-0000-0000-000000000000")
	path := filepath.Join(t.TempDir(), "profiles.json")
	body := `[{"name":"work","base_url":"` + srv.URL + `","token_env":"TEST_WORK_TOKEN"}]`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PENPOT_PROFILES", path)
	t.Setenv("TEST_WORK_TOKEN", "test-token-not-real")
	err := run(nil, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "profile work") {
		t.Fatalf("got %v", err)
	}
}

// TestRunServesHTTP starts the server on a free port and stops it with
// SIGTERM, the way a supervisor would.
func TestRunServesHTTP(t *testing.T) {
	clearEnv(t)
	srv := fakePenpot(t, "11111111-1111-1111-1111-111111111111")
	t.Setenv("PENPOT_BASE_URL", srv.URL)
	t.Setenv("PENPOT_ACCESS_TOKEN", "test-token-not-real")
	t.Setenv("PENPOT_MCP_HTTP_TOKEN", "test-http-token")

	done := make(chan error, 1)
	go func() { done <- run([]string{"-http", "127.0.0.1:0"}, &bytes.Buffer{}) }()
	time.Sleep(500 * time.Millisecond)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not stop on SIGTERM")
	}
}

func TestRunSkillsUsage(t *testing.T) {
	if err := run([]string{"skills"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("got %v", err)
	}
}

func TestRunSkillsInstallDir(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := run([]string{"skills", "install", "--dir", dir}, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "penpot-mcp", "SKILL.md")); err != nil {
		t.Fatalf("skill not written: %v", err)
	}
}

func TestRunSkillsInstallAgents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if err := run([]string{"skills", "install"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "penpot-mcp")); err != nil {
		t.Fatal(err)
	}
	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	if err := run([]string{"skills", "install"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfg, "skills", "penpot-mcp")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"skills", "install", "--agent", "codex"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "penpot-mcp")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"skills", "install", "--agent", "vim"}, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error for an unknown agent")
	}
	if err := run([]string{"skills", "install", "--bogus"}, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error for a bad flag")
	}
}
