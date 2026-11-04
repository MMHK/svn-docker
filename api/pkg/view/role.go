package view

// Role is one fixed role and its description.
type Role struct {
	Name        string `json:"name" example:"Administrator"`
	Description string `json:"description" example:"Full control over the management API."`
}

// RoleList is the GET /roles payload.
type RoleList struct {
	Items []Role `json:"items"`
}
