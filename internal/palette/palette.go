// Package palette reads colour tokens from a repository's own files: Swift
// and Kotlin constants, markdown token tables, CSS custom properties, JSON
// token files and plain hex lists. The rules for Swift, Kotlin and markdown
// are ported from the penpot skill's make_bootstrap.py.
package palette

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxSize caps the file read, so a large file cannot be pulled in whole.
const MaxSize = 1 << 20

// Color is one token: Name may hold a group path ("Brand / Primary").
type Color struct {
	Name    string  `json:"name"`
	Hex     string  `json:"hex"` // #RRGGBB, upper case
	Opacity float64 `json:"opacity"`
}

var (
	swiftToken  = regexp.MustCompile(`static\s+let\s+(\w+)\s*(?::\s*\w+\s*)?=\s*(?:token|tone|hex|color)\(\s*"(#[0-9A-Fa-f]{6,8})"\s*\)`)
	kotlinColor = regexp.MustCompile(`\b(\w+)\s*=\s*Color\(\s*0x([0-9A-Fa-f]{6,8})\s*\)`)
	cssVar      = regexp.MustCompile(`--([A-Za-z0-9_-]+)\s*:\s*(#[0-9A-Fa-f]{3,8})\b`)
	hexRE       = regexp.MustCompile(`#[0-9A-Fa-f]{6,8}\b`)
	listLine    = regexp.MustCompile(`^\s*(?:([A-Za-z][\w ./-]*?)\s*[:=]\s*)?(#[0-9A-Fa-f]{6,8})\b`)
)

// Open checks that path is a regular text file inside root (after
// symlinks), at most MaxSize bytes, and returns its contents.
func Open(root, path string) ([]byte, error) {
	rootAbs, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(rootAbs, path)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", filepath.Base(path), err)
	}
	rel, err := filepath.Rel(rootAbs, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("%s is outside the working directory %s", path, rootAbs)
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == ".env" || strings.HasPrefix(part, ".env.") || part == ".git" {
			return nil, fmt.Errorf("%s is not a palette file", rel)
		}
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", rel)
	}
	if info.Size() > MaxSize {
		return nil, fmt.Errorf("%s is %d bytes; palette files are read up to %d", rel, info.Size(), MaxSize)
	}
	data, err := os.ReadFile(real) // #nosec G304 -- confined to root above
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s is not a text file", rel)
	}
	return data, nil
}

// Parse reads colours from data, picking the reader by the file name.
// alphaFirst says whether an 8-digit hex is #AARRGGBB (Android, Swift) or
// #RRGGBBAA (CSS); Kotlin's Color(0x...) is always alpha first.
func Parse(name string, data []byte, alphaFirst bool) ([]Color, error) {
	text := string(data)
	var out []Color
	var err error
	switch ext := strings.ToLower(filepath.Ext(name)); ext {
	case ".swift":
		out, err = fromMatches(swiftToken.FindAllStringSubmatch(text, -1), alphaFirst)
	case ".kt", ".kts":
		out, err = fromMatches(prefixHash(kotlinColor.FindAllStringSubmatch(text, -1)), true)
	case ".md", ".markdown":
		out, err = fromMarkdown(text, alphaFirst)
	case ".css", ".scss", ".less":
		out, err = fromCSS(text, alphaFirst)
	case ".json":
		out, err = fromJSON(data, alphaFirst)
	default:
		out, err = fromList(text, alphaFirst)
	}
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no colours found in %s", filepath.Base(name))
	}
	return dedupe(out), nil
}

func prefixHash(ms [][]string) [][]string {
	for _, m := range ms {
		m[2] = "#" + m[2]
	}
	return ms
}

