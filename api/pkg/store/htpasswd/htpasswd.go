// Package htpasswd parses Apache htpasswd files and verifies the password
// hash formats that Apache mod_authn_file understands. Reads are stateless;
// writes are lossless, locked, atomic and always use APR1 (plan sections 5.2
// and 5.4).
package htpasswd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// Entry is one parsed `user:hash` record.
type Entry struct {
	User string
	Hash string
	Line int
}

// Line is one raw physical line. Raw excludes the line ending and is kept
// verbatim so the P2 writer can rewrite a single record without disturbing
// comments, ordering or CRLF.
type Line struct {
	Raw   string
	EOL   string
	Entry *Entry
}

// File is a parsed htpasswd file.
type File struct {
	Path  string
	Lines []Line

	entries []Entry
	users   []string
	index   map[string][]int
}

// Error is the typed failure returned by the store. It never carries a hash.
type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("htpasswd: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Verify checks password against a single htpasswd hash. It returns
// model.ErrInvalidCredentials on mismatch and a descriptive error for
// unsupported formats (notably plaintext).
func Verify(password, hash string) error {
	switch {
	case strings.HasPrefix(hash, "$apr1$"):
		return verifyMD5Crypt(password, hash, "$apr1$")
	case strings.HasPrefix(hash, "$1$"):
		return verifyMD5Crypt(password, hash, "$1$")
	case strings.HasPrefix(hash, "$5$"):
		return verifySHACrypt(password, hash, sha256Spec)
	case strings.HasPrefix(hash, "$6$"):
		return verifySHACrypt(password, hash, sha512Spec)
	case len(hash) >= 5 && strings.EqualFold(hash[:5], "{SHA}"):
		return verifySHA1(password, hash[5:])
	case strings.HasPrefix(hash, "$2a$"), strings.HasPrefix(hash, "$2b$"), strings.HasPrefix(hash, "$2y$"):
		return verifyBcrypt(password, hash)
	case isDESHash(hash):
		return verifyDES(password, hash)
	case !strings.HasPrefix(hash, "$") && !strings.HasPrefix(hash, "{") && len(hash) != 13:
		// A bare value that is not a known scheme is treated as plaintext.
		return errors.New("htpasswd: plaintext passwords are not accepted")
	default:
		return errors.New("htpasswd: unsupported password hash format")
	}
}

func isDESHash(hash string) bool {
	if len(hash) != 13 {
		return false
	}

	if strings.HasPrefix(hash, "$") || strings.HasPrefix(hash, "{") || hash[0] == '_' {
		return false
	}

	for i := 0; i < len(hash); i++ {
		if strings.IndexByte(cryptAlphabet, hash[i]) < 0 {
			return false
		}
	}

	return true
}

// Parse splits htpasswd content into raw lines plus parsed entries. Empty
// lines and comments are preserved but never indexed.
func Parse(path string, data []byte) *File {
	file := &File{
		Path:  path,
		index: make(map[string][]int),
	}

	text := string(data)
	for len(text) > 0 {
		var raw, eol string

		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			raw, text = text[:idx], text[idx+1:]
			eol = "\n"
			if strings.HasSuffix(raw, "\r") {
				raw = raw[:len(raw)-1]
				eol = "\r\n"
			}
		} else {
			raw, text = text, ""
		}

		line := Line{Raw: raw, EOL: eol}
		number := len(file.Lines) + 1

		trimmed := strings.TrimSpace(raw)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if user, hash, ok := strings.Cut(trimmed, ":"); ok {
				user = strings.TrimSpace(user)
				hash = strings.TrimSpace(hash)
				if user != "" && hash != "" {
					entry := Entry{User: user, Hash: hash, Line: number}
					file.entries = append(file.entries, entry)
					line.Entry = &entry

					if _, seen := file.index[user]; !seen {
						file.users = append(file.users, user)
					}

					file.index[user] = append(file.index[user], len(file.entries)-1)
				}
			}
		}

		file.Lines = append(file.Lines, line)
	}

	return file
}

// Users returns the unique account names in file order.
func (f *File) Users() []string {
	out := make([]string, len(f.users))
	copy(out, f.users)

	return out
}

// Entries returns every parsed record in file order.
func (f *File) Entries() []Entry {
	out := make([]Entry, len(f.entries))
	copy(out, f.entries)

	return out
}

// EntriesFor returns the records for one account, in file order. Duplicate
// records are kept: both are tried during verification.
func (f *File) EntriesFor(user string) []Entry {
	indices := f.index[user]
	out := make([]Entry, 0, len(indices))

	for _, i := range indices {
		out = append(out, f.entries[i])
	}

	return out
}

// Store is an htpasswd-backed user store. Reads are stateless; writes are
// serialised in-process and across processes (flock/LockFileEx) and always go
// through fsx.
type Store struct {
	path      string
	backupDir string
	keep      int

	writeMu sync.Mutex
}

// New builds a store for path. The file is read on demand so external edits
// (the v2 tool, a shell script) are visible without a restart.
func New(path string, opts ...Option) *Store {
	store := &Store{path: path, keep: fsx.DefaultBackupKeep}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// BackupDir returns the configured backup directory (empty disables backups).
func (s *Store) BackupDir() string {
	return s.backupDir
}

// Path returns the configured file path.
func (s *Store) Path() string {
	return s.path
}

// Load reads and parses the file. A missing or unreadable file is reported as
// a typed *Error, never as a panic.
func (s *Store) Load() (*File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, &Error{Op: "read", Path: s.path, Err: err}
	}

	return Parse(s.path, data), nil
}

// ListUsers implements model.UserStore. Hashes are never exposed.
func (s *Store) ListUsers(_ context.Context, query model.PageQuery) (model.Page[model.User], error) {
	file, err := s.Load()
	if err != nil {
		return model.Page[model.User]{}, err
	}

	all := make([]model.User, 0, len(file.users))
	for _, user := range file.users {
		all = append(all, model.User{Name: user})
	}

	normalized := query.Normalize()
	needle := strings.ToLower(normalized.Q)

	filtered := make([]model.User, 0, len(all))

	for _, user := range all {
		if needle == "" || strings.Contains(strings.ToLower(user.Name), needle) {
			filtered = append(filtered, user)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})

	total := len(filtered)
	start := (normalized.Page - 1) * normalized.Size
	if start > total {
		start = total
	}

	end := start + normalized.Size
	if end > total {
		end = total
	}

	return model.Page[model.User]{
		Items: filtered[start:end],
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// GetUser implements model.UserStore.
func (s *Store) GetUser(_ context.Context, name string) (model.User, error) {
	file, err := s.Load()
	if err != nil {
		return model.User{}, err
	}

	if len(file.EntriesFor(name)) == 0 {
		return model.User{}, &model.NotFoundError{Kind: model.KindUser, Name: name}
	}

	return model.User{Name: name}, nil
}

// VerifyPassword implements model.CredentialStore. It deliberately treats
// unknown users and wrong passwords identically.
func (s *Store) VerifyPassword(_ context.Context, username, password string) error {
	file, err := s.Load()
	if err != nil {
		return err
	}

	entries := file.EntriesFor(username)
	if len(entries) == 0 {
		return model.ErrInvalidCredentials
	}

	for _, entry := range entries {
		if verifyErr := Verify(password, entry.Hash); verifyErr == nil {
			return nil
		}
	}

	return model.ErrInvalidCredentials
}
