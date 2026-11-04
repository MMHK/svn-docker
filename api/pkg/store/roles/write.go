package roles

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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

var _ model.RoleWriter = (*Store)(nil)

// section is one `[name]` block of the role file.
type section struct {
	name        string // trimmed name without brackets
	index       int    // index in lines
	lastContent int    // index of the last non-blank line of the block
	accessPaths bool   // name ends with :AccessPaths
	user        string // owner without the :AccessPaths suffix
}

// roleLine is one editable role entry.
type roleLine struct {
	index      int
	role       model.Role
	valueStyle bool // `Role=<name>` (true) vs `<name>=` (false)
}

// segments splits the file into section blocks. Lines before the first header
// are ignored, which is exactly what the v2 reader does.
func segments(lines []fsx.TextLine) []section {
	out := make([]section, 0, 4)

	for index, line := range lines {
		if name, ok := line.IsSectionHeader(); ok {
			accessPaths := len(name) > len(AccessPathsSuffix) &&
				strings.EqualFold(name[len(name)-len(AccessPathsSuffix):], AccessPathsSuffix)

			owner := name
			if accessPaths {
				owner = strings.TrimSpace(name[:len(name)-len(AccessPathsSuffix)])
			}

			out = append(out, section{
				name:        name,
				index:       index,
				lastContent: index,
				accessPaths: accessPaths,
				user:        owner,
			})

			continue
		}

		if len(out) == 0 || line.Blank() || line.Comment() {
			continue
		}

		out[len(out)-1].lastContent = index
	}

	return out
}

// rolesOf collects the editable role lines belonging to username.
func rolesOf(lines []fsx.TextLine, blocks []section, username string) []roleLine {
	out := make([]roleLine, 0, 2)

	for _, block := range blocks {
		if block.accessPaths || !strings.EqualFold(block.user, username) {
			continue
		}

		for index := block.index + 1; index <= block.lastContent; index++ {
			key, value, ok := lines[index].SplitKeyValue()
			if !ok {
				continue
			}

			if strings.EqualFold(key, "Role") {
				role, valid := parseRole(value)
				if valid {
					out = append(out, roleLine{index: index, role: role, valueStyle: true})
				}

				continue
			}

			if role, valid := parseRole(key); valid {
				out = append(out, roleLine{index: index, role: role})
			}
		}
	}

	return out
}

// SetRoles replaces the role set of username and leaves every other line of
// the file byte identical (plan section 5.3).
func (s *Store) SetRoles(ctx context.Context, username string, roles []model.Role, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	desired, err := normaliseRoles(roles)
	if err != nil {
		return err
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, blocks []section) ([]fsx.TextLine, bool, error) {
		return setRolesLines(lines, blocks, username, desired)
	}, func(lines []fsx.TextLine) error {
		got := rolesOf(lines, segments(lines), username)
		if !sameRoleSet(got, desired) {
			return fmt.Errorf("role assignment for %q was not written as requested", username)
		}

		return nil
	})
}

