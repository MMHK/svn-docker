package view

import "time"

// Repository is the public repository representation.
type Repository struct {
	Name              string     `json:"name" example:"project"`
	Parent            string     `json:"parent" example:"/var/svnlocal"`
	URL               string     `json:"url,omitempty" example:"file:///var/svnlocal/project"`
	IsRepository      bool       `json:"isRepository" example:"true"`
	HeadRevision      int64      `json:"headRevision" example:"42"`
	LastChangedRev    int64      `json:"lastChangedRev" example:"42"`
	LastChangedAuthor string     `json:"lastChangedAuthor,omitempty" example:"mixmedia"`
	LastChangedAt     *time.Time `json:"lastChangedAt,omitempty"`
}

// RepositoryList is the GET /repos payload.
type RepositoryList struct {
	Items []Repository `json:"items"`
	ListMeta
}

// CreateRepositoryRequest is the POST /repos body.
type CreateRepositoryRequest struct {
	Name        string `json:"name" example:"project"`
	Structure   string `json:"structure,omitempty" example:"single" enums:"none,single,multi"`
	ProjectName string `json:"projectName,omitempty" example:"project"`
	AccessPath  bool   `json:"accessPath,omitempty" example:"true"`
}

// RepositoryCreated is the POST /repos 201 payload.
type RepositoryCreated struct {
	Name             string `json:"name" example:"project"`
	Parent           string `json:"parent" example:"/var/svnlocal"`
	URL              string `json:"url" example:"file:///var/svnlocal/project"`
	CreatedStructure string `json:"createdStructure" example:"single"`
}

// RepositoryEntry is one child in a tree listing.
type RepositoryEntry struct {
	Name     string     `json:"name" example:"trunk"`
	Kind     string     `json:"kind" example:"dir" enums:"dir,file"`
	Size     int64      `json:"size,omitempty" example:"1024"`
	Revision int64      `json:"revision" example:"42"`
	Author   string     `json:"author,omitempty" example:"mixmedia"`
	Date     *time.Time `json:"date,omitempty"`
}

// RepositoryTree is the GET /repos/{repo}/tree payload.
type RepositoryTree struct {
	Path     string            `json:"path" example:"trunk"`
	Revision int64             `json:"revision" example:"42"`
	Entries  []RepositoryEntry `json:"entries"`
	ListMeta
}
