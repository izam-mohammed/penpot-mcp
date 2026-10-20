package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type snapshotIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	Label  string `json:"label,omitempty" jsonschema:"a name for the version, such as before redesign"`
}

type restoreIn struct {
	FileID     string `json:"file_id" jsonschema:"the file"`
	SnapshotID string `json:"snapshot_id" jsonschema:"the version to go back to, from list_snapshots"`
	Confirm    bool   `json:"confirm,omitempty" jsonschema:"must be true; the file goes back to that version, so say so only after the user asked"`
}

type snapshot struct {
	ID      string `json:"id"`
	Label   string `json:"label,omitempty"`
	Created string `json:"created_at,omitempty"`
	By      string `json:"created_by,omitempty"`
}

type snapshotsOut struct {
	FileID    string     `json:"file_id"`
	Snapshots []snapshot `json:"snapshots"`
	Safety    *snapshot  `json:"safety_snapshot,omitempty" jsonschema:"the version taken of the current state before a restore"`
}

type pluginDataIn struct {
	FileID    string  `json:"file_id" jsonschema:"the file"`
	PageID    string  `json:"page_id,omitempty" jsonschema:"the shape's page; the first page when empty"`
	ShapeID   string  `json:"shape_id,omitempty" jsonschema:"the shape; the file itself when empty"`
	Namespace string  `json:"namespace" jsonschema:"your namespace, such as a plugin or tool name"`
	Key       string  `json:"key" jsonschema:"the key"`
	Value     *string `json:"value,omitempty" jsonschema:"set it to this; omit to read it, empty string to remove it"`
}

type pluginDataOut struct {
	Namespace string  `json:"namespace"`
	Key       string  `json:"key"`
	Value     *string `json:"value"`
}

func registerHistory(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_snapshots",
		Description: "A file's saved versions, newest first.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in snapshotIn) (*mcp.CallToolResult, snapshotsOut, error) {
		d := d.pick(ctx)
		out, err := listSnapshots(ctx, d, in.FileID)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_snapshot",
		Description: "Save a named version of a file, to come back to before a big edit.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in snapshotIn) (*mcp.CallToolResult, snapshot, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, snapshot{}, err
		}
		if in.Label != "" {
			if err := checkName(in.Label); err != nil {
				return nil, snapshot{}, fmt.Errorf("label: %w", err)
			}
		}
		snap, err := createSnapshot(ctx, d, in.FileID, in.Label)
		return nil, snap, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "restore_snapshot",
		Description: "Put a file back to a saved version. A version of the current state is saved first, " +
			"so the restore can be undone. Needs confirm: true.",
		Annotations: destructive,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in restoreIn) (*mcp.CallToolResult, snapshotsOut, error) {
		d := d.pick(ctx)
		out, err := restoreSnapshot(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "plugin_data",
		Description: "Read or write a small key-value note on a file or a shape, under a namespace, " +
			"for tools that need to remember things in the design itself.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in pluginDataIn) (*mcp.CallToolResult, pluginDataOut, error) {
		d := d.pick(ctx)
		out, err := pluginData(ctx, d, in)
		return nil, out, err
	})
}

func listSnapshots(ctx context.Context, d Deps, fileID string) (snapshotsOut, error) {
	if err := checkID("file_id", fileID); err != nil {
		return snapshotsOut{}, err
	}
	var rows []map[string]any
	if err := d.Client.Call(ctx, "get-file-snapshots", map[string]string{"fileId": fileID}, &rows); err != nil {
		return snapshotsOut{}, err
	}
	out := snapshotsOut{FileID: fileID, Snapshots: []snapshot{}}
	for _, r := range rows {
		out.Snapshots = append(out.Snapshots, snapshot{ID: str(r, "id"), Label: str(r, "label"), Created: str(r, "createdAt"), By: str(r, "createdBy")})
	}
	return out, nil
}

func createSnapshot(ctx context.Context, d Deps, fileID, label string) (snapshot, error) {
	params := map[string]string{"fileId": fileID}
	if label != "" {
		params["label"] = label
	}
	var r map[string]any
	if err := d.Client.Call(ctx, "create-file-snapshot", params, &r); err != nil {
		return snapshot{}, err
	}
	return snapshot{ID: str(r, "id"), Label: label, Created: str(r, "createdAt")}, nil
}