func setRolesLines(
	lines []fsx.TextLine,
	blocks []section,
	username string,
	desired []model.Role,
) ([]fsx.TextLine, bool, error) {
	existing := rolesOf(lines, blocks, username)
	if sameRoleSet(existing, desired) {
		return lines, false, nil
	}

	// Preserve the spelling the file already uses. A fresh section uses the
	// v2 layout (`Administrator=`), which is what data/userroleassignments.ini
	// contains today.
	valueStyle := false
	for _, line := range existing {
		if line.valueStyle {
			valueStyle = true

			break
		}
	}

	newline := fsx.Newline(lines)
	if len(existing) > 0 && existing[0].index < len(lines) && lines[existing[0].index].EOL != "" {
		newline = lines[existing[0].index].EOL
	}

	replacement := make([]fsx.TextLine, 0, len(desired))
	for _, role := range desired {
		raw := string(role) + "="
		if valueStyle {
			raw = "Role=" + string(role)
		}

		replacement = append(replacement, fsx.TextLine{Raw: raw, EOL: newline})
	}

	deletions := make(map[int]bool, len(existing))
	for _, line := range existing {
		deletions[line.index] = true
	}

	if len(existing) > 0 {
		delete(deletions, existing[0].index)

		return applyEdits(lines, deletions, map[int][]fsx.TextLine{existing[0].index: replacement}, nil), true, nil
	}

	// The user has no role line yet. Insert after the section header when the
	// section exists, otherwise append a new section.
	for _, block := range blocks {
		if block.accessPaths || !strings.EqualFold(block.user, username) {
			continue
		}

		if lines[block.index].EOL == "" {
			// The header is the last line and has no ending: give it one
			// before inserting below it.
			lines = append([]fsx.TextLine(nil), lines...)
			lines[block.index].EOL = newline
		}

		return applyEdits(lines, nil, nil, map[int][]fsx.TextLine{block.index: replacement}), true, nil
	}

	if len(desired) == 0 {
		return lines, false, nil
	}

	lines = append([]fsx.TextLine(nil), fsx.EnsureTrailingNewline(append([]fsx.TextLine(nil), lines...))...)
	if len(lines) > 0 && !lines[len(lines)-1].Blank() {
		lines = append(lines, fsx.TextLine{EOL: newline})
	}

	lines = append(lines, fsx.TextLine{Raw: "[" + username + "]", EOL: newline})
	lines = append(lines, replacement...)

	return lines, true, nil
}

// RemoveUser deletes the `[user]` and `[user:AccessPaths]` blocks in full.
func (s *Store) RemoveUser(ctx context.Context, username string, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, blocks []section) ([]fsx.TextLine, bool, error) {
		deletions := make(map[int]bool)

		for _, block := range blocks {
			if !strings.EqualFold(block.user, username) {
				continue
			}

			for index := block.index; index <= block.lastContent; index++ {
				deletions[index] = true
			}
		}

		if len(deletions) == 0 {
			return lines, false, nil
		}

		return applyEdits(lines, deletions, nil, nil), true, nil
	}, func(lines []fsx.TextLine) error {
		blocks := segments(lines)

		for _, block := range blocks {
			if strings.EqualFold(block.user, username) {
				return fmt.Errorf("role assignments for %q are still present", username)
			}
		}

		return nil
	})
}

// AddAccessPaths records project-manager scopes under `[user:AccessPaths]`
// using the v2 index keys. P3 wires this to the Project-Manager flow.
func (s *Store) AddAccessPaths(ctx context.Context, username string, paths []string, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	sanitised, err := normalisePaths(paths)
	if err != nil {
		return err
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, blocks []section) ([]fsx.TextLine, bool, error) {
		return addAccessPathsLines(lines, blocks, username, sanitised)
	}, nil)
}

func addAccessPathsLines(
	lines []fsx.TextLine,
	blocks []section,
	username string,
	paths []string,
) ([]fsx.TextLine, bool, error) {
	existing := make(map[string]bool)
	next := 0

	var target *section

	for index := range blocks {
		block := blocks[index]
		if !block.accessPaths || !strings.EqualFold(block.user, username) {
			continue
		}

		if target == nil {
			target = &blocks[index]
		}

		for line := block.index + 1; line <= block.lastContent; line++ {
			key, value, ok := lines[line].SplitKeyValue()
			if !ok || value == "" {
				continue
			}

			existing[value] = true

			if number, err := strconv.Atoi(key); err == nil && number >= next {
				next = number + 1
			}
		}
	}

	newline := fsx.Newline(lines)

	added := make([]fsx.TextLine, 0, len(paths))

	for _, path := range paths {
		if existing[path] {
			continue
		}

		existing[path] = true
		added = append(added, fsx.TextLine{
			Raw: strconv.Itoa(next) + "=" + path,
			EOL: newline,
		})
		next++
	}

	if len(added) == 0 {
		return lines, false, nil
	}

	if target == nil {
		lines = append([]fsx.TextLine(nil), fsx.EnsureTrailingNewline(append([]fsx.TextLine(nil), lines...))...)
		if len(lines) > 0 && !lines[len(lines)-1].Blank() {
			lines = append(lines, fsx.TextLine{EOL: newline})
		}

		lines = append(lines, fsx.TextLine{Raw: "[" + username + AccessPathsSuffix + "]", EOL: newline})
		lines = append(lines, added...)

		return lines, true, nil
	}

	if lines[target.lastContent].EOL == "" {
		lines = append([]fsx.TextLine(nil), lines...)
		lines[target.lastContent].EOL = newline
	}

	return applyEdits(lines, nil, nil, map[int][]fsx.TextLine{target.lastContent: added}), true, nil
}

