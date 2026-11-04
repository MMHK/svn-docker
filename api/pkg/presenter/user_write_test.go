package presenter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/store/memory"
)

// failingRoles is a RoleWriter that always fails, used to prove the delete
// cascade reports the step it stopped at.
type failingRoles struct {
	*memory.Store
	err error
}

func (f failingRoles) SetRoles(context.Context, string, []model.Role, model.WriteOptions) error {
	return f.err
}

func (f failingRoles) AddAccessPaths(context.Context, string, []string, model.WriteOptions) error {
	return f.err
}

func (f failingRoles) RemoveAccessPaths(context.Context, string, []string, model.WriteOptions) error {
	return f.err
}

func (f failingRoles) RemoveUser(context.Context, string, model.WriteOptions) error {
	return f.err
}

func TestDeleteCascadeReportsFailedStep(t *testing.T) {
	store := memory.New()
	tokens := auth.NewManager(auth.Options{TTL: time.Hour, Idle: time.Hour})

	presenter := NewUser(UserDeps{
		Users:      store,
		Accounts:   store,
		Roles:      store,
		RoleWriter: failingRoles{Store: store, err: errors.New("role file is read-only")},
		Groups:     store,
		Authz:      store,
		Tokens:     tokens,
	})

	err := presenter.Delete(context.Background(), "alice")
	if err == nil {
		t.Fatal("Delete() error = nil, want an operation error")
	}

	var operation *model.OperationError
	if !errors.As(err, &operation) {
		t.Fatalf("error = %T %v, want *model.OperationError", err, err)
	}

	if operation.Operation != "user.delete" || operation.Step != "roles" {
		t.Errorf("operation/step = %q/%q", operation.Operation, operation.Step)
	}

	if len(operation.Completed) != 1 || operation.Completed[0] != "authz" {
		t.Errorf("completed = %v, want [authz]", operation.Completed)
	}

	// The safe message names the steps and never leaks a path or a hash.
	message := operation.SafeMessage()
	if !strings.Contains(message, "roles") || !strings.Contains(message, "authz") {
		t.Errorf("SafeMessage() = %q", message)
	}

	if strings.Contains(message, "/") || strings.Contains(message, "apr1") {
		t.Errorf("SafeMessage() leaks internals: %q", message)
	}

	// The cascade stopped before touching the credentials file, so alice can
	// still authenticate. This is the documented "partially updated" state.
	if err := store.VerifyPassword(context.Background(), "alice", "testpass"); err != nil {
		t.Errorf("VerifyPassword(alice) error = %v, want the account to still exist", err)
	}
}

func TestDeleteRevokesTokens(t *testing.T) {
	store := memory.New()
	tokens := auth.NewManager(auth.Options{TTL: time.Hour, Idle: time.Hour})

	presenter := NewUser(UserDeps{
		Users:      store,
		Accounts:   store,
		Roles:      store,
		RoleWriter: store,
		Groups:     store,
		Authz:      store,
		Tokens:     tokens,
	})

	token, _, err := tokens.Issue("alice", []model.Role{model.RoleUser})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if err := presenter.Delete(context.Background(), "alice"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := tokens.Validate(token); err == nil {
		t.Error("the deleted account's token is still valid")
	}

	if _, err := store.GetUser(context.Background(), "alice"); !model.IsNotFound(err) {
		t.Errorf("GetUser(alice) error = %v, want NotFoundError", err)
	}
}

func TestChangePasswordRevokesTokens(t *testing.T) {
	store := memory.New()
	tokens := auth.NewManager(auth.Options{TTL: time.Hour, Idle: time.Hour})

	presenter := NewUser(UserDeps{
		Users:      store,
		Accounts:   store,
		Roles:      store,
		RoleWriter: store,
		Groups:     store,
		Authz:      store,
		Tokens:     tokens,
	})

	token, _, err := tokens.Issue("alice", []model.Role{model.RoleUser})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if err := presenter.ChangePassword(context.Background(), "alice", "brand-new", model.WriteOptions{}); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	if _, err := tokens.Validate(token); err == nil {
		t.Error("the token survived the password change")
	}

	if err := store.VerifyPassword(context.Background(), "alice", "brand-new"); err != nil {
		t.Errorf("VerifyPassword(new) error = %v", err)
	}

	if err := store.VerifyPassword(context.Background(), "alice", "testpass"); !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("VerifyPassword(old) error = %v, want ErrInvalidCredentials", err)
	}

	if err := presenter.ChangePassword(context.Background(), "nobody", "x", model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("ChangePassword(unknown) error = %v, want NotFoundError", err)
	}

	if err := presenter.ChangePassword(context.Background(), "alice", "", model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("ChangePassword(empty) error = %v, want ValidationError", err)
	}
}

func TestSetRolesAndGroupsValidate(t *testing.T) {
	presenter := newUserPresenter()
	ctx := context.Background()

	if _, err := presenter.SetRoles(ctx, "alice", []model.Role{"Nope"}, model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("SetRoles(unknown) error = %v, want ValidationError", err)
	}

	if _, err := presenter.SetRoles(ctx, "nobody", nil, model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("SetRoles(unknown user) error = %v, want NotFoundError", err)
	}

	if _, err := presenter.SetGroups(ctx, "alice", []string{"missing"}, model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("SetGroups(unknown group) error = %v, want ValidationError", err)
	}

	roles, err := ParseRoles([]string{"user", "Repository-Creator", "User"})
	if err != nil {
		t.Fatalf("ParseRoles() error = %v", err)
	}

	if len(roles) != 2 || roles[0] != model.RoleUser || roles[1] != model.RoleRepositoryCreator {
		t.Errorf("ParseRoles() = %v, want canonical, de-duplicated roles", roles)
	}

	if _, err := ParseRoles([]string{"Nope"}); !model.IsValidation(err) {
		t.Errorf("ParseRoles(unknown) error = %v, want ValidationError", err)
	}
}

func TestCreateValidatesNameAndPassword(t *testing.T) {
	presenter := newUserPresenter()
	ctx := context.Background()

	if _, err := presenter.Create(ctx, "a b", "password"); !model.IsValidation(err) {
		t.Errorf("Create(bad name) error = %v", err)
	}

	if _, err := presenter.Create(ctx, "dave", ""); !model.IsValidation(err) {
		t.Errorf("Create(empty password) error = %v", err)
	}

	created, err := presenter.Create(ctx, "dave", "dave-secret")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Name != "dave" {
		t.Errorf("created = %+v", created)
	}

	if _, err := presenter.Create(ctx, "dave", "other"); !model.IsConflict(err) {
		t.Errorf("Create(duplicate) error = %v, want ConflictError", err)
	}
}
