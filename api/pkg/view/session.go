package view

import "time"

// SessionUser is the `me` object embedded in the session response.
type SessionUser struct {
	Name string `json:"name" example:"alice"`
}

// Session is the POST /session, GET /session and POST /session/refresh
// payload. Token is omitted by GET /session and DELETE /session.
type Session struct {
	Token         string      `json:"token,omitempty" example:"svna_..."`
	ExpiresAt     time.Time   `json:"expiresAt"`
	IdleExpiresAt time.Time   `json:"idleExpiresAt"`
	Me            SessionUser `json:"me"`
}

// LoginRequest is the POST /session body.
type LoginRequest struct {
	Username string `json:"username" example:"alice"`
	Password string `json:"password" example:"secret"`
}
