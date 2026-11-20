package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite docs/book/src/reference/cli.md from the flag sets")

const (
	cliDoc    = "../../docs/book/src/reference/cli.md"
	configDoc = "../../docs/book/src/configuration.md"
)

func flagTable(fset *flag.FlagSet) string {
	var b strings.Builder
	b.WriteString("| Flag | Default | Does |\n| --- | --- | --- |\n")
	fset.VisitAll(func(f *flag.Flag) {
		def := "none"
		if f.DefValue != "" && f.DefValue != "false" {
			def = "`" + f.DefValue + "`"
		}
		fmt.Fprintf(&b, "| `--%s` | %s | %s |\n", f.Name, def, strings.ReplaceAll(f.Usage, "|", "\\|"))
	})
	return b.String()
}

func renderCLIDoc() string {
	main := flag.NewFlagSet("penpot-mcp", flag.ContinueOnError)
	mainFlags(main)
	skills := flag.NewFlagSet("skills install", flag.ContinueOnError)
	skillsFlags(skills)

	var b strings.Builder
	b.WriteString("<!-- Generated from the flag sets in cmd/penpot-mcp/main.go by `go test ./cmd/penpot-mcp -run TestCLIDoc -update`. Do not edit. -->\n\n")
	b.WriteString("# Command line\n\n")
	b.WriteString("`penpot-mcp` has two forms. Everything else is set with environment variables, see [configuration](../configuration.md).\n\n")
	b.WriteString("## `penpot-mcp`\n\n")
	b.WriteString("Starts the server. With no flag it speaks MCP over stdio, which is what `claude mcp add` runs.\n\n")
	b.WriteString("```sh\npenpot-mcp [--http HOST:PORT] [--version]\n```\n\n")
	b.WriteString(flagTable(main))
	b.WriteString("\n## `penpot-mcp skills install`\n\n")
	b.WriteString("Writes the agent skills embedded in the binary into the agent's skills directory. It needs no token. It refuses a skill directory it did not write unless `--force` is given.\n\n")
	b.WriteString("```sh\npenpot-mcp skills install [--agent claude|codex] [--dir PATH] [--force]\n```\n\n")
	b.WriteString(flagTable(skills))
	b.WriteString("\nWith `--agent claude` the directory is `$CLAUDE_CONFIG_DIR/skills`, or `~/.claude/skills` when that is not set. With `--agent codex` it is `~/.agents/skills`.\n")
	return b.String()
}

// TestCLIDoc keeps reference/cli.md in step with the flag sets.
func TestCLIDoc(t *testing.T) {
	got := renderCLIDoc()
	if *update {
		if err := os.WriteFile(cliDoc, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(cliDoc)
	if err != nil {
		t.Fatalf("%v; run go test ./cmd/penpot-mcp -run TestCLIDoc -update", err)
	}
	if string(want) != got {
		t.Errorf("%s is out of date; run go test ./cmd/penpot-mcp -run TestCLIDoc -update", cliDoc)
	}
}

var envName = regexp.MustCompile(`"(PENPOT_[A-Z0-9_]+)"`)

// TestConfigurationNamesEveryEnvVar fails when a PENPOT_ variable the code
// reads is missing from the configuration page.
func TestConfigurationNamesEveryEnvVar(t *testing.T) {
	page, err := os.ReadFile(configDoc)
	if err != nil {
		t.Fatal(err)
	}
	read := map[string]bool{}
	for _, root := range []string{".", "../../internal"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path) // #nosec G304 -- walks this repository's own sources
			if err != nil {
				return err
			}
			for _, m := range envName.FindAllStringSubmatch(string(src), -1) {
				read[m[1]] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(read) == 0 {
		t.Fatal("found no PENPOT_ variable in the code; the scan is broken")
	}
	var names []string
	for n := range read {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if !strings.Contains(string(page), "`"+n+"`") {
			t.Errorf("%s reads %s but %s does not name it", "the code", n, configDoc)
		}
	}
}
