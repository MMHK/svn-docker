package htpasswd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// Option customises a Store.
type Option func(*Store)

// WithBackupDir makes every write copy the previous file into dir first.
func WithBackupDir(dir string) Option {
	return func(store *Store) {
		store.backupDir = dir
	}
}

// WithBackupKeep sets the backup rotation depth (default 30).
func WithBackupKeep(keep int) Option {
	return func(store *Store) {
		if keep > 0 {
			store.keep = keep
		}
	}
}

var (
	_ model.UserStore       = (*Store)(nil)
	_ model.AccountWriter   = (*Store)(nil)
	_ model.CredentialStore = (*Store)(nil)
)

// Bytes reassembles the file. Untouched lines are emitted verbatim, including
// their original line endings, so a single-record edit leaves every other byte
// of the file identical (plan section 5.2).
func (f *File) Bytes() []byte {
	var builder strings.Builder

	for _, line := range f.Lines {
		builder.WriteString(line.Raw)
		builder.WriteString(line.EOL)
	}

	return []byte(builder.String())
}

// CreateAccount adds `name:APR1(password)` for a new account.
func (s *Store) CreateAccount(ctx context.Context, name, password string, opts model.WriteOptions) error {
	name = strings.TrimSpace(name)

	if err := model.ValidateUserName(name); err != nil {
		return err
	}

	if err := model.ValidatePassword(password); err != nil {
		return err
	}

	hash, err := HashAPR1(password)
	if err != nil {
		return &Error{Op: "hash", Path: s.path, Err: err}
	}

	// Refuse to write a hash we cannot verify: a broken hash would lock the
	// account out of every SVN client.
	if err := Verify(password, hash); err != nil {
		return &Error{Op: "hash", Path: s.path, Err: fmt.Errorf("generated APR1 hash does not verify")}
	}

	return s.mutate(ctx, name, opts, func(file *File) (bool, error) {
		if len(file.EntriesFor(name)) > 0 {
			return false, &model.ConflictError{
				Kind:   model.KindUser,
				Name:   name,
				Reason: "already exists",
			}
		}

		file.Lines = appendEntry(file.Lines, name, hash)

		return true, nil
	}, func(file *File) error {
		return verifyEntries(file, name, password)
	})
}

// SetPassword replaces the hash of an existing account.
func (s *Store) SetPassword(ctx context.Context, name, password string, opts model.WriteOptions) error {
	name = strings.TrimSpace(name)

	if err := model.ValidateUserName(name); err != nil {
		return err
	}

	if err := model.ValidatePassword(password); err != nil {
		return err
	}

	hash, err := HashAPR1(password)
	if err != nil {
		return &Error{Op: "hash", Path: s.path, Err: err}
	}

	if err := Verify(password, hash); err != nil {
		return &Error{Op: "hash", Path: s.path, Err: fmt.Errorf("generated APR1 hash does not verify")}
	}

	return s.mutate(ctx, name, opts, func(file *File) (bool, error) {
		lines := entryLineNumbers(file, name)
		if len(lines) == 0 {
			return false, &model.NotFoundError{Kind: model.KindUser, Name: name}
		}

		for _, number := range lines {
			line := &file.Lines[number-1]
			// Keep the original `user:` spelling and any surrounding padding;
			// only the hash changes.
			user, _, _ := strings.Cut(line.Raw, ":")
			line.Raw = user + ":" + hash
		}

		return true, nil
	}, func(file *File) error {
		return verifyEntries(file, name, password)
	})
}

// DeleteUser removes every line belonging to name. Other accounts, comments
// and line endings are preserved verbatim.
func (s *Store) DeleteUser(ctx context.Context, name string, opts model.WriteOptions) error {
	name = strings.TrimSpace(name)

	if err := model.ValidateUserName(name); err != nil {
		return err
	}

	return s.mutate(ctx, name, opts, func(file *File) (bool, error) {
		lines := entryLineNumbers(file, name)
		if len(lines) == 0 {
			return false, &model.NotFoundError{Kind: model.KindUser, Name: name}
		}

		remove := make(map[int]bool, len(lines))
		for _, number := range lines {
			remove[number] = true
		}

		kept := make([]Line, 0, len(file.Lines))
		for index, line := range file.Lines {
			if remove[index+1] {
				continue
			}

			kept = append(kept, line)
		}

		file.Lines = kept

		return true, nil
	}, func(file *File) error {
		if len(file.EntriesFor(name)) > 0 {
			return fmt.Errorf("account %q is still present after the delete", name)
		}

		return nil
	})
}

