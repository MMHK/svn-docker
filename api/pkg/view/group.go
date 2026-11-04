package view

// Group is the public group representation.
type Group struct {
	Name        string   `json:"name" example:"devs"`
	Description string   `json:"description,omitempty" example:"Development team"`
	Members     []string `json:"members" example:"alice,bob"`
}

// GroupDetail adds the access paths that reference the group.
type GroupDetail struct {
	Group
	AccessPaths []string `json:"accessPaths" example:"project:/trunk"`
}

// GroupList is the GET /groups payload.
type GroupList struct {
	Items []Group `json:"items"`
	ListMeta
}

// CreateGroupRequest is the POST /groups body.
type CreateGroupRequest struct {
	Name string `json:"name" example:"devs"`
}

// GroupMembers is the GET/PUT /groups/{name}/members payload.
type GroupMembers struct {
	Name  string   `json:"name" example:"devs"`
	Users []string `json:"users" example:"alice,bob"`
}

// GroupMembersRequest replaces the whole member list of a group.
type GroupMembersRequest struct {
	Users []string `json:"users" example:"alice,bob"`
}

// GroupAccessPaths is the GET /groups/{name}/access-paths payload.
type GroupAccessPaths struct {
	Name        string   `json:"name" example:"devs"`
	AccessPaths []string `json:"accessPaths" example:"project:/trunk"`
}
