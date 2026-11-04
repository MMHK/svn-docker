package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

var _ model.GroupWriter = (*Store)(nil)

// CreateGroup appends `name =` to the [groups] block, or creates the block when
// the file has none. Existing lines (including comments and unknown sections)
// are byte identical afterwards.
func (s *Store) CreateGroup(ctx context.Context, group model.Group, opts model.WriteOptions) error {
	if err := group.Validate(); err != nil {
		return err
	}

	name := strings.TrimSpace(group.Name)

	members, err := normaliseMembers(group.Members)
	if err != nil {
		return err
	}

	return s.mutateTarget(ctx, model.KindGroup, name, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		newline := fsx.Newline(lines)
		line := fsx.TextLine{Raw: groupLine(name, members), EOL: newline}

		groups := findBlock(sections, GroupsSection)
		if groups == nil {
			return appendToFile(
				lines,
				newline,
				fsx.TextLine{Raw: "[" + GroupsSection + "]", EOL: newline},
				line,
			), true, nil
		}

		if hasEntryKey(lines, groups, name) {
			return nil, false, &model.ConflictError{
				Kind:   model.KindGroup,
				Name:   name,
				Reason: "already exists",
			}
		}

		return insertAfter(lines, groups.lastContent, []fsx.TextLine{line}, newline), true, nil
	}, func(file *File) error {
		if !file.HasGroup(name) {
			return fmt.Errorf("group %q was not created", name)
		}

		if !sameMembers(file.UsersOfGroup(name), members) {
			return fmt.Errorf("members of group %q were not written as requested", name)
		}

		return nil
	})
}

// DeleteGroup removes every `name = ...` line of [groups] and every `@name`
// ACL entry in the access-path sections. User entries with the same spelling
// are never touched (plan T3.2).
func (s *Store) DeleteGroup(ctx context.Context, name string, opts model.WriteOptions) error {
	if err := model.ValidateGroupName(name); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	principal := "@" + name

	return s.mutateTarget(ctx, model.KindGroup, name, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		found := false
		deletions := make(map[int]bool)

		if groups := findBlock(sections, GroupsSection); groups != nil {
			for _, index := range groups.entries {
				key, _, ok := lines[index].SplitKeyValue()
				if !ok || !strings.EqualFold(key, name) {
					continue
				}

				found = true
				deletions[index] = true
			}
		}

		if !found {
			return nil, false, &model.NotFoundError{Kind: model.KindGroup, Name: name}
		}

		for _, section := range sections {
			if section.groups {
				continue
			}

			for _, index := range section.entries {
				key, _, ok := lines[index].SplitKeyValue()
				if ok && strings.EqualFold(key, principal) {
					deletions[index] = true
				}
			}
		}

		return applyEdits(lines, deletions, nil, nil), true, nil
	}, func(file *File) error {
		if file.HasGroup(name) {
			return fmt.Errorf("group %q is still defined", name)
		}

		for _, section := range file.Sections {
			if strings.EqualFold(section.Name, GroupsSection) {
				continue
			}

			for _, entry := range section.Entries {
				if strings.EqualFold(entry.Principal, principal) {
					return fmt.Errorf("%s is still referenced by %s", principal, section.Name)
				}
			}
		}

		return nil
	})
}

// SetGroupMembers replaces the member list of an existing group. Only the
// matching `[groups]` line (and duplicate copies of it) is rewritten.
func (s *Store) SetGroupMembers(ctx context.Context, name string, members []string, opts model.WriteOptions) error {
	if err := model.ValidateGroupName(name); err != nil {
		return err
	}

	name = strings.TrimSpace(name)

	desired, err := normaliseMembers(members)
	if err != nil {
		return err
	}

	return s.mutateTarget(ctx, model.KindGroup, name, opts, func(lines []fsx.TextLine, sections []block) ([]fsx.TextLine, bool, error) {
		groups := findBlock(sections, GroupsSection)
		if groups == nil {
			return nil, false, &model.NotFoundError{Kind: model.KindGroup, Name: name}
		}

		indices := make([]int, 0, 1)
		current := make([]string, 0, len(desired))
		firstKey := name

		for _, index := range groups.entries {
			key, value, ok := lines[index].SplitKeyValue()
			if !ok || !strings.EqualFold(key, name) {
				continue
			}

			if len(indices) == 0 {
				firstKey = key
			}

			indices = append(indices, index)
			current = append(current, splitMembers(value)...)
		}

		if len(indices) == 0 {
			return nil, false, &model.NotFoundError{Kind: model.KindGroup, Name: name}
		}

		if sameMembers(current, desired) && len(indices) == 1 {
			return lines, false, nil
		}

		newline := fsx.Newline(lines)
		replacement := []fsx.TextLine{{
			Raw: groupLine(firstKey, desired),
			EOL: normaliseEOL(lines[indices[0]].EOL, newline),
		}}

		deletions := make(map[int]bool, len(indices))
		for _, index := range indices[1:] {
			deletions[index] = true
		}

		return applyEdits(lines, deletions, map[int][]fsx.TextLine{indices[0]: replacement}, nil), true, nil
	}, func(file *File) error {
		if !file.HasGroup(name) {
			return fmt.Errorf("group %q disappeared while writing its members", name)
		}

		if !sameMembers(file.UsersOfGroup(name), desired) {
			return fmt.Errorf("members of group %q were not written as requested", name)
		}

		return nil
	})
}
