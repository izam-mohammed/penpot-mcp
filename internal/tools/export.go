package tools

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/palette"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
	"github.com/izam-mohammed/penpot-mcp/internal/transit"
)

// maxBoards caps one export_boards call.
const maxBoards = 100

var formats = map[string]string{"png": ".png", "jpeg": ".jpg", "webp": ".webp", "svg": ".svg", "pdf": ".pdf"}

type exportShapeIn struct {
	FileID    string  `json:"file_id" jsonschema:"the file"`
	PageID    string  `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	ShapeID   string  `json:"shape_id" jsonschema:"the board or shape to export"`
	Format    string  `json:"format,omitempty" jsonschema:"png (default), jpeg, webp, svg or pdf"`
	Scale     float64 `json:"scale,omitempty" jsonschema:"0.25 to 4, default 1; 2 for retina PNGs"`
	Path      string  `json:"path" jsonschema:"where to write it, inside the working directory, with the format's extension"`
	Overwrite bool    `json:"overwrite,omitempty" jsonschema:"replace a file already at path"`
}

type exportBoardsIn struct {
	FileID    string  `json:"file_id" jsonschema:"the file"`
	PageID    string  `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	Dir       string  `json:"dir" jsonschema:"folder inside the working directory; each board is written as <board name>.<ext>"`
	Prefix    string  `json:"prefix,omitempty" jsonschema:"only boards whose name starts with this"`
	Format    string  `json:"format,omitempty" jsonschema:"png (default), jpeg, webp, svg or pdf"`
	Scale     float64 `json:"scale,omitempty" jsonschema:"0.25 to 4, default 1"`
	Overwrite bool    `json:"overwrite,omitempty" jsonschema:"replace files already there"`
}

type exportFileIn struct {
	FileID    string `json:"file_id" jsonschema:"the file"`
	Path      string `json:"path" jsonschema:"where to write the .penpot file, inside the working directory"`
	Overwrite bool   `json:"overwrite,omitempty" jsonschema:"replace a file already at path"`
}

type importFileIn struct {
	ProjectID string `json:"project_id" jsonschema:"the project to import into"`
	Path      string `json:"path" jsonschema:"a .penpot file inside the working directory"`
	Name      string `json:"name,omitempty" jsonschema:"the new file's name; the file name when empty"`
}

