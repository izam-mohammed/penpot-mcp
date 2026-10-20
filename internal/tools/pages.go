package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

type addPageIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	Name   string `json:"name" jsonschema:"page name"`
}

type renamePageIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	PageID string `json:"page_id" jsonschema:"the page to rename"`
	Name   string `json:"name" jsonschema:"new page name"`
}

type pageOut struct {
	FileID   string            `json:"file_id"`
	FileName string            `json:"file_name"`
	PageID   string            `json:"page_id"`
	PageName string            `json:"page_name"`
	Pages    []penpot.PageMeta `json:"pages"`
}

func registerPages(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_page",
		Description: "Add an empty page at the end of a file. Returns its id and the file's pages in order.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in addPageIn) (*mcp.CallToolResult, pageOut, error) {
		out, err := addPage(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rename_page",
		Description: "Rename a page.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in renamePageIn) (*mcp.CallToolResult, pageOut, error) {
		out, err := renamePage(ctx, d, in)
		return nil, out, err
	})
}

func addPage(ctx context.Context, d Deps, in addPageIn) (pageOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return pageOut{}, err
	}
	if err := checkName(in.Name); err != nil {
		return pageOut{}, err
	}
	// The {id, name} form makes Penpot build the empty page and its root frame.
	id := penpot.NewID()
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{{"type": "add-page", "id": id, "name": in.Name}}); err != nil {
		return pageOut{}, err
	}
	return pagesAfter(ctx, d, in.FileID, id)
}

func renamePage(ctx context.Context, d Deps, in renamePageIn) (pageOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return pageOut{}, err
	}
	if err := checkID("page_id", in.PageID); err != nil {
		return pageOut{}, err
	}
	if err := checkName(in.Name); err != nil {
		return pageOut{}, err
	}
	if _, err := d.Writer.Resolve(ctx, in.FileID, in.PageID); err != nil {
		return pageOut{}, err
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{{"type": "mod-page", "id": in.PageID, "name": in.Name}}); err != nil {
		return pageOut{}, err
	}
	return pagesAfter(ctx, d, in.FileID, in.PageID)
}

// pagesAfter re-reads the file's pages, so the cache and the reply show the
// change, and picks out pageID.
func pagesAfter(ctx context.Context, d Deps, fileID, pageID string) (pageOut, error) {
	name, pages, err := d.Writer.Pages(ctx, fileID)
	if err != nil {
		return pageOut{}, err
	}
	for _, p := range pages {
		if p.ID == pageID {
			return pageOut{FileID: fileID, FileName: name, PageID: p.ID, PageName: p.Name, Pages: pages}, nil
		}
	}
	return pageOut{}, fmt.Errorf("page %s is not in file %q after the write; reload and check", pageID, name)
}
