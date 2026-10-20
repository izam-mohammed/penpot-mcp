package penpot

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/izam-mohammed/penpot-mcp/internal/transit"
)

// fileState is the cached revision of a file, refreshed on conflicts.
type fileState struct {
	name    string
	revn    int64
	vern    int64
	canEdit bool
	pages   []PageMeta
}

// Writer sends update-file calls. Penpot allows one update-file in flight per
// profile, so every write goes through one lock.
type Writer struct {
	c         *Client
	sessionID string

	mu    sync.Mutex
	files map[string]*fileState
}

// NewWriter returns a writer with a fresh session id.
func NewWriter(c *Client) *Writer {
	return &Writer{c: c, sessionID: NewID(), files: map[string]*fileState{}}
}

// Target is what a write landed in, echoed back to the caller.
type Target struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	PageID   string `json:"page_id"`
	PageName string `json:"page_name"`
}

// Resolve loads the file (cached) and picks pageID, or the first page when
// pageID is empty.
func (w *Writer) Resolve(ctx context.Context, fileID, pageID string) (Target, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	st, err := w.state(ctx, fileID, false)
	if err != nil {
		return Target{}, err
	}
	return pick(fileID, st, pageID)
}

func pick(fileID string, st *fileState, pageID string) (Target, error) {
	if len(st.pages) == 0 {
		return Target{}, fmt.Errorf("file %q has no pages", st.name)
	}
	p := st.pages[0]
	if pageID != "" {
		found := false
		for _, cand := range st.pages {
			if cand.ID == pageID {
				p, found = cand, true
				break
			}
		}
		if !found {
			return Target{}, fmt.Errorf("page %s is not in file %q", pageID, st.name)
		}
	}
	return Target{FileID: fileID, FileName: st.name, PageID: p.ID, PageName: p.Name}, nil
}

// Pages returns the file's pages in order.
func (w *Writer) Pages(ctx context.Context, fileID string) (string, []PageMeta, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	st, err := w.state(ctx, fileID, true)
	if err != nil {
		return "", nil, err
	}
	return st.name, append([]PageMeta(nil), st.pages...), nil
}

// Forget drops the cached state of a file, after a call that changed its
// name or pages outside Apply.
func (w *Writer) Forget(fileID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.files, fileID)
}

func (w *Writer) state(ctx context.Context, fileID string, refresh bool) (*fileState, error) {
	if st, ok := w.files[fileID]; ok && !refresh {
		return st, nil
	}
	var f File
	if err := w.c.Call(ctx, "get-file", map[string]string{"id": fileID}, &f); err != nil {
		return nil, err
	}
	st := &fileState{name: f.Name, revn: f.Revn, vern: f.Vern, canEdit: f.Permissions.CanEdit}
	for _, id := range f.Data.Pages {
		p := f.Data.PagesIndex[id]
		st.pages = append(st.pages, PageMeta{ID: id, Name: p.Name})
	}
	w.files[fileID] = st
	return st, nil
}

// Apply sends changes to a file in one update-file call. On a revn or vern
// conflict it re-reads the file once and retries.
func (w *Writer) Apply(ctx context.Context, fileID string, changes []map[string]any) error {
	return w.apply(ctx, fileID, func(st *fileState, res *UpdateResult) error {
		return w.c.Call(ctx, "update-file", map[string]any{
			"id":        fileID,
			"sessionId": w.sessionID,
			"revn":      st.revn,
			"vern":      st.vern,
			"changes":   changes,
		}, res)
	})
}

// ApplyTransit is Apply for changes Penpot decodes only from transit, such as
// mov-objects.
func (w *Writer) ApplyTransit(ctx context.Context, fileID string, changes []transit.Map) error {
	items := make([]any, len(changes))
	for i, c := range changes {
		items[i] = c
	}
	return w.apply(ctx, fileID, func(st *fileState, res *UpdateResult) error {
		body, err := transit.Marshal(transit.Map{
			{K: "id", V: transit.UUID(fileID)},
			{K: "session-id", V: transit.UUID(w.sessionID)},
			{K: "revn", V: st.revn},
			{K: "vern", V: st.vern},
			{K: "changes", V: items},
		})
		if err != nil {
			return err
		}
		return w.c.CallTransit(ctx, "update-file", body, res)
	})
}

func (w *Writer) apply(ctx context.Context, fileID string, send func(*fileState, *UpdateResult) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	st, err := w.state(ctx, fileID, false)
	if err != nil {
		return err
	}
	if !st.canEdit {
		return fmt.Errorf("this token cannot edit file %q", st.name)
	}
	for attempt := 0; ; attempt++ {
		var res UpdateResult
		err := send(st, &res)
		if err == nil {
			st.revn = res.Revn + 1
			return nil
		}
		if attempt == 0 && (IsCode(err, "vern-conflict") || IsCode(err, "revn-conflict")) {
			if st, err = w.state(ctx, fileID, true); err != nil {
				return err
			}
			continue
		}
		return err
	}
}

// NewID returns a random UUID v4.
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