type written struct {
	Path   string `json:"path"`
	Name   string `json:"name,omitempty"`
	Bytes  int    `json:"bytes"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type exportOut struct {
	FileID string    `json:"file_id"`
	Files  []written `json:"files"`
}

type fileImportOut struct {
	ProjectID string   `json:"project_id"`
	FileIDs   []string `json:"file_ids"`
}

func registerExport(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "export_shape",
		Description: "Render a board or shape to PNG, JPEG, WEBP, SVG or PDF and write it to a path in the working directory. " +
			"Needs PENPOT_EMAIL and PENPOT_PASSWORD: Penpot's exporter takes only a login session. A blank render is an error.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in exportShapeIn) (*mcp.CallToolResult, exportOut, error) {
		out, err := exportShape(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "export_boards",
		Description: "Export every top-level board of a page (optionally only names with a prefix) into a folder, one file per board.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in exportBoardsIn) (*mcp.CallToolResult, exportOut, error) {
		out, err := exportBoards(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "export_file",
		Description: "Download a whole file as .penpot (Penpot's own format, re-importable) to a path in the working directory.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in exportFileIn) (*mcp.CallToolResult, exportOut, error) {
		out, err := exportFile(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "import_file",
		Description: "Import a .penpot file from the working directory into a project as a new file.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in importFileIn) (*mcp.CallToolResult, fileImportOut, error) {
		out, err := importFile(ctx, d, in)
		return nil, out, err
	})
}

// outPath checks that path is inside root, has one of exts, and is not an
// existing file unless overwrite; it creates the parent folders.
func outPath(root, path string, exts []string, overwrite bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is required")
	}
	rootAbs, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(rootAbs, path)
	}
	path = filepath.Clean(path)
	ext := strings.ToLower(filepath.Ext(path))
	ok := false
	for _, e := range exts {
		ok = ok || ext == e
	}
	if !ok {
		return "", fmt.Errorf("path must end in %s", strings.Join(exts, " or "))
	}
	// Resolve symlinks on the deepest folder that exists.
	dir := filepath.Dir(path)
	existing := dir
	for {
		if _, err := os.Stat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			break
		}
		existing = parent
	}
	realDir, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	rest, _ := filepath.Rel(existing, dir)
	full := filepath.Join(realDir, rest, filepath.Base(path))
	rel, err := filepath.Rel(rootAbs, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the working directory %s", path, rootAbs)
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == ".git" {
			return "", errors.New("exports never go inside .git")
		}
	}
	if info, err := os.Lstat(full); err == nil {
		if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", fmt.Errorf("%s exists and is not a plain file", rel)
		}
		if !overwrite {
			return "", fmt.Errorf("%s exists; pass overwrite: true to replace it", rel)
		}
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return "", err
	}
	return full, nil
}

func writeOut(path string, data []byte, overwrite bool) error {
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o600) // #nosec G304 -- checked by outPath
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// checkPNG reports a truncated or blank (single-colour) PNG, as the penpot
// skill's check_boards.py does, and returns its size.
func checkPNG(data []byte) (w, h int, err error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, fmt.Errorf("the PNG is truncated or broken: %w", err)
	}
	b := img.Bounds()
	if b.Empty() {
		return 0, 0, errors.New("the PNG is empty")
	}
	if flat(img) {
		return b.Dx(), b.Dy(), errors.New("the PNG is a single flat colour: the render came out blank")
	}
	return b.Dx(), b.Dy(), nil
}

func flat(img image.Image) bool {
	b := img.Bounds()
	r0, g0, b0, a0 := img.At(b.Min.X, b.Min.Y).RGBA()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if r != r0 || g != g0 || bl != b0 || a != a0 {
				return false
			}
		}
	}
	return true
}

func formatOf(f string) (string, string, error) {
	if f == "" {
		f = "png"
	}
	ext, ok := formats[f]
	if !ok {
		return "", "", errors.New("format must be png, jpeg, webp, svg or pdf")
	}
	return f, ext, nil
}

func checkScale(s float64) error {
	if s != 0 && (s < 0.25 || s > 4) {
		return errors.New("scale must be 0.25 to 4")
	}
	return nil
}

// renderTo renders one shape and writes it to path.
func renderTo(ctx context.Context, d Deps, t penpot.Target, id, name, format string, scale float64, path string, overwrite bool) (written, error) {
	data, err := d.Exporter.Render(ctx, penpot.Export{
		FileID: t.FileID, PageID: t.PageID, ObjectID: id, Type: format, Scale: scale, Name: name,
	})
	if err != nil {
		return written{}, err
	}
	w := written{Path: path, Name: name, Bytes: len(data)}
	if format == "png" {
		if w.Width, w.Height, err = checkPNG(data); err != nil {
			return written{}, fmt.Errorf("%q: %w; nothing was written", name, err)
		}
	}
	if err := writeOut(path, data, overwrite); err != nil {
		return written{}, err
	}
	return w, nil
}

func exportShape(ctx context.Context, d Deps, in exportShapeIn) (exportOut, error) {
	ref := pageRef{FileID: in.FileID, PageID: in.PageID}
	if err := ref.validate(); err != nil {
		return exportOut{}, err
	}
	if err := checkID("shape_id", in.ShapeID); err != nil {
		return exportOut{}, err
	}
	format, ext, err := formatOf(in.Format)
	if err != nil {
		return exportOut{}, err
	}
	if err := checkScale(in.Scale); err != nil {
		return exportOut{}, err
	}
	exts := []string{ext}
	if format == "jpeg" {
		exts = append(exts, ".jpeg")
	}
	path, err := outPath(d.Root, in.Path, exts, in.Overwrite)
	if err != nil {
		return exportOut{}, err
	}
	if d.Exporter == nil {
		return exportOut{}, penpot.ErrNoExporter
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return exportOut{}, err
	}
	p, err := loadRaw(ctx, d, t)
	if err != nil {
		return exportOut{}, err
	}
	o, err := p.get(in.ShapeID, t)
	if err != nil {
		return exportOut{}, err
	}
	w, err := renderTo(ctx, d, t, in.ShapeID, str(o, "name"), format, in.Scale, path, in.Overwrite)
	if err != nil {
		return exportOut{}, err
	}
	return exportOut{FileID: in.FileID, Files: []written{w}}, nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._ -]+`)

// fileName turns a board name into a safe file name.
func fileName(name string) string {
	n := strings.Trim(unsafeName.ReplaceAllString(name, "-"), " .-")
	if n == "" {
		n = "board"
	}
	if len(n) > 120 {
		n = n[:120]
	}
	return n
}