// RemoveAccessPaths deletes the matching `N=path` lines of `[user:AccessPaths]`.
// The indexes of the remaining lines are left untouched so the file stays
// readable by the v2 tool.
func (s *Store) RemoveAccessPaths(ctx context.Context, username string, paths []string, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	sanitised, err := normalisePaths(paths)
	if err != nil {
		return err
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, blocks []section) ([]fsx.TextLine, bool, error) {
		remove := make(map[string]bool, len(sanitised))
		for _, path := range sanitised {
			remove[path] = true
		}

		deletions := make(map[int]bool)

		for _, block := range blocks {
			if !block.accessPaths || !strings.EqualFold(block.user, username) {
				continue
			}

			for line := block.index + 1; line <= block.lastContent; line++ {
				_, value, ok := lines[line].SplitKeyValue()
				if ok && remove[value] {
					deletions[line] = true
				}
			}
		}

		if len(deletions) == 0 {
			return lines, false, nil
		}

		return applyEdits(lines, deletions, nil, nil), true, nil
	}, nil)
}

// SetAccessPaths replaces the whole `[user:AccessPaths]` list.
func (s *Store) SetAccessPaths(ctx context.Context, username string, paths []string, opts model.WriteOptions) error {
	if err := model.ValidateUserName(username); err != nil {
		return err
	}

	sanitised, err := normalisePaths(paths)
	if err != nil {
		return err
	}

	return s.mutate(ctx, username, opts, func(lines []fsx.TextLine, blocks []section) ([]fsx.TextLine, bool, error) {
		current := make([]string, 0, len(sanitised))
		deletions := make(map[int]bool)

		var target *section

		for index := range blocks {
			block := blocks[index]
			if !block.accessPaths || !strings.EqualFold(block.user, username) {
				continue
			}

			if target == nil {
				target = &blocks[index]
			}

			for line := block.index + 1; line <= block.lastContent; line++ {
				if _, value, ok := lines[line].SplitKeyValue(); ok {
					current = append(current, value)
					deletions[line] = true
				}
			}
		}

		if equalStrings(current, sanitised) {
			return lines, false, nil
		}

		newline := fsx.Newline(lines)
		replacement := make([]fsx.TextLine, 0, len(sanitised))

		for index, path := range sanitised {
			replacement = append(replacement, fsx.TextLine{Raw: strconv.Itoa(index) + "=" + path, EOL: newline})
		}

		if target != nil {
			delete(deletions, target.index)
			// Re-insert at the end of the block: delete the old value lines and
			// put the new list after the last surviving line of the section.
			insertAfter := target.index
			for line := target.index + 1; line <= target.lastContent; line++ {
				if !deletions[line] {
					insertAfter = line
				}
			}

			return applyEdits(lines, deletions, nil, map[int][]fsx.TextLine{insertAfter: replacement}), true, nil
		}

		if len(sanitised) == 0 {
			return lines, false, nil
		}

		lines = append([]fsx.TextLine(nil), fsx.EnsureTrailingNewline(append([]fsx.TextLine(nil), lines...))...)
		if len(lines) > 0 && !lines[len(lines)-1].Blank() {
			lines = append(lines, fsx.TextLine{EOL: newline})
		}

		lines = append(lines, fsx.TextLine{Raw: "[" + username + AccessPathsSuffix + "]", EOL: newline})
		lines = append(lines, replacement...)

		return lines, true, nil
	}, nil)
}

