package presenter

import (
	"time"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/store/memory"
)

// newUserPresenter wires the user presenter to the in-memory fakes.
func newUserPresenter() *User {
	store := memory.New()

	return NewUser(UserDeps{
		Users:      store,
		Accounts:   store,
		Roles:      store,
		RoleWriter: store,
		Groups:     store,
		Authz:      store,
		Tokens:     auth.NewManager(auth.Options{TTL: time.Hour, Idle: time.Hour}),
	})
}
