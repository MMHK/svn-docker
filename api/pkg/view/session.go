package view

import "time"

// SessionUser is the `me` object embedded in the session response.
type SessionUser struct {
	Name  string   `json:"name" example:"alice"`
	Roles []string `json:"roles" example:"Administrator"`
}

// Session is the POST /session, GET /session and POST /session/refresh
// payload. Token is omitted by GET /session and DELETE /session.
type Session struct {
	Token         string      `json:"token,omitempty" example:"svna_..."`
	ExpiresAt     time.Time   `json:"expiresAt"`
	IdleExpiresAt time.Time   `json:"idleExpiresAt"`
	Me            SessionUser `json:"me"`
	Roles         []string    `json:"roles" example:"Administrator"`
	Permissions   []string    `json:"permissions" example:"repository:view,user:view"`
}

// LoginRequest is the POST /session body.
type LoginRequest struct {
	Username string `json:"username" example:"alice"`
	Password string `json:"password" example:"secret"`
}

// SetupRequest is the POST /setup/admin body.
type SetupRequest struct {
	Username string `json:"username" example:"alice"`
	Token    string `json:"token" example:"bootstrap-token-from-stderr"`
}

// SetupResult is the POST /setup/admin success payload.
type SetupResult struct {
	Username string `json:"username" example:"alice"`
	Role     string `json:"role" example:"Administrator"`
}
