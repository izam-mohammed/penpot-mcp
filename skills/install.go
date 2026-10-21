// Package skills holds the Claude Code and Codex skills shipped with
// penpot-mcp, embedded in the binary so `penpot-mcp skills install` can
// write them without a checkout.
package skills

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// FS is every skill directory, as built into the binary.
//
//go:embed penpot-mcp penpot-mcp-*
var FS embed.FS

// Marker is the file that marks a skill directory as written by Install, so
// a later install may replace it.
const Marker = ".penpot-mcp"

var safeName = regexp.MustCompile(`^penpot-mcp(-[a-z]+)?$`)

// Names lists the skills in fsys: its top-level directories.
func Names(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !safeName.MatchString(e.Name()) {
			return nil, fmt.Errorf("skill %q: not a penpot-mcp skill name", e.Name())
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

// Install writes every skill in fsys to dir/<name>/, replacing a directory
// an earlier Install wrote. Without force it refuses a directory or symlink
// it did not write, and goes on with the rest. It prints one line per skill
// written to out.
func Install(fsys fs.FS, dir, version string, force bool, out io.Writer) error {
	names, err := Names(fsys)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	var refused []error
	for _, name := range names {
		target := filepath.Join(dir, name)
		if err := clear(target, force); err != nil {
			refused = append(refused, err)
			continue
		}
		sub, err := fs.Sub(fsys, name)
		if err != nil {
			return err
		}
		if err := os.CopyFS(target, sub); err != nil {
			return fmt.Errorf("%s: %w", target, err)
		}
		if err := os.WriteFile(filepath.Join(target, Marker), []byte(version+"\n"), 0o600); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "wrote %s\n", target)
	}
	return errors.Join(refused...)
}

// clear makes room for a skill at target: nothing there, or a directory
// Install wrote, or anything at all when forced.
func clear(target string, force bool) error {
	info, err := os.Lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !force {
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink (from make install-skill?); leaving it, use --force to replace it", target)
		}
		if _, err := os.Stat(filepath.Join(target, Marker)); err != nil {
			return fmt.Errorf("%s was not written by penpot-mcp; leaving it, use --force to replace it", target)
		}
	}
	return os.RemoveAll(target)
}
