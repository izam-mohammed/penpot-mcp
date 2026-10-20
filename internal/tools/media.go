package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// maxImageBytes caps what create_image reads from disk.
const maxImageBytes = 20 << 20

// readImage reads a local image file. Only regular files whose bytes are a
// PNG, JPEG, WebP, GIF or SVG are read, so a model cannot use it to pull
// other files into Penpot.
func readImage(path string) (data []byte, mtype, name string, err error) {
	if path == "" || !filepath.IsAbs(path) {
		return nil, "", "", fmt.Errorf("image_path must be an absolute path")
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, "", "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, "", "", fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, "", "", fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > maxImageBytes {
		return nil, "", "", fmt.Errorf("%s is larger than %d MB", path, maxImageBytes>>20)
	}
	data, err = io.ReadAll(io.LimitReader(f, maxImageBytes+1))
	if err != nil {
		return nil, "", "", fmt.Errorf("read %s: %w", path, err)
	}
	mtype = imageType(data, path)
	if mtype == "" {
		return nil, "", "", fmt.Errorf("%s is not a PNG, JPEG, WebP, GIF or SVG image", path)
	}
	name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return data, mtype, name, nil
}

func imageType(data []byte, path string) string {
	switch t := http.DetectContentType(data); t {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return t
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		head := data
		if len(head) > 4096 {
			head = head[:4096]
		}
		if bytes.Contains(head, []byte("<svg")) {
			return "image/svg+xml"
		}
	}
	return ""
}

// uploadImage stores a local image in the file's media.
func uploadImage(ctx context.Context, d Deps, fileID, path string) (penpot.Media, error) {
	data, mtype, name, err := readImage(path)
	if err != nil {
		return penpot.Media{}, err
	}
	var m penpot.Media
	err = d.Client.Upload(ctx, "upload-file-media-object", map[string]string{
		"file-id": fileID, "is-local": "true", "name": name,
	}, "content", filepath.Base(path), mtype, data, &m)
	if err != nil {
		return penpot.Media{}, err
	}
	if m.ID == "" || m.Width <= 0 || m.Height <= 0 {
		return penpot.Media{}, fmt.Errorf("penpot did not return the uploaded image's size")
	}
	return m, nil
}
