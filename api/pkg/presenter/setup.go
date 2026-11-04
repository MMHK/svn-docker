package presenter

import (
	"context"
	"crypto/subtle"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// setupFileMode is the mode used for the files the bootstrap flow creates. It
// matches the mode the stores use for their own atomic writes.
const setupFileMode os.FileMode = 0o640

// setupUserListLimit bounds the account list returned to an unauthenticated
// caller by GET /setup/status.
const setupUserListLimit = 100

// SetupDeps are the ports and paths the bootstrap flow needs. It may create
// the auth and access files when a fresh deployment has none.
type SetupDeps struct {
	Users          model.UserStore
	Accounts       model.AccountWriter
	AuthFile       string
	AccessFile     string
	BootstrapToken string
}

// Setup serves the first-user bootstrap flow (plan D6, task T1.6). It replaces
// the v2 admin/admin backdoor: a one-time token, printed to stderr at startup
// when SVN_ADMIN_BOOTSTRAP_TOKEN is not set, is required.
type Setup struct {
	users      model.UserStore
	accounts   model.AccountWriter
	authFile   string
	accessFile string
	bootstrap  string
}

// NewSetup builds the setup presenter.
func NewSetup(deps SetupDeps) *Setup {
	return &Setup{
		users:      deps.Users,
		accounts:   deps.Accounts,
		authFile:   deps.AuthFile,
		accessFile: deps.AccessFile,
		bootstrap:  deps.BootstrapToken,
	}
}

// Required reports whether the deployment still needs its first account. It
// is used by the auth middleware to gate the rest of the API.
func (p *Setup) Required(ctx context.Context) (bool, error) {
	if p.users == nil {
		return true, nil
	}

	page, err := p.users.ListUsers(ctx, model.PageQuery{Page: 1, Size: 1})
	if err != nil {
		return false, err
	}

	return page.Total == 0, nil
}

// Status is the unauthenticated diagnostic behind GET /setup/status. It never
// carries an absolute path, a hash or a password.
func (p *Setup) Status(ctx context.Context) view.SetupStatus {
	authFile := inspectPath(p.authFile)
	accessFile := inspectPath(p.accessFile)

	status := view.SetupStatus{
		AuthFile: view.SetupAuthFileStatus{
			Exists:   authFile.Exists,
			Readable: authFile.Readable,
			Writable: authFile.Writable,
		},
		AccessFile: view.SetupFileStatus{
			Exists:   accessFile.Exists,
			Readable: accessFile.Readable,
			Writable: accessFile.Writable,
		},
		Users: []string{},
	}

	// A broken credential file must not turn the diagnostic into a 500: report
	// zero accounts instead of leaking the store error.
	if p.users != nil {
		page, err := p.users.ListUsers(ctx, model.PageQuery{Page: 1, Size: model.MaxPageSize})
		if err == nil {
			status.AuthFile.UserCount = page.Total

			names := make([]string, 0, len(page.Items))
			for _, user := range page.Items {
				names = append(names, user.Name)
			}

			sort.Strings(names)

			if len(names) > setupUserListLimit {
				names = names[:setupUserListLimit]
			}

			status.Users = names
		}
	}

	switch {
	case !authFile.Exists:
		status.Reason = "missing_auth_file"
	case !accessFile.Exists:
		status.Reason = "missing_access_file"
	case status.AuthFile.UserCount == 0:
		status.Reason = "no_users"
	default:
		status.Reason = ""
	}

	status.SetupRequired = status.Reason != ""
	status.CanCreateAccount = p.canCreateAccount(authFile)

	// Account names are only useful while setup is pending; once an account
	// exists the unauthenticated endpoint stays minimal.
	if !status.SetupRequired {
		status.Users = []string{}
	}

	return status
}

// Admin creates (or reuses) an account, provided the bootstrap token matches
// and no account exists yet.
func (p *Setup) Admin(ctx context.Context, username, password, token string) (view.SetupResult, error) {
	username = strings.TrimSpace(username)

	if p.bootstrap == "" || token == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(p.bootstrap)) != 1 {
		return view.SetupResult{}, &model.ForbiddenError{Reason: "invalid bootstrap token"}
	}

	if err := (model.User{Name: username}).Validate(); err != nil {
		return view.SetupResult{}, err
	}

	required, err := p.Required(ctx)
	if err != nil {
		return view.SetupResult{}, err
	}

	if !required {
		return view.SetupResult{}, &model.ConflictError{
			Kind:   model.KindUser,
			Name:   username,
			Reason: "a user already exists",
		}
	}

	initialized, err := p.ensureFiles()
	if err != nil {
		return view.SetupResult{}, err
	}

	_, err = p.users.GetUser(ctx, username)
	exists := err == nil

	if err != nil && !model.IsNotFound(err) {
		return view.SetupResult{}, err
	}

	created := false

	if !exists {
		if strings.TrimSpace(password) == "" {
			return view.SetupResult{}, &model.ValidationError{
				Field:  "password",
				Reason: "password is required to create the first user",
			}
		}

		if err := model.ValidatePassword(password); err != nil {
			return view.SetupResult{}, err
		}

		if err := p.accounts.CreateAccount(ctx, username, password, model.WriteOptions{}); err != nil {
			return view.SetupResult{}, err
		}

		created = true
	}

	return view.SetupResult{
		Username:         username,
		AccountCreated:   created,
		InitializedFiles: initialized,
	}, nil
}

// ensureFiles creates the auth and access files when they are missing. It
// returns the names of the files it actually created.
func (p *Setup) ensureFiles() ([]string, error) {
	targets := []struct {
		name string
		path string
	}{
		{name: "authFile", path: p.authFile},
		{name: "accessFile", path: p.accessFile},
	}

	created := make([]string, 0, len(targets))

	for _, target := range targets {
		if strings.TrimSpace(target.path) == "" {
			return nil, &model.StorageUnavailableError{
				Reason: target.name + " path is not configured",
			}
		}

		didCreate, err := fsx.EnsureFile(target.path, setupFileMode)
		if err != nil {
			return nil, &model.StorageUnavailableError{
				Reason: storageReason(target.name, err),
			}
		}

		if didCreate {
			created = append(created, target.name)
		}
	}

	return created, nil
}

// storageReason turns a file-creation failure into a message that names the
// resource but never an absolute path.
func storageReason(name string, err error) string {
	resource := strings.TrimSuffix(name, "File") + " file"

	if errors.Is(err, os.ErrNotExist) {
		return resource + " directory does not exist"
	}

	return resource + " directory is not writable"
}

// canCreateAccount reports whether the bootstrap flow can create the
// credential file: either it is writable now, or its directory lets us create
// it.
func (p *Setup) canCreateAccount(authFile view.PathInfo) bool {
	if authFile.Exists {
		return authFile.Writable
	}

	if strings.TrimSpace(p.authFile) == "" {
		return false
	}

	dir := filepath.Dir(p.authFile)

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}

	return writable(dir, true)
}
