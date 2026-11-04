// Package auth implements opaque bearer session tokens, the static RBAC
// matrix and the login rate limiter. Tokens are never persisted: only their
// SHA-256 digest is kept in memory, which is what plan section 5.7 requires.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
)

// TokenPrefix marks a session token so it is recognisable in logs and support
// requests without revealing anything.
const TokenPrefix = "svna_"

// Defaults from plan section 5.7.
const (
	DefaultTTL        = 12 * time.Hour
	DefaultIdle       = 2 * time.Hour
	DefaultMaxPerUser = 5
)

// tokenBytes is the CSPRNG output size (32 bytes -> 43 base64url characters).
const tokenBytes = 32

// Session is the server-side view of one issued token.
type Session struct {
	Username      string
	Roles         []model.Role
	CreatedAt     time.Time
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
}

// Options configures a Manager. Zero values fall back to the plan defaults.
type Options struct {
	TTL        time.Duration
	Idle       time.Duration
	MaxPerUser int
	Now        func() time.Time
	Rand       io.Reader
}

type record struct {
	digest   [sha256.Size]byte
	session  Session
	lastSeen time.Time
}

// Manager keeps sessions in memory. A restart invalidates everything, which is
// the documented default (SVN_ADMIN_SESSION_STORE=file is out of scope).
type Manager struct {
	mu      sync.Mutex
	records []*record

	ttl        time.Duration
	idle       time.Duration
	maxPerUser int
	now        func() time.Time
	rand       io.Reader
}

// NewManager builds a token manager.
func NewManager(opts Options) *Manager {
	manager := &Manager{
		ttl:        opts.TTL,
		idle:       opts.Idle,
		maxPerUser: opts.MaxPerUser,
		now:        opts.Now,
		rand:       opts.Rand,
	}

	if manager.ttl <= 0 {
		manager.ttl = DefaultTTL
	}

	if manager.idle <= 0 {
		manager.idle = DefaultIdle
	}

	if manager.maxPerUser <= 0 {
		manager.maxPerUser = DefaultMaxPerUser
	}

	if manager.now == nil {
		manager.now = time.Now
	}

	if manager.rand == nil {
		manager.rand = rand.Reader
	}

	return manager
}

// Issue creates a new token for username. When the per-user limit is reached
// the oldest token is revoked first.
func (m *Manager) Issue(username string, roles []model.Role) (string, Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.issueLocked(username, roles, m.now())
}

func (m *Manager) issueLocked(username string, roles []model.Role, now time.Time) (string, Session, error) {
	m.pruneLocked(now)

	token, err := m.newToken()
	if err != nil {
		return "", Session{}, err
	}

	digest := sha256.Sum256([]byte(token))

	session := Session{
		Username:      username,
		Roles:         append([]model.Role(nil), roles...),
		CreatedAt:     now,
		ExpiresAt:     now.Add(m.ttl),
		IdleExpiresAt: now.Add(m.idle),
	}

	m.records = append(m.records, &record{digest: digest, session: session, lastSeen: now})

	for m.countForUser(username) > m.maxPerUser {
		m.revokeOldestLocked(username)
	}

	return token, copySession(session), nil
}

func (m *Manager) newToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := io.ReadFull(m.rand, buffer); err != nil {
		return "", err
	}

	return TokenPrefix + base64.RawURLEncoding.EncodeToString(buffer), nil
}

// Validate resolves token and slides its idle deadline.
func (m *Manager) Validate(token string) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.pruneLocked(now)

	index := m.findLocked(token)
	if index < 0 {
		return Session{}, &model.UnauthorizedError{Reason: "invalid or expired token"}
	}

	record := m.records[index]
	record.lastSeen = now
	record.session.IdleExpiresAt = now.Add(m.idle)

	return copySession(record.session), nil
}

// Refresh rotates token: the old token is revoked before the new one is
// issued, so it can never be used twice.
func (m *Manager) Refresh(token string) (string, Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.pruneLocked(now)

	index := m.findLocked(token)
	if index < 0 {
		return "", Session{}, &model.UnauthorizedError{Reason: "invalid or expired token"}
	}

	previous := m.records[index]
	username := previous.session.Username
	roles := append([]model.Role(nil), previous.session.Roles...)

	m.records = append(m.records[:index], m.records[index+1:]...)

	return m.issueLocked(username, roles, now)
}

// Revoke removes token. Revoking an unknown token is a no-op.
func (m *Manager) Revoke(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := m.findLocked(token)
	if index >= 0 {
		m.records = append(m.records[:index], m.records[index+1:]...)
	}

	return nil
}

// RevokeUser removes every token belonging to username and returns the count.
// P2 wires this to password changes and user deletion.
func (m *Manager) RevokeUser(username string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept := m.records[:0]

	removed := 0

	for _, record := range m.records {
		if record.session.Username == username {
			removed++

			continue
		}

		kept = append(kept, record)
	}

	m.records = kept

	return removed
}

// Len reports the number of live sessions (used by tests and diagnostics).
func (m *Manager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneLocked(m.now())

	return len(m.records)
}

func (m *Manager) findLocked(token string) int {
	digest, ok := tokenDigest(token)
	if !ok {
		return -1
	}

	found := -1

	for i, record := range m.records {
		// Constant-time comparison for every record: no early exit.
		if subtle.ConstantTimeCompare(record.digest[:], digest[:]) == 1 {
			found = i
		}
	}

	return found
}

func (m *Manager) pruneLocked(now time.Time) {
	kept := m.records[:0]

	for _, record := range m.records {
		if !now.Before(record.session.ExpiresAt) || !now.Before(record.session.IdleExpiresAt) {
			continue
		}

		kept = append(kept, record)
	}

	m.records = kept
}

func (m *Manager) countForUser(username string) int {
	count := 0

	for _, record := range m.records {
		if record.session.Username == username {
			count++
		}
	}

	return count
}

func (m *Manager) revokeOldestLocked(username string) {
	for i, record := range m.records {
		if record.session.Username == username {
			m.records = append(m.records[:i], m.records[i+1:]...)

			return
		}
	}
}

func tokenDigest(token string) ([sha256.Size]byte, bool) {
	if !strings.HasPrefix(token, TokenPrefix) {
		return [sha256.Size]byte{}, false
	}

	encoded := token[len(TokenPrefix):]
	if len(encoded) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		return [sha256.Size]byte{}, false
	}

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(decoded) != tokenBytes {
		return [sha256.Size]byte{}, false
	}

	return sha256.Sum256([]byte(token)), true
}

func copySession(session Session) Session {
	out := session
	out.Roles = append([]model.Role(nil), session.Roles...)

	return out
}

// NewBootstrapToken returns a fresh one-time setup secret.
func NewBootstrapToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// ErrEmptyToken is returned when an Authorization header carries no token.
var ErrEmptyToken = errors.New("auth: empty bearer token")
