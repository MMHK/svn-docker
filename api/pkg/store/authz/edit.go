package authz

import (
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// findBlock returns the first section with the given name (case-insensitive),
// or nil. Sections are matched case-insensitively for lookup while the
// original spelling is always preserved on write.
func findBlock(sections []block, name string) *block {
	for index := range sections {
		if strings.EqualFold(sections[index].name, name) {
			return &sections[index]
		}
	}

	return nil
}

// findBlocks returns every section with the given name. A hand-maintained authz
// file may repeat a section; editing must touch all of them or the parsed view
// (which takes the last section) would disagree with the file.
func findBlocks(sections []block, name string) []*block {
	out := make([]*block, 0, 1)

	for index := range sections {
		if strings.EqualFold(sections[index].name, name) {
			out = append(out, &sections[index])
		}
	}

	return out
}

// entryKeys returns the parsed key of every editable entry line of a block.
func entryKeys(lines []fsx.TextLine, target *block) []string {
	out := make([]string, 0, len(target.entries))

	for _, index := range target.entries {
		key, _, ok := lines[index].SplitKeyValue()
		if !ok {
			continue
		}

		out = append(out, key)
	}

	return out
}

// hasEntryKey reports whether the block carries an entry with key, matched
// case-insensitively (the authz reader treats principals and groups this way).
func hasEntryKey(lines []fsx.TextLine, target *block, key string) bool {
	for _, candidate := range entryKeys(lines, target) {
		if strings.EqualFold(candidate, key) {
			return true
		}
	}

	return false
}

// appendToFile adds lines at the end of the file, making sure the previous last
// line has an ending and that sections stay separated by one blank line.
func appendToFile(lines []fsx.TextLine, newline string, added ...fsx.TextLine) []fsx.TextLine {
	out := append([]fsx.TextLine(nil), lines...)
	out = fsx.EnsureTrailingNewline(out)

	if len(out) > 0 && !out[len(out)-1].Blank() {
		out = append(out, fsx.TextLine{EOL: newline})
	}

	return append(out, added...)
}

// insertAfter puts lines directly below the anchor line. The anchor is given an
// ending first when it is a last line without one.
func insertAfter(lines []fsx.TextLine, anchor int, added []fsx.TextLine, newline string) []fsx.TextLine {
	if anchor < 0 || anchor >= len(lines) {
		return append(lines, added...)
	}

	if lines[anchor].EOL == "" {
		lines = append([]fsx.TextLine(nil), lines...)
		lines[anchor].EOL = newline
	}

	return applyEdits(lines, nil, nil, map[int][]fsx.TextLine{anchor: added})
}

// normaliseMembers validates and de-duplicates a member list while preserving
// the submitted order.
func normaliseMembers(members []string) ([]string, error) {
	out := make([]string, 0, len(members))
	seen := make(map[string]bool, len(members))

	for _, member := range members {
		member = strings.TrimSpace(member)

		if err := model.ValidateUserName(member); err != nil {
			return nil, err
		}

		if seen[member] {
			continue
		}

		seen[member] = true
		out = append(out, member)
	}

	return out, nil
}

// groupLine renders one `[groups]` entry. An empty member list keeps the group
// defined as `name =`, which is how a freshly created group looks.
func groupLine(name string, members []string) string {
	if len(members) == 0 {
		return name + " ="
	}

	return name + " = " + strings.Join(members, ", ")
}

func sameMembers(left, right []string) bool {
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
