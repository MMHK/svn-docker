package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

var _ model.AccessPathWriter = (*Store)(nil)

// CreateAccessPath appends a new `[repo:/path]` section at the end of the file.
// Everything already present is preserved verbatim (plan T3.3).
func (s *Store) CreateAccessPath(ctx context.Context, path string, opts model.WriteOptions) error {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	return s.mutateTarget(ctx, model.KindAccessPath, path, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		if findBlock(sections, path) != nil {
			return nil, false, &model.ConflictError{
				Kind:   model.KindAccessPath,
				Name:   path,
				Reason: "already exists",
			}
		}

		newline := fsx.Newline(lines)

		// Append directly after the previous line instead of inserting a blank
		// separator: deleting the section again then restores the file exactly,
		// and every original blank line stays where it was.
		out := append([]fsx.TextLine(nil), lines...)
		out = fsx.EnsureTrailingNewline(out)

		return append(out, fsx.TextLine{Raw: "[" + path + "]", EOL: newline}), true, nil
	}, func(file *File) error {
		if _, ok := file.byName[strings.ToLower(path)]; !ok {
			return fmt.Errorf("access path %q was not created", path)
		}

		return nil
	})
}

// DeleteAccessPath removes every section with this name: the header and every
// ACL entry below it. Comments and blank lines that belong to other sections
// are untouched.
func (s *Store) DeleteAccessPath(ctx context.Context, path string, opts model.WriteOptions) error {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	return s.mutateTarget(ctx, model.KindAccessPath, path, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		matches := findBlocks(sections, path)
		if len(matches) == 0 {
			return nil, false, &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
		}

		deletions := make(map[int]bool)

		for _, target := range matches {
			for index := target.index; index <= target.lastContent; index++ {
				deletions[index] = true
			}
		}

		return applyEdits(lines, deletions, nil, nil), true, nil
	}, func(file *File) error {
		if _, ok := file.byName[strings.ToLower(path)]; ok {
			return fmt.Errorf("access path %q is still present", path)
		}

		return nil
	})
}

// SetACL creates or updates one `principal = permission` line. A permission of
// "none" removes the line, which is the documented way to clear an assignment
// (plan T3.4). Only a user, `@group` or `*` may be written; `$authenticated`,
// `$anonymous` and `&alias` entries are read-only.
func (s *Store) SetACL(ctx context.Context, path, principal string, permission model.Permission, opts model.WriteOptions) error {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	if !permission.Valid() {
		return &model.ValidationError{Field: "permission", Reason: "must be one of none, r, rw"}
	}

	if !model.ValidPrincipal(principal) {
		return &model.ValidationError{
			Field:  "principal",
			Reason: "must be a user, an @group or *; $authenticated, $anonymous and &aliases are read-only",
		}
	}

	if permission == model.PermissionNone {
		return s.DeleteACL(ctx, path, principal, opts)
	}

	return s.mutateTarget(ctx, model.KindAccessPath, path, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		target := findBlock(sections, path)
		if target == nil {
			return nil, false, &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
		}

		indices := matchingEntries(lines, target, principal)
		newline := fsx.Newline(lines)
		value := string(permission)

		if len(indices) == 0 {
			line := fsx.TextLine{Raw: principal + " = " + value, EOL: newline}

			return insertAfter(lines, target.lastContent, []fsx.TextLine{line}, newline), true, nil
		}

		if len(indices) == 1 {
			if _, existing, ok := lines[indices[0]].SplitKeyValue(); ok && existing == value {
				return lines, false, nil
			}
		}

		replacement := []fsx.TextLine{{
			Raw: principal + " = " + value,
			EOL: normaliseEOL(lines[indices[0]].EOL, newline),
		}}

		deletions := make(map[int]bool, len(indices)-1)
		for _, index := range indices[1:] {
			deletions[index] = true
		}

		return applyEdits(lines, deletions, map[int][]fsx.TextLine{indices[0]: replacement}, nil), true, nil
	}, func(file *File) error {
		for _, entry := range file.ACL(path) {
			if entry.Principal == principal {
				if entry.Permission != permission {
					return fmt.Errorf("permission for %q on %q was not written as requested", principal, path)
				}

				return nil
			}
		}

		return fmt.Errorf("%q was not added to the ACL of %q", principal, path)
	})
}

// DeleteACL removes every line for principal from the path section. Removing an
// entry that is already absent is a no-op; a missing path is a 404.
func (s *Store) DeleteACL(ctx context.Context, path, principal string, opts model.WriteOptions) error {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	if !model.ValidPrincipal(principal) {
		return &model.ValidationError{
			Field:  "principal",
			Reason: "must be a user, an @group or *; $authenticated, $anonymous and &aliases are read-only",
		}
	}

	return s.mutateTarget(ctx, model.KindAccessPath, path, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		target := findBlock(sections, path)
		if target == nil {
			return nil, false, &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
		}

		indices := matchingEntries(lines, target, principal)
		if len(indices) == 0 {
			return lines, false, nil
		}

		deletions := make(map[int]bool, len(indices))
		for _, index := range indices {
			deletions[index] = true
		}

		return applyEdits(lines, deletions, nil, nil), true, nil
	}, func(file *File) error {
		for _, entry := range file.ACL(path) {
			if entry.Principal == principal {
				return fmt.Errorf("%q is still present in the ACL of %q", principal, path)
			}
		}

		return nil
	})
}

// matchingEntries returns the entry lines of a block whose key equals
// principal. Authz principals are case-sensitive, so this is an exact match.
func matchingEntries(lines []fsx.TextLine, target *block, principal string) []int {
	out := make([]int, 0, 1)

	for _, index := range target.entries {
		key, _, ok := lines[index].SplitKeyValue()
		if ok && key == principal {
			out = append(out, index)
		}
	}

	return out
}
