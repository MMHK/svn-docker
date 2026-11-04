package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// Option customises a Store.
type Option func(*Store)

// WithBackupDir copies the previous file into dir before every write.
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

var _ model.AuthzWriter = (*Store)(nil)

// block is one `[section]` of the authz file.
type block struct {
	name        string
	index       int
	lastContent int
	groups      bool
	entries     []int // indices of `key = value` lines
}

// blocks splits the file into section blocks while remembering which lines
// carry entries, so an edit can touch exactly the lines it must.
func blocks(lines []fsx.TextLine) []block {
	out := make([]block, 0, 4)

	for index, line := range lines {
		if name, ok := line.IsSectionHeader(); ok {
			out = append(out, block{
				name:        name,
				index:       index,
				lastContent: index,
				groups:      strings.EqualFold(name, GroupsSection),
			})

			continue
		}

		if len(out) == 0 || line.Blank() || line.Comment() {
			continue
		}

		current := &out[len(out)-1]

		if _, _, ok := line.SplitKeyValue(); !ok {
			continue
		}

		current.entries = append(current.entries, index)
		current.lastContent = index
	}

	return out
}

// SetGroupsOfUser makes username a member of exactly the given groups. Other
// members, comments, ordering and the rest of the authz file are preserved
// (plan T2.3, T3.2).
func (s *Store) SetGroupsOfUser(ctx context.Context, username string, groups []string, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	desired := make([]string, 0, len(groups))
	seen := make(map[string]bool, len(groups))

	for _, group := range groups {
		if err := model.ValidateGroupName(group); err != nil {
			return err
		}

		if seen[group] {
			continue
		}

		seen[group] = true
		desired = append(desired, group)
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		return setGroupsLines(lines, sections, username, desired)
	}, func(file *File) error {
		got := make(map[string]bool)

		for _, name := range file.Groups() {
			for _, member := range file.UsersOfGroup(name) {
				if member == username {
					got[name] = true
				}
			}
		}

		if len(got) != len(desired) {
			return fmt.Errorf("group membership for %q was not written as requested", username)
		}

		for _, group := range desired {
			if !got[group] {
				return fmt.Errorf("group membership for %q was not written as requested", username)
			}
		}

		return nil
	})
}

func setGroupsLines(
	lines []fsx.TextLine,
	sections []block,
	username string,
	desired []string,
) ([]fsx.TextLine, bool, error) {
	wanted := make(map[string]bool, len(desired))
	for _, group := range desired {
		wanted[group] = true
	}

	newline := fsx.Newline(lines)

	var groupsBlock *block

	for index := range sections {
		if sections[index].groups {
			groupsBlock = &sections[index]

			break
		}
	}

	deletions := make(map[int]bool)
	replacements := make(map[int][]fsx.TextLine)
	insertions := make(map[int][]fsx.TextLine)

	have := make(map[string]bool)

	if groupsBlock != nil {
		for _, index := range groupsBlock.entries {
			key, value, ok := lines[index].SplitKeyValue()
			if !ok || key == "" {
				continue
			}

			members := splitMembers(value)
			present := containsMember(members, username)

			if wanted[key] {
				have[key] = true

				if present {
					continue
				}

				members = append(members, username)
				replacements[index] = []fsx.TextLine{{
					Raw: key + " = " + strings.Join(members, separatorOf(value)),
					EOL: normaliseEOL(lines[index].EOL, newline),
				}}

				continue
			}

			if !present {
				continue
			}

			remaining := make([]string, 0, len(members))
			for _, member := range members {
				if member != username {
					remaining = append(remaining, member)
				}
			}

			if len(remaining) == 0 {
				// The group has no members left: drop the (now meaningless)
				// definition instead of leaving `devs = ` behind.
				deletions[index] = true

				continue
			}

			replacements[index] = []fsx.TextLine{{
				Raw: key + " = " + strings.Join(remaining, separatorOf(value)),
				EOL: normaliseEOL(lines[index].EOL, newline),
			}}
		}
	}

	added := make([]fsx.TextLine, 0, len(desired))
	for _, group := range desired {
		if have[group] {
			continue
		}

		added = append(added, fsx.TextLine{Raw: group + " = " + username, EOL: newline})
	}

	if len(deletions) == 0 && len(replacements) == 0 && len(added) == 0 {
		return lines, false, nil
	}

	if groupsBlock == nil {
		if len(added) == 0 {
			return lines, false, nil
		}

		lines = append([]fsx.TextLine(nil), fsx.EnsureTrailingNewline(append([]fsx.TextLine(nil), lines...))...)
		if len(lines) > 0 && !lines[len(lines)-1].Blank() {
			lines = append(lines, fsx.TextLine{EOL: newline})
		}

		lines = append(lines, fsx.TextLine{Raw: "[" + GroupsSection + "]", EOL: newline})
		lines = append(lines, added...)

		return lines, true, nil
	}

	if len(added) > 0 {
		anchor := groupsBlock.lastContent

		if lines[anchor].EOL == "" {
			lines = append([]fsx.TextLine(nil), lines...)
			lines[anchor].EOL = newline
		}

		insertions[anchor] = added
	}

	return applyEdits(lines, deletions, replacements, insertions), true, nil
}