// AssignRole keeps the P1 bootstrap flow working: it adds one role to the
// current set through the same lossless editor.
func (s *Store) AssignRole(ctx context.Context, username string, role model.Role) error {
	if !role.Valid() {
		return &model.ValidationError{Field: "role", Reason: "unknown role " + string(role)}
	}

	assignments, err := s.Assignments(ctx)
	if err != nil {
		return err
	}

	roles := append([]model.Role(nil), assignments.Roles[username]...)
	for _, held := range roles {
		if held == role {
			return nil
		}
	}

	return s.SetRoles(ctx, username, append(roles, role), model.WriteOptions{})
}

// mutate is the shared locked read-modify-write cycle for the role file.
func (s *Store) mutate(
	_ context.Context,
	username string,
	opts model.WriteOptions,
	edit func(lines []fsx.TextLine, blocks []section) ([]fsx.TextLine, bool, error),
	check func(lines []fsx.TextLine) error,
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
			Name:   username,
			Reason: "the role file changed since it was read",
		}
	}

	lines := fsx.SplitLines(snapshot.Data)
	updated, changed, err := edit(lines, segments(lines))
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
				Kind:   model.KindUser,
				Name:   username,
				Reason: "the role file was modified by another writer",
			}
		}

		return &Error{Op: "write", Path: s.path, Err: err}
	}

	s.invalidate()

	if check != nil {
		written, err := fsx.ReadFile(s.path)
		if err != nil {
			return &Error{Op: "verify", Path: s.path, Err: err}
		}

		if err := check(fsx.SplitLines(written.Data)); err != nil {
			// Put the previous bytes back rather than leaving a half applied
			// edit behind.
			if rollbackErr := fsx.WriteAtomic(s.path, snapshot.Data, ""); rollbackErr != nil {
				return &Error{
					Op:   "rollback",
					Path: s.path,
					Err:  fmt.Errorf("%v (rollback also failed: %w)", err, rollbackErr),
				}
			}

			s.invalidate()

			return &Error{Op: "verify", Path: s.path, Err: err}
		}
	}

	return nil
}

// invalidate drops the read cache after a write so the next read re-parses.
func (s *Store) invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loaded = false
	s.cache = nil
}

// applyEdits rebuilds the line slice: replacement lines take the place of their
// anchor line, insertion lines follow their anchor, deletions disappear and
// everything else is copied verbatim.
func applyEdits(
	lines []fsx.TextLine,
	deletions map[int]bool,
	replacements map[int][]fsx.TextLine,
	insertions map[int][]fsx.TextLine,
) []fsx.TextLine {
	out := make([]fsx.TextLine, 0, len(lines)+len(replacements)+len(insertions))

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

func normaliseRoles(roles []model.Role) ([]model.Role, error) {
	out := make([]model.Role, 0, len(roles))

	for _, role := range roles {
		if !role.Valid() {
			return nil, &model.ValidationError{Field: "roles", Reason: "unknown role " + string(role)}
		}

		if !containsRole(out, role) {
			out = append(out, role)
		}
	}

	return out, nil
}

func containsRole(roles []model.Role, candidate model.Role) bool {
	for _, role := range roles {
		if role == candidate {
			return true
		}
	}

	return false
}

func sameRoleSet(existing []roleLine, desired []model.Role) bool {
	if len(existing) != len(desired) {
		return false
	}

	have := make([]string, 0, len(existing))
	for _, line := range existing {
		have = append(have, string(line.role))
	}

	want := make([]string, 0, len(desired))
	for _, role := range desired {
		want = append(want, string(role))
	}

	sort.Strings(have)
	sort.Strings(want)

	for index := range have {
		if have[index] != want[index] {
			return false
		}
	}

	return true
}

func normalisePaths(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))

	for _, path := range paths {
		if err := (model.AccessPath{Path: path}).Validate(); err != nil {
			return nil, err
		}

		if seen[path] {
			continue
		}

		seen[path] = true
		out = append(out, path)
	}

	return out, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
