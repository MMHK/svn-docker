package view

// ACLEntry is one principal/permission pair on an access path.
type ACLEntry struct {
	Principal  string `json:"principal" example:"@devs"`
	Permission string `json:"permission" example:"rw" enums:"none,r,rw"`
}

// AccessPath is the public access-path representation.
type AccessPath struct {
	Path string     `json:"path" example:"project:/trunk"`
	ACL  []ACLEntry `json:"acl"`
}

// AccessPathList is the GET /access-paths payload.
type AccessPathList struct {
	Items []AccessPath `json:"items"`
	ListMeta
}

// CreateAccessPathRequest is the POST /access-paths body.
type CreateAccessPathRequest struct {
	Path string `json:"path" example:"project:/trunk"`
}

// SetACLPermissionRequest is the PUT /access-paths/{path}/acl/{principal} body.
type SetACLPermissionRequest struct {
	Permission string `json:"permission" example:"rw" enums:"none,r,rw"`
}

// UserAccessPath is one effective access source of an account. inherited is
// true for a `@group` or `*` entry and false for a direct user entry.
type UserAccessPath struct {
	Path       string `json:"path" example:"project:/trunk"`
	Permission string `json:"permission" example:"rw" enums:"none,r,rw"`
	Inherited  bool   `json:"inherited" example:"true"`
	Source     string `json:"source" example:"group" enums:"user,group,global"`
	Group      string `json:"group,omitempty" example:"devs"`
}

// UserAccessPaths is the GET /users/{name}/access-paths payload.
type UserAccessPaths struct {
	Name  string           `json:"name" example:"alice"`
	Items []UserAccessPath `json:"items"`
}
