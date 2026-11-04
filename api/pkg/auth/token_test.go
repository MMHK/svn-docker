package auth

import (
	"strings"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func newTestManager(t *testing.T) (*Manager, *fakeClock) {
	t.Helper()

	clock := &fakeClock{now: time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)}
	manager := NewManager(Options{
		TTL:        12 * time.Hour,
		Idle:       2 * time.Hour,
		MaxPerUser: 5,
		Now:        clock.Now,
	})

	return manager, clock
}

func TestIssueAndValidate(t *testing.T) {
	manager, _ := newTestManager(t)

	token, session, err := manager.Issue("alice")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if !strings.HasPrefix(token, TokenPrefix) {
		t.Errorf("token %q does not carry the %q prefix", token, TokenPrefix)
	}

	if len(token) != len(TokenPrefix)+43 {
		t.Errorf("token length = %d, want %d", len(token), len(TokenPrefix)+43)
	}

	if session.Username != "alice" {
		t.Errorf("session = %+v", session)
	}

	validated, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if validated.Username != "alice" {
		t.Errorf("username = %q", validated.Username)
	}

	if manager.Len() != 1 {
		t.Errorf("Len() = %d, want 1", manager.Len())
	}
}

func TestValidateRejectsMalformedTokens(t *testing.T) {
	manager, _ := newTestManager(t)

	for _, token := range []string{
		"",
		"not-a-token",
		"svna_short",
		"svna_" + strings.Repeat("A", 43),
		"svna_" + strings.Repeat("!", 43),
	} {
		if _, err := manager.Validate(token); err == nil {
			t.Errorf("Validate(%q) error = nil, want unauthorized", token)
		}
	}
}

func TestAbsoluteExpiry(t *testing.T) {
	manager, clock := newTestManager(t)

	token, _, err := manager.Issue("alice")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	// Keep the session idle-alive while crossing the absolute TTL.
	for i := 0; i < 7; i++ {
		clock.Advance(90 * time.Minute)
		if _, err := manager.Validate(token); err != nil {
			t.Fatalf("Validate() at %v error = %v", time.Duration(i+1)*90*time.Minute, err)
		}
	}

	clock.Advance(2 * time.Hour)

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("Validate() after the 12h TTL error = nil, want unauthorized")
	}
}

func TestIdleExpiry(t *testing.T) {
	manager, clock := newTestManager(t)

	token, _, err := manager.Issue("alice")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	clock.Advance(90 * time.Minute)

	if _, err := manager.Validate(token); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// A successful request slides the idle deadline.
	clock.Advance(90 * time.Minute)

	if _, err := manager.Validate(token); err != nil {
		t.Fatalf("Validate() after slide error = %v", err)
	}

	clock.Advance(121 * time.Minute)

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("Validate() after idle timeout error = nil, want unauthorized")
	}
}

func TestRevoke(t *testing.T) {
	manager, _ := newTestManager(t)

	token, _, err := manager.Issue("alice")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if err := manager.Revoke(token); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("Validate() after Revoke() error = nil, want unauthorized")
	}

	if manager.Len() != 0 {
		t.Errorf("Len() = %d, want 0", manager.Len())
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	manager, _ := newTestManager(t)

	token, _, err := manager.Issue("alice")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	rotated, session, err := manager.Refresh(token)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if rotated == token {
		t.Fatal("Refresh() returned the same token")
	}

	if session.Username != "alice" {
		t.Errorf("refreshed session = %+v", session)
	}

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("old token still validates after refresh")
	}

	if _, err := manager.Validate(rotated); err != nil {
		t.Fatalf("new token does not validate: %v", err)
	}

	if manager.Len() != 1 {
		t.Errorf("Len() = %d, want 1", manager.Len())
	}
}

func TestMaxFiveTokensPerUser(t *testing.T) {
	manager, _ := newTestManager(t)

	tokens := make([]string, 0, 6)

	for i := 0; i < 6; i++ {
		token, _, err := manager.Issue("alice")
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}

		tokens = append(tokens, token)
	}

	if manager.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", manager.Len())
	}

	if _, err := manager.Validate(tokens[0]); err == nil {
		t.Error("oldest token still validates; it should have been revoked")
	}

	for _, token := range tokens[1:] {
		if _, err := manager.Validate(token); err != nil {
			t.Errorf("token was revoked unexpectedly: %v", err)
		}
	}

	// Another user has an independent budget.
	if _, _, err := manager.Issue("bob"); err != nil {
		t.Fatalf("Issue(bob) error = %v", err)
	}

	if manager.Len() != 6 {
		t.Errorf("Len() = %d, want 6", manager.Len())
	}
}

func TestRevokeUser(t *testing.T) {
	manager, _ := newTestManager(t)

	for i := 0; i < 3; i++ {
		if _, _, err := manager.Issue("alice"); err != nil {
			t.Fatalf("Issue() error = %v", err)
		}
	}

	if _, _, err := manager.Issue("bob"); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if removed := manager.RevokeUser("alice"); removed != 3 {
		t.Errorf("RevokeUser() = %d, want 3", removed)
	}

	if manager.Len() != 1 {
		t.Errorf("Len() = %d, want 1", manager.Len())
	}
}

func TestNewBootstrapToken(t *testing.T) {
	token, err := NewBootstrapToken()
	if err != nil {
		t.Fatalf("NewBootstrapToken() error = %v", err)
	}

	if len(token) != 43 {
		t.Errorf("length = %d, want 43", len(token))
	}
}
