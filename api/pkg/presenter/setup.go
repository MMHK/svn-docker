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
	Admins         model.AdminStore
	AuthFile       string
	AccessFile     string
	AdminFile      string
	BootstrapToken string
}

// Setup serves the first-user bootstrap flow (plan D6, task T1.6). It replaces
// the v2 admin/admin backdoor: a one-time token, printed to stderr at startup
// when SVN_ADMIN_BOOTSTRAP_TOKEN is not set, is required.
type Setup struct {
	users      model.UserStore
	accounts   model.AccountWriter
	admins     model.AdminStore
	authFile   string
	accessFile string
	adminFile  string
	bootstrap  string
}

// NewSetup builds the setup presenter.
func NewSetup(deps SetupDeps) *Setup {
	return &Setup{
		users:      deps.Users,
		accounts:   deps.Accounts,
		admins:     deps.Admins,
		authFile:   deps.AuthFile,
		accessFile: deps.AccessFile,
		adminFile:  deps.AdminFile,
		bootstrap:  deps.BootstrapToken,
	}
}

// Required reports whether the deployment still needs its first account or its
// administrator marker. It is used by the auth middleware to gate the rest of
// the API.
func (p *Setup) Required(ctx context.Context) (bool, error) {
	if p.users == nil {
		return true, nil
	}

	page, err := p.users.ListUsers(ctx, model.PageQuery{Page: 1, Size: 1})
	if err != nil {
		return false, err
	}

	if page.Total == 0 {
		return true, nil
	}

	admin, err := p.currentAdmin(ctx)
	if err != nil {
		return false, err
	}

	return admin == "", nil
}

// hasUsers reports whether the credential file already has an account.
func (p *Setup) hasUsers(ctx context.Context) (bool, error) {
	if p.users == nil {
		return false, nil
	}

	page, err := p.users.ListUsers(ctx, model.PageQuery{Page: 1, Size: 1})
	if err != nil {
		return false, err
	}

	return page.Total > 0, nil
}

// currentAdmin returns the configured administrator, or "" when none is set.
func (p *Setup) currentAdmin(ctx context.Context) (string, error) {
	if p.admins == nil {
		return "", nil
	}

	return p.admins.Admin(ctx)
}

// adminName is currentAdmin for the unauthenticated diagnostic: a read error
// is reported as "no administrator" so GET /setup/status never turns into a
// 500.
func (p *Setup) adminName(ctx context.Context) string {
	name, err := p.currentAdmin(ctx)
	if err != nil {
		return ""
	}

	return name
}

// setAdmin writes the marker, translating filesystem failures into the same
// 503 the auth/access file flow uses.
func (p *Setup) setAdmin(ctx context.Context, name string) error {
	if p.admins == nil {
		return &model.StorageUnavailableError{Reason: "admin path is not configured"}
	}

	if err := p.admins.SetAdmin(ctx, name); err != nil {
		if model.IsValidation(err) || model.IsStorageUnavailable(err) {
			return err
		}

		return &model.StorageUnavailableError{Reason: storageReason("adminFile", err)}
	}

	return nil
}

// Status is the unauthenticated diagnostic behind GET /setup/status. It never
// carries an absolute path, a hash or a password.
func (p *Setup) Status(ctx context.Context) view.SetupStatus {
	authFile := inspectPath(p.authFile)
	accessFile := inspectPath(p.accessFile)
	adminFile := inspectPath(p.adminFile)

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
		AdminFile: view.SetupFileStatus{
			Exists:   adminFile.Exists,
			Readable: adminFile.Readable,
			Writable: adminFile.Writable,
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
	case p.adminName(ctx) == "":
		status.Reason = "no_admin"
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

// Admin creates (or reuses) an account and records it as the single svnadmin
// administrator, provided the bootstrap token matches.
func (p *Setup) Admin(ctx context.Context, username, password, token string) (view.SetupResult, error) {
	username = strings.TrimSpace(username)

	if p.bootstrap == "" || token == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(p.bootstrap)) != 1 {
		return view.SetupResult{}, &model.ForbiddenError{Reason: "invalid bootstrap token"}
	}

	if err := (model.User{Name: username}).Validate(); err != nil {
		return view.SetupResult{}, err
	}

	hasUsers, err := p.hasUsers(ctx)
	if err != nil {
		return view.SetupResult{}, err
	}

	admin, err := p.currentAdmin(ctx)
	if err != nil {
		return view.SetupResult{}, err
	}

	// A first-run deployment may always rewrite the marker. Once accounts
	// exist, the marker can only be filled in when it is still empty; moving
	// the administrator afterwards is a manual file edit (plan D29, D31).
	if hasUsers && admin != "" {
		return view.SetupResult{}, &model.ConflictError{
			Kind:   model.KindUser,
			Name:   username,
			Reason: "an administrator already exists",
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

	if err := p.setAdmin(ctx, username); err != nil {
		return view.SetupResult{}, err
	}

	return view.SetupResult{
		Username:         username,
		AccountCreated:   created,
		InitializedFiles: initialized,
	}, nil
}

// ensureFiles creates the auth, access and admin files when they are missing.
// It returns the names of the files it actually created.
func (p *Setup) ensureFiles() ([]string, error) {
	targets := []struct {
		name string
		path string
	}{
		{name: "authFile", path: p.authFile},
		{name: "accessFile", path: p.accessFile},
		{name: "adminFile", path: p.adminFile},
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
