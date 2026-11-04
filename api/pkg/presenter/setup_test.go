package presenter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/store/htpasswd"
)

const setupTestToken = "test-bootstrap-token"

type setupFixture struct {
	dir        string
	authFile   string
	accessFile string
	accounts   *htpasswd.Store
}

// newSetupFixture points every bootstrap path into a fresh directory; with
// create=false none of the files exist yet.
func newSetupFixture(t *testing.T, create bool) setupFixture {
	t.Helper()

	dir := t.TempDir()

	fixture := setupFixture{
		dir:        dir,
		authFile:   filepath.Join(dir, "svn-auth-file"),
		accessFile: filepath.Join(dir, "svn-access-file"),
		accounts:   htpasswd.New(filepath.Join(dir, "svn-auth-file")),
	}

	if !create {
		return fixture
	}

	for _, file := range []string{fixture.authFile, fixture.accessFile} {
		if err := os.WriteFile(file, nil, 0o640); err != nil {
			t.Fatalf("create %s: %v", file, err)
		}
	}

	return fixture
}

func (f setupFixture) presenter() *Setup {
	return NewSetup(SetupDeps{
		Users:          f.accounts,
		Accounts:       f.accounts,
		AuthFile:       f.authFile,
		AccessFile:     f.accessFile,
		BootstrapToken: setupTestToken,
	})
}

func (f setupFixture) writeAuth(t *testing.T, content string) {
	t.Helper()

	if err := os.WriteFile(f.authFile, []byte(content), 0o640); err != nil {
		t.Fatalf("write auth file: %v", err)
	}
}

func TestSetupCreatesFirstUserFromEmptyDeployment(t *testing.T) {
	fixture := newSetupFixture(t, false)
	setup := fixture.presenter()
	ctx := context.Background()

	result, err := setup.Admin(ctx, "alice", "Passw0rd!", setupTestToken)
	if err != nil {
		t.Fatalf("Admin() error = %v", err)
	}

	if !result.AccountCreated {
		t.Error("AccountCreated = false, want true")
	}

	want := []string{"authFile", "accessFile"}
	if len(result.InitializedFiles) != len(want) {
		t.Fatalf("InitializedFiles = %v, want %v", result.InitializedFiles, want)
	}

	for index, name := range want {
		if result.InitializedFiles[index] != name {
			t.Errorf("InitializedFiles[%d] = %q, want %q", index, result.InitializedFiles[index], name)
		}
	}

	for _, path := range []string{fixture.authFile, fixture.accessFile} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}

		if os.PathSeparator == '/' && info.Mode().Perm() != 0o640 {
			t.Errorf("%s mode = %o, want 640", path, info.Mode().Perm())
		}
	}

	if info, err := os.Stat(fixture.accessFile); err != nil || info.Size() != 0 {
		t.Errorf("access file = %+v, %v; want an empty file", info, err)
	}

	if err := fixture.accounts.VerifyPassword(ctx, "alice", "Passw0rd!"); err != nil {
		t.Errorf("VerifyPassword() error = %v", err)
	}

	status := setup.Status(ctx)
	if status.SetupRequired || status.Reason != "" {
		t.Errorf("status after setup = %+v, want setup complete", status)
	}

	if len(status.Users) != 0 {
		t.Errorf("Users = %v, want empty after setup", status.Users)
	}
}

func TestSetupRequiresPasswordForNewAccount(t *testing.T) {
	fixture := newSetupFixture(t, true)
	setup := fixture.presenter()

	_, err := setup.Admin(context.Background(), "alice", "", setupTestToken)
	if err == nil {
		t.Fatal("Admin() error = nil, want a validation error")
	}

	var validation *model.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want *model.ValidationError", err)
	}

	if validation.Field != "password" {
		t.Errorf("Field = %q, want password", validation.Field)
	}
}

func TestSetupRejectsSecondUser(t *testing.T) {
	fixture := newSetupFixture(t, true)

	hash, err := htpasswd.HashAPR1("original-password")
	if err != nil {
		t.Fatalf("HashAPR1() error = %v", err)
	}

	fixture.writeAuth(t, "bob:"+hash+"\n")

	setup := fixture.presenter()

	_, err = setup.Admin(context.Background(), "alice", "Passw0rd!", setupTestToken)
	if err == nil {
		t.Fatal("Admin() error = nil, want a conflict")
	}

	var conflict *model.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error = %v, want *model.ConflictError", err)
	}
}

func TestSetupRejectsBadBootstrapToken(t *testing.T) {
	fixture := newSetupFixture(t, false)
	setup := fixture.presenter()

	_, err := setup.Admin(context.Background(), "alice", "Passw0rd!", "wrong-token")
	if err == nil {
		t.Fatal("Admin() error = nil, want forbidden")
	}

	var forbidden *model.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("error = %v, want *model.ForbiddenError", err)
	}

	if _, statErr := os.Stat(fixture.authFile); !os.IsNotExist(statErr) {
		t.Error("a rejected bootstrap created the auth file")
	}
}

func TestSetupReportsStorageUnavailableWhenDirectoryIsMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist")

	setup := NewSetup(SetupDeps{
		Users:          htpasswd.New(filepath.Join(missing, "svn-auth-file")),
		Accounts:       htpasswd.New(filepath.Join(missing, "svn-auth-file")),
		AuthFile:       filepath.Join(missing, "svn-auth-file"),
		AccessFile:     filepath.Join(missing, "svn-access-file"),
		BootstrapToken: setupTestToken,
	})

	_, err := setup.Admin(context.Background(), "alice", "Passw0rd!", setupTestToken)
	if err == nil {
		t.Fatal("Admin() error = nil, want storage unavailable")
	}

	if !model.IsStorageUnavailable(err) {
		t.Fatalf("error = %v, want *model.StorageUnavailableError", err)
	}
}
