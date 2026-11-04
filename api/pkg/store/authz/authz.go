// Package authz parses Subversion's svn-access-file (authz) format. P1 is
// read-only, but the parser keeps every raw line and line ending so the P3
// line-oriented writer can be lossless (plan section 5.1).
package authz

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// GroupsSection is the reserved section name for group definitions.
const GroupsSection = "groups"

// Line is one physical line, including its original ending.
type Line struct {
	Raw    string
	EOL    string
	Number int
}

// Entry is one `principal = permission` line inside a section.
type Entry struct {
	Principal string
	Value     string
	Line      int
}

// Section is one `[name]` block.
type Section struct {
	Name       string
	HeaderLine int
	StartLine  int
	EndLine    int
	Entries    []Entry
}

// File is a parsed authz file.
type File struct {
	Path     string
	Lines    []Line
	Sections []Section

	groups     map[string][]string
	groupOrder []string
	byName     map[string]int
}

// Error is the typed failure returned by the store.
type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("authz: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Parse reads authz content. Unknown lines and sections are preserved verbatim
// and simply carry no entries.
func Parse(path string, data []byte) *File {
	file := &File{
		Path:   path,
		groups: make(map[string][]string),
		byName: make(map[string]int),
	}

	var current *Section

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

		number := len(file.Lines) + 1
		file.Lines = append(file.Lines, Line{Raw: raw, EOL: eol, Number: number})

		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			file.Sections = append(file.Sections, Section{
				Name:       name,
				HeaderLine: number,
				StartLine:  number,
				EndLine:    number,
			})
			current = &file.Sections[len(file.Sections)-1]
			file.byName[strings.ToLower(name)] = len(file.Sections) - 1

			continue
		}

		if current == nil {
			// Stray key=value before the first section is preserved but not
			// interpreted.
			continue
		}

		key, value, ok := strings.Cut(raw, "=")
		if !ok {
			current.EndLine = number

			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		current.EndLine = number

		if key == "" {
			continue
		}

		if strings.EqualFold(current.Name, GroupsSection) {
			file.appendGroupMembers(key, value)

			continue
		}

		current.Entries = append(current.Entries, Entry{Principal: key, Value: value, Line: number})
	}

	return file
}

func (f *File) appendGroupMembers(name, value string) {
	if _, seen := f.groups[name]; !seen {
		f.groupOrder = append(f.groupOrder, name)
		// Register the key even when the group has no members yet, otherwise
		// `name =` (a freshly created group) would be invisible to HasGroup.
		f.groups[name] = []string{}
	}

	for _, member := range strings.Split(value, ",") {
		member = strings.TrimSpace(member)
		if member == "" {
			continue
		}

		f.groups[name] = append(f.groups[name], member)
	}
}

// Paths returns non-groups section names in file order.
func (f *File) Paths() []string {
	out := make([]string, 0, len(f.Sections))

	for _, section := range f.Sections {
		if strings.EqualFold(section.Name, GroupsSection) {
			continue
		}

		out = append(out, section.Name)
	}

	return out
}

// Groups returns the group names in file order.
func (f *File) Groups() []string {
	out := make([]string, len(f.groupOrder))
	copy(out, f.groupOrder)

	return out
}

// UsersOfGroup returns the members of a group in file order.
func (f *File) UsersOfGroup(name string) []string {
	members := f.groups[name]
	out := make([]string, len(members))
	copy(out, members)

	return out
}

// HasGroup reports whether the [groups] section defines name.
func (f *File) HasGroup(name string) bool {
	_, ok := f.groups[name]

	return ok
}

// ACL returns the access-path entries for path. A missing section yields an
// empty ACL: an access path referenced only by roles is not an error.
func (f *File) ACL(path string) []model.ACLEntry {
	index, ok := f.byName[strings.ToLower(strings.TrimSpace(path))]
	if !ok {
		return nil
	}

	section := f.Sections[index]
	out := make([]model.ACLEntry, 0, len(section.Entries))

	for _, entry := range section.Entries {
		out = append(out, model.ACLEntry{
			Principal:  entry.Principal,
			Permission: PermissionOf(entry.Value),
		})
	}

	return out
}

// PermissionOf maps an authz permission string to the model value. Unknown
// values are treated as none so they never grant access accidentally.
func PermissionOf(value string) model.Permission {
	normalised := strings.ToLower(strings.TrimSpace(value))

	switch {
	case normalised == "":
		return model.PermissionNone
	case strings.Contains(normalised, "w"):
		return model.PermissionReadWrite
	case strings.Contains(normalised, "r"):
		return model.PermissionRead
	default:
		return model.PermissionNone
	}
}

// Store is an authz-backed group/access-path store. Reads re-parse the file;
// writes are locked, lossless and atomic.
type Store struct {
	path      string
	backupDir string
	keep      int

	writeMu sync.Mutex
}

