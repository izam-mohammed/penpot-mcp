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
		d := d.pick(ctx)
		out, err := addPage(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rename_page",
		Description: "Rename a page.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in renamePageIn) (*mcp.CallToolResult, pageOut, error) {
		d := d.pick(ctx)
		out, err := renamePage(ctx, d, in)
		return nil, out, err
	})

	registerPagesRest(s, d)
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

type deletePageIn struct {
	FileID  string `json:"file_id" jsonschema:"the file"`
	PageID  string `json:"page_id" jsonschema:"the page to delete, with everything on it"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"must be true; say so only after the user asked for this page to be deleted"`
}

type reorderPageIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	PageID string `json:"page_id" jsonschema:"the page to move"`
	Index  int    `json:"index" jsonschema:"its new position, 0 is first"`
}

func registerPagesRest(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_page",
		Description: "Delete a page and everything on it. Needs confirm: true, given only after the user asked. " +
			"A file's last page is never deleted.",
		Annotations: destructive,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deletePageIn) (*mcp.CallToolResult, pageOut, error) {
		d := d.pick(ctx)
		out, err := deletePage(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "reorder_pages",
		Description: "Move a page to another position in the file's page list.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in reorderPageIn) (*mcp.CallToolResult, pageOut, error) {
		d := d.pick(ctx)
		out, err := reorderPage(ctx, d, in)
		return nil, out, err
	})
}

func deletePage(ctx context.Context, d Deps, in deletePageIn) (pageOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return pageOut{}, err
	}
	if err := checkID("page_id", in.PageID); err != nil {
		return pageOut{}, err
	}
	if err := checkConfirm(in.Confirm, "a page"); err != nil {
		return pageOut{}, err
	}
	name, pages, err := d.Writer.Pages(ctx, in.FileID)
	if err != nil {
		return pageOut{}, err
	}
	if !pageListed(pages, in.PageID) {
		return pageOut{}, fmt.Errorf("page %s is not in file %q", in.PageID, name)
	}
	gone := ""
	for _, p := range pages {
		if p.ID == in.PageID {
			gone = p.Name
		}
	}
	if len(pages) == 1 {
		return pageOut{}, fmt.Errorf("%q is the last page of %q; Penpot needs one", gone, name)
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{{"type": "del-page", "id": in.PageID}}); err != nil {
		return pageOut{}, err
	}
	d.log().Info("deleted page", "file_id", in.FileID, "page_id", in.PageID, "name", gone)
	name, pages, err = d.Writer.Pages(ctx, in.FileID)
	if err != nil {
		return pageOut{}, err
	}
	return pageOut{FileID: in.FileID, FileName: name, PageID: in.PageID, PageName: gone, Pages: pages}, nil
}

func pageListed(pages []penpot.PageMeta, id string) bool {
	for _, p := range pages {
		if p.ID == id {
			return true
		}
	}
	return false
}

func reorderPage(ctx context.Context, d Deps, in reorderPageIn) (pageOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return pageOut{}, err
	}
	if err := checkID("page_id", in.PageID); err != nil {
		return pageOut{}, err
	}
	name, pages, err := d.Writer.Pages(ctx, in.FileID)
	if err != nil {
		return pageOut{}, err
	}
	if !pageListed(pages, in.PageID) {
		return pageOut{}, fmt.Errorf("page %s is not in file %q", in.PageID, name)
	}
	if in.Index < 0 || in.Index >= len(pages) {
		return pageOut{}, fmt.Errorf("index must be 0 to %d", len(pages)-1)
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{{"type": "mov-page", "id": in.PageID, "index": in.Index}}); err != nil {
		return pageOut{}, err
	}
	return pagesAfter(ctx, d, in.FileID, in.PageID)
}