func fromMatches(ms [][]string, alphaFirst bool) ([]Color, error) {
	var out []Color
	for _, m := range ms {
		c, err := split(m[1], m[2], alphaFirst)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// fromMarkdown reads the richest table whose first header is Token, Name,
// Colour or Color; rows whose value holds no hex are skipped.
func fromMarkdown(text string, alphaFirst bool) ([]Color, error) {
	var best, rows [][]string
	var header []string
	inTable := false
	flush := func() {
		if inTable && len(header) > 0 {
			switch strings.ToLower(strings.Trim(header[0], "` ")) {
			case "token", "name", "colour", "color":
				if len(rows) > len(best) {
					best = rows
				}
			}
		}
		inTable, header, rows = false, nil, nil
	}
	for _, raw := range append(strings.Split(text, "\n"), "") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") && len(line) > 1 {
			cells := strings.Split(strings.Trim(line, "|"), "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			if !inTable {
				inTable, header = true, cells
				continue
			}
			if strings.Trim(strings.Join(cells, ""), "-: ") == "" {
				continue
			}
			rows = append(rows, cells)
			continue
		}
		flush()
	}
	var out []Color
	for _, r := range best {
		if len(r) < 2 {
			continue
		}
		h := hexRE.FindString(r[1])
		if h == "" {
			continue
		}
		c, err := split(strings.ReplaceAll(r[0], "`", ""), h, alphaFirst)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func fromCSS(text string, alphaFirst bool) ([]Color, error) {
	var out []Color
	for _, m := range cssVar.FindAllStringSubmatch(text, -1) {
		h := m[2]
		if len(h) == 4 { // #RGB
			h = "#" + strings.Repeat(h[1:2], 2) + strings.Repeat(h[2:3], 2) + strings.Repeat(h[3:4], 2)
		}
		if len(h) != 7 && len(h) != 9 {
			continue
		}
		c, err := split(m[1], h, alphaFirst)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// fromJSON reads design-token JSON: DTCG ({"$value", "$type": "color"}),
// Style Dictionary ({"value"}) or a flat {"name": "#hex"} map. Nested
// groups become the path.
func fromJSON(data []byte, alphaFirst bool) ([]Color, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("not JSON: %w", err)
	}
	var out []Color
	var walk func(path []string, v any) error
	walk = func(path []string, v any) error {
		switch t := v.(type) {
		case string:
			if len(path) > 0 && hexRE.MatchString(t) && hexRE.FindString(t) == t {
				c, err := split(strings.Join(path, " / "), t, alphaFirst)
				if err != nil {
					return err
				}
				out = append(out, c)
			}
		case map[string]any:
			for _, key := range []string{"$value", "value"} {
				if val, ok := t[key].(string); ok {
					if typ, ok := t["$type"].(string); ok && typ != "color" {
						return nil
					}
					return walk(path, val)
				}
			}
			keys := make([]string, 0, len(t))
			for k := range t {
				if !strings.HasPrefix(k, "$") {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			for _, k := range keys {
				if err := walk(append(append([]string(nil), path...), k), t[k]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(nil, doc); err != nil {
		return nil, err
	}
	return out, nil
}

// fromList reads one colour per line: "#hex", "name: #hex" or "name = #hex".
func fromList(text string, alphaFirst bool) ([]Color, error) {
	var out []Color
	for i, line := range strings.Split(text, "\n") {
		m := listLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			name = "Color " + strconv.Itoa(i+1)
		}
		c, err := split(name, m[2], alphaFirst)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// split turns a 6- or 8-digit hex into #RRGGBB and an opacity.
func split(name, hex string, alphaFirst bool) (Color, error) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	name = strings.TrimSpace(name)
	if name == "" {
		return Color{}, errors.New("a colour has no name")
	}
	switch len(h) {
	case 6:
		return Color{Name: name, Hex: "#" + strings.ToUpper(h), Opacity: 1}, nil
	case 8:
		rgb, a := h[:6], h[6:]
		if alphaFirst {
			rgb, a = h[2:], h[:2]
		}
		v, err := strconv.ParseUint(a, 16, 8)
		if err != nil {
			return Color{}, err
		}
		op, _ := strconv.ParseFloat(strconv.FormatFloat(float64(v)/255, 'f', 4, 64), 64)
		return Color{Name: name, Hex: "#" + strings.ToUpper(rgb), Opacity: op}, nil
	}
	return Color{}, fmt.Errorf("%s: %q is not a 6- or 8-digit hex", name, hex)
}

// dedupe keeps the first colour of each name.
func dedupe(in []Color) []Color {
	seen := map[string]bool{}
	out := in[:0]
	for _, c := range in {
		if !seen[c.Name] {
			seen[c.Name] = true
			out = append(out, c)
		}
	}
	return out
}
