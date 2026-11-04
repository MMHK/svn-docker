package view

// CreateUserRequest is the POST /users body. The password is write-only: it is
// never echoed, stored in the domain model or logged.
type CreateUserRequest struct {
	Username string `json:"username" example:"alice"`
	Password string `json:"password" example:"correct horse battery staple"`
}

// ChangePasswordRequest is the PUT /users/{name}/password body.
type ChangePasswordRequest struct {
	Password string `json:"password" example:"new-passphrase"`
}

// UserGroups is the GET/PUT /users/{name}/groups payload.
type UserGroups struct {
	Name   string   `json:"name" example:"alice"`
	Groups []string `json:"groups" example:"devs"`
}

// UserGroupsRequest replaces the whole group membership of one account.
type UserGroupsRequest struct {
	Groups []string `json:"groups" example:"devs,ops"`
}