// RemoveUser strips username from every group and from every access-path ACL.
// The rest of the file is preserved, which is what the cascade in plan T2.3
// needs before the htpasswd line is deleted.
func (s *Store) RemoveUser(ctx context.Context, username string, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		deletions := make(map[int]bool)
		replacements := make(map[int][]fsx.TextLine)

		newline := fsx.Newline(lines)

		for _, section := range sections {
			for _, index := range section.entries {
				key, value, ok := lines[index].SplitKeyValue()
				if !ok {
					continue
				}

				if section.groups {
					members := splitMembers(value)
					if !containsMember(members, username) {
						continue
					}

					remaining := make([]string, 0, len(members))
					for _, member := range members {
						if member != username {
							remaining = append(remaining, member)
						}
					}

					if len(remaining) == 0 {
						deletions[index] = true

						continue
					}

					replacements[index] = []fsx.TextLine{{
						Raw: key + " = " + strings.Join(remaining, separatorOf(value)),
						EOL: normaliseEOL(lines[index].EOL, newline),
					}}

					continue
				}

				if key == username {
					deletions[index] = true
				}
			}
		}

		if len(deletions) == 0 && len(replacements) == 0 {
			return lines, false, nil
		}

		return applyEdits(lines, deletions, replacements, nil), true, nil
	}, func(file *File) error {
		for _, name := range file.Groups() {
			for _, member := range file.UsersOfGroup(name) {
				if member == username {
					return fmt.Errorf("%q is still a member of group %q after the delete", username, name)
				}
			}
		}

		for _, section := range file.Sections {
			if strings.EqualFold(section.Name, GroupsSection) {
				continue
			}

			for _, entry := range section.Entries {
				if entry.Principal == username {
					return fmt.Errorf("%q still holds an ACL entry after the delete", username)
				}
			}
		}

		return nil
	})
}

// mutate runs the locked, hash-checked, lossless read-modify-write cycle for a
// user-level edit. The shared implementation is mutateTarget.
func (s *Store) mutate(
	ctx context.Context,
	username string,
	opts model.WriteOptions,
	edit func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error),
	check func(file *File) error,
) error {
	return s.mutateTarget(ctx, model.KindUser, username, opts, edit, check)
}

// mutateTarget runs the locked, hash-checked, lossless read-modify-write cycle.
// The target kind only shapes the typed conflict error the transport maps to
// 409; the edit itself is line oriented and never rewrites unaffected lines.
func (s *Store) mutateTarget(
	_ context.Context,
	kind model.Kind,
	name string,
	opts model.WriteOptions,
	edit func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error),
	check func(file *File) error,
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
			Kind:   kind,
			Name:   name,
			Reason: "the authz file changed since it was read",
		}
	}

	lines := fsx.SplitLines(snapshot.Data)
	updated, changed, err := edit(lines, blocks(lines))
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	if _, err := fsx.Backup(s.path, s.backupDir, s.keep); err != nil {
		return &Error{Op: "backup", Path: s.path, Err: err}
	}

	if err := fsx.WriteAtomic(s.path, fsx.JoinLines(updated), snapshot.Hash); err != nil {
		if fsx.IsConflict(err) {
			return &model.ConflictError{
				Kind:   kind,
				Name:   name,
				Reason: "the authz file was modified by another writer",
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

// applyEdits rebuilds the line slice: replacement lines take the place of
// their anchor, insertion lines follow theirs, deletions disappear and every
// other line is copied verbatim.
func applyEdits(
	lines []fsx.TextLine,
	deletions map[int]bool,
	replacements map[int][]fsx.TextLine,
	insertions map[int][]fsx.TextLine,
) []fsx.TextLine {
	out := make([]fsx.TextLine, 0, len(lines)+len(insertions))

	for index, line := range lines {
		if replacement, ok := replacements[index]; ok {
			out = append(out, replacement...)

			continue
		}

		if !deletions[index] {
			out = append(out, line)
		}

		if insertion, ok := insertions[index]; ok {
			out = append(out, insertion...)
		}
	}

	return out
}

// splitMembers parses a `[groups]` value into trimmed members.
func splitMembers(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		if member := strings.TrimSpace(part); member != "" {
			out = append(out, member)
		}
	}

	return out
}

func containsMember(members []string, username string) bool {
	for _, member := range members {
		if member == username {
			return true
		}
	}

	return false
}

// separatorOf keeps the spacing style of an existing member list. A list with
// no separator yet (a single member) gets the conventional ", ".
func separatorOf(value string) string {
	if !strings.Contains(value, ",") {
		return ", "
	}

	if strings.Contains(value, ", ") {
		return ", "
	}

	return ","
}

func normaliseEOL(eol, fallback string) string {
	if eol == "" {
		return fallback
	}

	return eol
}