func exportBoards(ctx context.Context, d Deps, in exportBoardsIn) (exportOut, error) {
	ref := pageRef{FileID: in.FileID, PageID: in.PageID}
	if err := ref.validate(); err != nil {
		return exportOut{}, err
	}
	format, ext, err := formatOf(in.Format)
	if err != nil {
		return exportOut{}, err
	}
	if err := checkScale(in.Scale); err != nil {
		return exportOut{}, err
	}
	if strings.TrimSpace(in.Dir) == "" {
		return exportOut{}, errors.New("dir is required")
	}
	if d.Exporter == nil {
		return exportOut{}, penpot.ErrNoExporter
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return exportOut{}, err
	}
	p, err := loadRaw(ctx, d, t)
	if err != nil {
		return exportOut{}, err
	}
	type board struct{ id, name, path string }
	var boards []board
	seen := map[string]bool{}
	for _, id := range children(p.objects[penpot.ZeroID]) {
		o := p.objects[id]
		name := str(o, "name")
		if str(o, "type") != penpot.TypeBoard || !strings.HasPrefix(name, in.Prefix) {
			continue
		}
		file := fileName(name)
		if seen[file] {
			return exportOut{}, fmt.Errorf("two boards would both be written as %s%s; rename one", file, ext)
		}
		seen[file] = true
		path, err := outPath(d.Root, filepath.Join(in.Dir, file+ext), []string{ext}, in.Overwrite)
		if err != nil {
			return exportOut{}, err
		}
		boards = append(boards, board{id, name, path})
	}
	if len(boards) == 0 {
		return exportOut{}, fmt.Errorf("no top-level boards on page %q start with %q", t.PageName, in.Prefix)
	}
	if len(boards) > maxBoards {
		return exportOut{}, fmt.Errorf("%d boards match; export at most %d per call (use prefix)", len(boards), maxBoards)
	}
	out := exportOut{FileID: in.FileID}
	for _, b := range boards {
		w, err := renderTo(ctx, d, t, b.id, b.name, format, in.Scale, b.path, in.Overwrite)
		if err != nil {
			return out, fmt.Errorf("after %d of %d boards: %w", len(out.Files), len(boards), err)
		}
		out.Files = append(out.Files, w)
	}
	return out, nil
}

// sseEnd returns the data of a server-sent-event stream's "end" event, or
// the error an "error" event carries.
func sseEnd(stream []byte) (string, error) {
	event := ""
	sc := bufio.NewScanner(bytes.NewReader(stream))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			switch event {
			case "end":
				return data, nil
			case "error":
				return "", fmt.Errorf("penpot: %s", data)
			}
		}
	}
	return "", errors.New("the stream ended without an end event")
}

func exportFile(ctx context.Context, d Deps, in exportFileIn) (exportOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return exportOut{}, err
	}
	path, err := outPath(d.Root, in.Path, []string{".penpot"}, in.Overwrite)
	if err != nil {
		return exportOut{}, err
	}
	var stream []byte
	if err := d.Client.Call(ctx, "export-binfile", map[string]any{
		"fileId": in.FileID, "includeLibraries": false, "embedAssets": false,
	}, &stream); err != nil {
		return exportOut{}, err
	}
	end, err := sseEnd(stream)
	if err != nil {
		return exportOut{}, err
	}
	v, err := transit.Unmarshal([]byte(end))
	if err != nil {
		return exportOut{}, fmt.Errorf("reading the export's end event: %w", err)
	}
	uri, _ := v.(string)
	if uri == "" {
		return exportOut{}, fmt.Errorf("the export's end event has no download uri")
	}
	data, err := d.Client.Download(ctx, uri, penpot.MaxExport)
	if err != nil {
		return exportOut{}, err
	}
	if err := writeOut(path, data, in.Overwrite); err != nil {
		return exportOut{}, err
	}
	return exportOut{FileID: in.FileID, Files: []written{{Path: path, Bytes: len(data)}}}, nil
}

func importFile(ctx context.Context, d Deps, in importFileIn) (fileImportOut, error) {
	if err := checkID("project_id", in.ProjectID); err != nil {
		return fileImportOut{}, err
	}
	if strings.ToLower(filepath.Ext(in.Path)) != ".penpot" {
		return fileImportOut{}, errors.New("path must be a .penpot file")
	}
	data, err := palette.OpenBinary(d.Root, in.Path, penpot.MaxExport)
	if err != nil {
		return fileImportOut{}, err
	}
	name := in.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(in.Path), filepath.Ext(in.Path))
	}
	if err := checkName(name); err != nil {
		return fileImportOut{}, err
	}
	var stream []byte
	if err := d.Client.Upload(ctx, "import-binfile", map[string]string{"name": name, "project-id": in.ProjectID},
		"file", filepath.Base(in.Path), "application/zip", data, &stream); err != nil {
		return fileImportOut{}, err
	}
	end, err := sseEnd(stream)
	if err != nil {
		return fileImportOut{}, err
	}
	v, err := transit.Unmarshal([]byte(end))
	if err != nil {
		return fileImportOut{}, fmt.Errorf("reading the import's end event: %w", err)
	}
	out := fileImportOut{ProjectID: in.ProjectID, FileIDs: []string{}}
	if list, ok := v.([]any); ok {
		for _, id := range list {
			if s, ok := id.(string); ok {
				out.FileIDs = append(out.FileIDs, s)
			}
		}
	}
	return out, nil
}
