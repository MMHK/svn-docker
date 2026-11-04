package view

// SetupRequest is the POST /setup/admin body. Password is only used when the
// account does not exist yet: the first user is created with it. When the
// account already exists the password is ignored and the existing credential
// is left untouched.
type SetupRequest struct {
	Username string `json:"username" example:"alice"`
	Password string `json:"password" example:"Passw0rd!"`
	Token    string `json:"token" example:"bootstrap-token-from-stderr"`
}

// SetupResult is the POST /setup/admin success payload. InitializedFiles lists
// the files this call actually created (authFile, accessFile).
type SetupResult struct {
	Username         string   `json:"username" example:"alice"`
	AccountCreated   bool     `json:"accountCreated" example:"true"`
	InitializedFiles []string `json:"initializedFiles" example:"authFile,accessFile"`
}

// SetupFileStatus is the state of one bootstrap-related file. It never carries
// a path, content or hash.
type SetupFileStatus struct {
	Exists   bool `json:"exists" example:"true"`
	Readable bool `json:"readable" example:"true"`
	Writable bool `json:"writable" example:"true"`
}

// SetupAuthFileStatus adds the account count to the credential file state.
type SetupAuthFileStatus struct {
	Exists    bool `json:"exists" example:"true"`
	Readable  bool `json:"readable" example:"true"`
	Writable  bool `json:"writable" example:"true"`
	UserCount int  `json:"userCount" example:"3"`
}

// SetupStatus is the GET /setup/status payload. Reason is one of
// "missing_auth_file", "missing_access_file", "no_users" or "" (setup done).
type SetupStatus struct {
	SetupRequired    bool                `json:"setupRequired" example:"true"`
	Reason           string              `json:"reason" example:"no_users"`
	AuthFile         SetupAuthFileStatus `json:"authFile"`
	AccessFile       SetupFileStatus     `json:"accessFile"`
	Users            []string            `json:"users"`
	CanCreateAccount bool                `json:"canCreateAccount" example:"true"`
}
