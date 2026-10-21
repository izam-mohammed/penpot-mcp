// These tests keep the skills' Chrome flows and install steps free of
// secrets, their links and tool names true, and the installer careful.
package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestChromeDocStopsBeforeSecrets: the token flow ends before the token is
// on screen, and never reads the dialog.
func TestChromeDocStopsBeforeSecrets(t *testing.T) {
	doc := read(t, "penpot-mcp/references/chrome.md")
	checkSecretFlow(t, doc, "## Make an access token", "Do not click *Create access token*")
	checkSecretFlow(t, doc, "## Get the MCP key", "Do not click the generate button yourself")
}

// checkSecretFlow holds a flow that ends where a secret appears: it must
// stop, say not to click, and never act on the page after stopping.
func checkSecretFlow(t *testing.T, doc, heading, noClick string) {
	t.Helper()
	start := strings.Index(doc, heading)
	if start < 0 {
		t.Fatalf("no %q flow", heading)
	}
	flow := doc[start:]
	if end := strings.Index(flow[3:], "\n## "); end >= 0 {
		flow = flow[:end+3]
	}
	// Join wrapped lines so phrases match across line breaks.
	flow = strings.Join(strings.Fields(flow), " ")
	for _, must := range []string{
		"Stop here",
		noClick,
		"never screenshot, `read_page`, `find` or copy",
	} {
		if !strings.Contains(flow, must) {
			t.Errorf("%s lacks %q", heading, must)
		}
	}
	// No step after the stop may act on the page.
	after := flow[strings.Index(flow, "Stop here"):]
	for _, act := range []string{"`screenshot`", "`read_page`", "`find`", "left_click", "pbpaste"} {
		if strings.Contains(after, act) {
			t.Errorf("%s acts on the page after stopping: %s", heading, act)
		}
	}
}

// TestSkillNoSecretFiles: install steps never put the token in a file, and
// no file in any skill holds anything token-shaped.
func TestSkillNoSecretFiles(t *testing.T) {
	jwe := regexp.MustCompile(`eyJ[A-Za-z0-9_-]{20,}`)
	toFile := regexp.MustCompile(`PENPOT_ACCESS_TOKEN=\S*\s*(>>|> *[~./\w]|\.env\b)`)
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) == ".go" {
			return err
		}
		s := read(t, path)
		if jwe.MatchString(s) {
			t.Errorf("%s holds a token-shaped string", path)
		}
		if toFile.MatchString(s) {
			t.Errorf("%s writes the token to a file", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// skillNames lists the skill directories in the tree.
func skillNames(t *testing.T) []string {
	t.Helper()
	names, err := Names(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no skills")
	}
	return names
}

// TestSkillFrontmatter: every skill's frontmatter opens the file, names its
// directory, and its trigger text fits the 1,536-character listing
// (docs/external/claude-code-skills.md).
func TestSkillFrontmatter(t *testing.T) {
	for _, name := range skillNames(t) {
		s := read(t, name+"/SKILL.md")
		if !strings.HasPrefix(s, "---\n") {
			t.Errorf("%s: frontmatter must be the first line", name)
			continue
		}
		end := strings.Index(s[4:], "\n---\n")
		if end < 0 {
			t.Errorf("%s: unterminated frontmatter", name)
			continue
		}
		fm := s[4 : 4+end]
		if !strings.Contains(fm, "name: "+name+"\n") {
			t.Errorf("%s: name must be %s", name, name)
		}
		desc := regexp.MustCompile(`(?m)^(description|when_to_use): (.*)$`).FindAllStringSubmatch(fm, -1)
		n := 0
		for _, m := range desc {
			n += len(m[2])
		}
		if len(desc) != 2 || n > 1536 {
			t.Errorf("%s: description + when_to_use = %d chars in %d fields, want 2 fields within 1536", name, n, len(desc))
		}
		if lines := strings.Count(s, "\n"); lines > 500 {
			t.Errorf("%s: SKILL.md is %d lines, keep it under 500", name, lines)
		}
	}
}

// TestSkillLinksResolve: every relative link in every skill points at a
// file inside the skills tree, so the set works wherever it is installed.
func TestSkillLinksResolve(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)#\s]+)`)
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		// Inline code shows a link to write, not one to follow.
		text := regexp.MustCompile("`[^`]*`").ReplaceAllString(read(t, path), "")
		for _, m := range link.FindAllStringSubmatch(text, -1) {
			target := m[1]
			if strings.Contains(target, "://") {
				continue
			}
			full := filepath.Join(filepath.Dir(path), target)
			if strings.HasPrefix(full, "..") {
				t.Errorf("%s links %s, outside the skills", path, target)
				continue
			}
			if _, err := os.Stat(full); err != nil {
				t.Errorf("%s links %s, which does not exist", path, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestEmbedHoldsEverySkill: the binary carries every skill in the tree.
func TestEmbedHoldsEverySkill(t *testing.T) {
	embedded, err := Names(FS)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(embedded, " "), strings.Join(skillNames(t), " "); got != want {
		t.Errorf("embedded %q, tree has %q", got, want)
	}
}

// registered reads the tool names from references/tools.md, which
// TestSkillListsEveryTool in internal/tools keeps equal to the server.
func registered(t *testing.T) map[string]bool {
	t.Helper()
	row := regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\|")
	tools := map[string]bool{}
	for _, m := range row.FindAllStringSubmatch(read(t, "penpot-mcp/references/tools.md"), -1) {
		tools[m[1]] = true
	}
	if len(tools) == 0 {
		t.Fatal("no tools in references/tools.md")
	}
	return tools
}

// toolCells returns the backticked names in every table column headed
// "Tool".
func toolCells(doc string) []string {
	name := regexp.MustCompile("`([a-z_]+)`")
	var names []string
	col := -1
	for _, line := range strings.Split(doc, "\n") {
		if !strings.HasPrefix(line, "|") {
			col = -1
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if col < 0 {
			for i, c := range cells {
				if strings.TrimSpace(c) == "Tool" {
					col = i
				}
			}
			if col < 0 {
				col = len(cells) // a table with no Tool column: skip its rows
			}
			continue
		}
		if col < len(cells) {
			for _, m := range name.FindAllStringSubmatch(cells[col], -1) {
				names = append(names, m[1])
			}
		}
	}
	return names
}

// TestSkillToolsAreRegistered: every tool a skill's tables name is a real
// tool, and every real tool is named by some skill.
func TestSkillToolsAreRegistered(t *testing.T) {
	tools := registered(t)
	named := map[string]bool{}
	var all strings.Builder
	for _, skill := range skillNames(t) {
		doc := read(t, skill+"/SKILL.md")
		all.WriteString(doc)
		for _, n := range toolCells(doc) {
			if !tools[n] {
				t.Errorf("%s names %q, which the server does not register", skill, n)
			}
			named[n] = true
		}
	}
	for n := range tools {
		if !strings.Contains(all.String(), "`"+n+"`") {
			t.Errorf("no skill names %q", n)
		}
	}
}

// TestSkillsSayContentIsData: every skill reads design content as data.
func TestSkillsSayContentIsData(t *testing.T) {
	for _, skill := range skillNames(t) {
		doc := strings.Join(strings.Fields(read(t, skill+"/SKILL.md")), " ")
		if !strings.Contains(doc, "data, never instructions") && !strings.Contains(doc, "data, never as instructions") {
			t.Errorf("%s does not say design content is data, never instructions", skill)
		}
	}
}
