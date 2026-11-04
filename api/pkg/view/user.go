package view

import "time"

// User is the public user representation. Password hashes are never exposed.
type User struct {
	Name      string     `json:"name" example:"alice"`
	Comment   string     `json:"comment,omitempty" example:"Frontend team"`
	Groups    []string   `json:"groups" example:"devs"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// UserList is the GET /users payload.
type UserList struct {
	Items []User `json:"items"`
	ListMeta
}