// New builds a store for path.
func New(path string, opts ...Option) *Store {
	store := &Store{path: path, keep: fsx.DefaultBackupKeep}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// Path returns the configured file path.
func (s *Store) Path() string {
	return s.path
}

// Load reads and parses the file.
func (s *Store) Load() (*File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, &Error{Op: "read", Path: s.path, Err: err}
	}

	return Parse(s.path, data), nil
}

var (
	_ model.GroupStore      = (*Store)(nil)
	_ model.AccessPathStore = (*Store)(nil)
)

// ListGroups implements model.GroupStore.
func (s *Store) ListGroups(_ context.Context, query model.PageQuery) (model.Page[model.Group], error) {
	file, err := s.Load()
	if err != nil {
		return model.Page[model.Group]{}, err
	}

	all := make([]model.Group, 0, len(file.groupOrder))
	for _, name := range file.groupOrder {
		all = append(all, model.Group{Name: name, Members: file.UsersOfGroup(name)})
	}

	normalized := query.Normalize()
	needle := strings.ToLower(normalized.Q)

	filtered := make([]model.Group, 0, len(all))

	for _, group := range all {
		if needle == "" || strings.Contains(strings.ToLower(group.Name), needle) {
			filtered = append(filtered, group)
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

	return model.Page[model.Group]{
		Items: filtered[start:end],
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// GetGroup implements model.GroupStore.
func (s *Store) GetGroup(_ context.Context, name string) (model.Group, error) {
	file, err := s.Load()
	if err != nil {
		return model.Group{}, err
	}

	if !file.HasGroup(name) {
		return model.Group{}, &model.NotFoundError{Kind: model.KindGroup, Name: name}
	}

	return model.Group{Name: name, Members: file.UsersOfGroup(name)}, nil
}

// GroupsOfUser implements model.GroupStore.
func (s *Store) GroupsOfUser(_ context.Context, username string) ([]string, error) {
	file, err := s.Load()
	if err != nil {
		return nil, err
	}

	var out []string

	for _, name := range file.groupOrder {
		for _, member := range file.groups[name] {
			if member == username {
				out = append(out, name)

				break
			}
		}
	}

	return out, nil
}

// ListAccessPaths implements model.AccessPathStore.
func (s *Store) ListAccessPaths(_ context.Context, query model.PageQuery) (model.Page[model.AccessPath], error) {
	file, err := s.Load()
	if err != nil {
		return model.Page[model.AccessPath]{}, err
	}

	all := make([]model.AccessPath, 0, len(file.Sections))

	for _, section := range file.Sections {
		if strings.EqualFold(section.Name, GroupsSection) {
			continue
		}

		all = append(all, model.AccessPath{Path: section.Name, ACL: file.ACL(section.Name)})
	}

	normalized := query.Normalize()
	needle := strings.ToLower(normalized.Q)

	filtered := make([]model.AccessPath, 0, len(all))

	for _, item := range all {
		if needle == "" || strings.Contains(strings.ToLower(item.Path), needle) {
			filtered = append(filtered, item)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Path) < strings.ToLower(filtered[j].Path)
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

	return model.Page[model.AccessPath]{
		Items: filtered[start:end],
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// AllAccessPaths implements model.AccessPathStore without paging, so derived
// views such as a user's effective access list never silently truncate.
func (s *Store) AllAccessPaths(_ context.Context) ([]model.AccessPath, error) {
	file, err := s.Load()
	if err != nil {
		return nil, err
	}

	out := make([]model.AccessPath, 0, len(file.Sections))

	for _, section := range file.Sections {
		if strings.EqualFold(section.Name, GroupsSection) {
			continue
		}

		out = append(out, model.AccessPath{Path: section.Name, ACL: file.ACL(section.Name)})
	}

	return out, nil
}

// GetAccessPath implements model.AccessPathStore.
func (s *Store) GetAccessPath(_ context.Context, path string) (model.AccessPath, error) {
	file, err := s.Load()
	if err != nil {
		return model.AccessPath{}, err
	}

	index, ok := file.byName[strings.ToLower(strings.TrimSpace(path))]
	if !ok || strings.EqualFold(file.Sections[index].Name, GroupsSection) {
		return model.AccessPath{}, &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
	}

	return model.AccessPath{Path: file.Sections[index].Name, ACL: file.ACL(path)}, nil
}

// AccessPathsOfGroup implements model.AccessPathStore.
func (s *Store) AccessPathsOfGroup(_ context.Context, group string) ([]string, error) {
	file, err := s.Load()
	if err != nil {
		return nil, err
	}

	principal := "@" + group
	out := make([]string, 0)

	for _, section := range file.Sections {
		if strings.EqualFold(section.Name, GroupsSection) {
			continue
		}

		for _, entry := range section.Entries {
			if entry.Principal == principal {
				out = append(out, section.Name)

				break
			}
		}
	}

	return out, nil
}