func restoreSnapshot(ctx context.Context, d Deps, in restoreIn) (snapshotsOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return snapshotsOut{}, err
	}
	if err := checkID("snapshot_id", in.SnapshotID); err != nil {
		return snapshotsOut{}, err
	}
	if !in.Confirm {
		return snapshotsOut{}, fmt.Errorf("restoring replaces the file's current state: needs confirm: true; ask the user first")
	}
	have, err := listSnapshots(ctx, d, in.FileID)
	if err != nil {
		return snapshotsOut{}, err
	}
	found := false
	for _, s := range have.Snapshots {
		found = found || s.ID == in.SnapshotID
	}
	if !found {
		return snapshotsOut{}, fmt.Errorf("snapshot %s is not a version of this file", in.SnapshotID)
	}
	safety, err := createSnapshot(ctx, d, in.FileID, "before restore "+time.Now().UTC().Format("2006-01-02 15:04"))
	if err != nil {
		return snapshotsOut{}, fmt.Errorf("saving the current state first: %w; nothing was restored", err)
	}
	if err := d.Client.Call(ctx, "restore-file-snapshot", map[string]string{"fileId": in.FileID, "id": in.SnapshotID}, nil); err != nil {
		return snapshotsOut{}, err
	}
	d.Writer.Forget(in.FileID)
	d.log().Info("restored snapshot", "file_id", in.FileID, "snapshot_id", in.SnapshotID, "safety_snapshot", safety.ID)
	return snapshotsOut{FileID: in.FileID, Snapshots: []snapshot{{ID: in.SnapshotID}}, Safety: &safety}, nil
}

func pluginData(ctx context.Context, d Deps, in pluginDataIn) (pluginDataOut, error) {
	ref := pageRef{FileID: in.FileID, PageID: in.PageID}
	if err := ref.validate(); err != nil {
		return pluginDataOut{}, err
	}
	if !refRE.MatchString(in.Namespace) || !refRE.MatchString(in.Key) {
		return pluginDataOut{}, fmt.Errorf("namespace and key must be 1 to 64 letters, digits, _, . or -")
	}
	if in.Value != nil && len(*in.Value) > 10_000 {
		return pluginDataOut{}, fmt.Errorf("value must be at most 10000 bytes")
	}
	out := pluginDataOut{Namespace: in.Namespace, Key: in.Key}
	var data map[string]any
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return pluginDataOut{}, err
	}
	if in.ShapeID != "" {
		if err := checkID("shape_id", in.ShapeID); err != nil {
			return pluginDataOut{}, err
		}
		p, err := loadRaw(ctx, d, t)
		if err != nil {
			return pluginDataOut{}, err
		}
		o, err := p.get(in.ShapeID, t)
		if err != nil {
			return pluginDataOut{}, err
		}
		data, _ = o["pluginData"].(map[string]any)
	} else {
		var f struct {
			Data struct {
				PluginData map[string]any `json:"pluginData"`
			} `json:"data"`
		}
		if err := d.Client.Call(ctx, "get-file", map[string]string{"id": in.FileID}, &f); err != nil {
			return pluginDataOut{}, err
		}
		data = f.Data.PluginData
	}
	if in.Value == nil {
		if ns, ok := data[in.Namespace].(map[string]any); ok {
			if v, ok := ns[in.Key].(string); ok {
				out.Value = &v
			}
		}
		return out, nil
	}
	change := map[string]any{"type": "set-plugin-data", "objectType": "file", "namespace": in.Namespace, "key": in.Key}
	if in.ShapeID != "" {
		change["objectType"], change["objectId"], change["pageId"] = "shape", in.ShapeID, t.PageID
	}
	if *in.Value == "" {
		change["value"] = nil
	} else {
		change["value"] = *in.Value
		out.Value = in.Value
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{change}); err != nil {
		return pluginDataOut{}, err
	}
	return out, nil
}
