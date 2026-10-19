package penpot

import "encoding/json"

// Profile is the subset of get-profile this server reads.
type Profile struct {
	ID               string `json:"id"`
	Fullname         string `json:"fullname"`
	Email            string `json:"email"`
	DefaultTeamID    string `json:"defaultTeamId"`
	DefaultProjectID string `json:"defaultProjectId"`
}

// Team is one row of get-teams.
type Team struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

// Project is one row of get-projects or get-all-projects.
type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TeamID     string `json:"teamId"`
	TeamName   string `json:"teamName,omitempty"`
	IsDefault  bool   `json:"isDefault"`
	ModifiedAt string `json:"modifiedAt,omitempty"`
}

// FileSummary is one row of get-project-files.
type FileSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProjectID  string `json:"projectId"`
	IsShared   bool   `json:"isShared"`
	ModifiedAt string `json:"modifiedAt,omitempty"`
}

// Permissions is get-file's permissions block.
type Permissions struct {
	CanEdit bool `json:"canEdit"`
	CanRead bool `json:"canRead"`
}

// File is the part of get-file this server reads.
type File struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ProjectID   string      `json:"projectId"`
	Revn        int64       `json:"revn"`
	Vern        int64       `json:"vern"`
	Permissions Permissions `json:"permissions"`
	Data        struct {
		Pages      []string            `json:"pages"`
		PagesIndex map[string]PageMeta `json:"pagesIndex"`
	} `json:"data"`
}

// PageMeta is a page's id and name.
type PageMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Page is get-page's reply.
type Page struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Objects map[string]Object `json:"objects"`
}

// Object is the part of a shape the tree view needs.
type Object struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	X        *float64 `json:"x"`
	Y        *float64 `json:"y"`
	Width    *float64 `json:"width"`
	Height   *float64 `json:"height"`
	ParentID string   `json:"parentId"`
	FrameID  string   `json:"frameId"`
	Shapes   []string `json:"shapes,omitempty"`
}

// UpdateResult is update-file's reply. Revn is the value before the update.
type UpdateResult struct {
	Revn   int64             `json:"revn"`
	Lagged []json.RawMessage `json:"lagged"`
}