// mutate runs one locked, hash-checked, lossless read-modify-write cycle:
// lock -> read -> edit -> backup -> atomic write -> self-verify (with rollback).
func (s *Store) mutate(
	_ context.Context,
	name string,
	opts model.WriteOptions,
	edit func(*File) (bool, error),
	check func(*File) error,
) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	handle, err := fsx.Lock(s.path)
	if err != nil {
		return &Error{Op: "lock", Path: s.path, Err: err}
	}
	defer func() { _ = handle.Unlock() }()

	snapshot, err := fsx.ReadFile(s.path)
	if err != nil {
		return &Error{Op: "read", Path: s.path, Err: err}
	}

	if opts.ExpectedHash != "" && !strings.EqualFold(opts.ExpectedHash, snapshot.Hash) {
		return &model.ConflictError{
			Kind:   model.KindUser,
			Name:   name,
			Reason: "the credentials file changed since it was read",
		}
	}

	file := Parse(s.path, snapshot.Data)

	changed, err := edit(file)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	if _, err := fsx.Backup(s.path, s.backupDir, s.keep); err != nil {
		return &Error{Op: "backup", Path: s.path, Err: err}
	}

	if err := fsx.WriteAtomic(s.path, file.Bytes(), snapshot.Hash); err != nil {
		if fsx.IsConflict(err) {
			return &model.ConflictError{
				Kind:   model.KindUser,
				Name:   name,
				Reason: "the credentials file was modified by another writer",
			}
		}

		return &Error{Op: "write", Path: s.path, Err: err}
	}

	if check == nil {
		return nil
	}

	written, err := fsx.ReadFile(s.path)
	if err != nil {
		return &Error{Op: "verify", Path: s.path, Err: err}
	}

	if err := check(Parse(s.path, written.Data)); err != nil {
		// The file did not reach the state we asked for: put the previous
		// bytes back instead of leaving a broken credential file behind.
		if rollbackErr := fsx.WriteAtomic(s.path, snapshot.Data, ""); rollbackErr != nil {
			return &Error{
				Op:   "rollback",
				Path: s.path,
				Err:  fmt.Errorf("%v (rollback also failed: %w)", err, rollbackErr),
			}
		}

		return &Error{Op: "verify", Path: s.path, Err: err}
	}

	return nil
}

// verifyEntries asserts that password verifies against every record of name.
func verifyEntries(file *File, name, password string) error {
	entries := file.EntriesFor(name)
	if len(entries) == 0 {
		return fmt.Errorf("account %q is missing after the write", name)
	}

	for _, entry := range entries {
		if err := Verify(password, entry.Hash); err != nil {
			return fmt.Errorf("the written hash for %q does not verify", name)
		}
	}

	return nil
}

// entryLineNumbers returns the 1-based physical line numbers of the records
// belonging to name.
func entryLineNumbers(file *File, name string) []int {
	entries := file.EntriesFor(name)
	out := make([]int, 0, len(entries))

	for _, entry := range entries {
		out = append(out, entry.Line)
	}

	return out
}

// appendEntry adds a record at the end of the file, keeping the existing line
// ending style and adding a trailing newline only when the file lacks one.
func appendEntry(lines []Line, name, hash string) []Line {
	newline := newlineOf(lines)

	if len(lines) > 0 && lines[len(lines)-1].EOL == "" {
		lines[len(lines)-1].EOL = newline
	}

	return append(lines, Line{Raw: name + ":" + hash, EOL: newline})
}

func newlineOf(lines []Line) string {
	for _, line := range lines {
		if line.EOL != "" {
			return line.EOL
		}
	}

	return "\n"
}
