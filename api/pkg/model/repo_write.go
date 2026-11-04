package model

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// RepositoryStructure is the optional directory layout created together with a
// repository (plan T4.3).
type RepositoryStructure string

const (
	// StructureNone creates an empty repository.
	StructureNone RepositoryStructure = "none"
	// StructureSingle creates trunk/branches/tags at the repository root.
	StructureSingle RepositoryStructure = "single"
	// StructureMulti creates <projectName>/trunk|branches|tags.
	StructureMulti RepositoryStructure = "multi"
)

// ParseRepositoryStructure validates a user supplied structure. An empty value
// defaults to "none" so the field stays optional.
func ParseRepositoryStructure(raw string) (RepositoryStructure, error) {
	switch strings.TrimSpace(raw) {
	case "", string(StructureNone):
		return StructureNone, nil
	case string(StructureSingle):
		return StructureSingle, nil
	case string(StructureMulti):
		return StructureMulti, nil
	default:
		return "", &ValidationError{
			Field:  "structure",
			Reason: `must be one of "none", "single" or "multi"`,
		}
	}
}

// LayoutDirs returns the directories to create relative to the repository
// root. Multi-project layouts nest the whole set under projectName.
func (s RepositoryStructure) LayoutDirs(projectName string) []string {
	switch s {
	case StructureSingle:
		return []string{"trunk", "branches", "tags"}
	case StructureMulti:
		if projectName == "" {
			return nil
		}

		return []string{
			projectName + "/trunk",
			projectName + "/branches",
			projectName + "/tags",
		}
	default:
		return nil
	}
}

// RepositorySpec is a validated repository creation request.
type RepositorySpec struct {
	Name        string
	Parent      string
	Structure   RepositoryStructure
	ProjectName string
}

// Validate checks the name, the structure and the project name contract.
func (s RepositorySpec) Validate() error {
	if err := ValidateRepositoryName(s.Name); err != nil {
		return err
	}

	switch s.Structure {
	case "", StructureNone, StructureSingle:
		if s.ProjectName != "" {
			return &ValidationError{
				Field:  "projectName",
				Reason: `is only allowed with structure "multi"`,
			}
		}
	case StructureMulti:
		if s.ProjectName == "" {
			return &ValidationError{
				Field:  "projectName",
				Reason: `is required when structure is "multi"`,
			}
		}

		if err := ValidateRepositoryName(s.ProjectName); err != nil {
			var validation *ValidationError
			if errors.As(err, &validation) {
				return &ValidationError{Field: "projectName", Reason: validation.Reason}
			}

			return err
		}
	default:
		return &ValidationError{
			Field:  "structure",
			Reason: `must be one of "none", "single" or "multi"`,
		}
	}

	return nil
}

// RepositoryEntry is one child of a repository directory.
type RepositoryEntry struct {
	Name     string
	Kind     string
	Size     int64
	Revision int64
	Author   string
	Date     time.Time
}

// RepositoryTree is one directory listing before paging.
type RepositoryTree struct {
	Path     string
	Revision int64
	Entries  []RepositoryEntry
}

// RepositoryDeletion reports where a deleted repository was moved.
type RepositoryDeletion struct {
	Name      string
	TrashPath string
}

// ValidateRepositoryPath checks the `path` query of GET /repos/{repo}/tree.
//
// The value must stay relative (SVN resolves it against the repository root):
// absolute paths, Windows drive letters, backslashes, NUL/control characters
// and "." or ".." segments are rejected. An empty path means the root.
func ValidateRepositoryPath(raw string) error {
	if raw == "" {
		return nil
	}

	if strings.ContainsAny(raw, "\\\x00") {
		return &ValidationError{Field: "path", Reason: "must not contain backslashes or NUL"}
	}

	if strings.HasPrefix(raw, "/") {
		return &ValidationError{Field: "path", Reason: "must be a relative path"}
	}

	if len(raw) >= 2 && raw[1] == ':' {
		return &ValidationError{Field: "path", Reason: "must be a relative path"}
	}

	for _, r := range raw {
		if unicode.IsControl(r) {
			return &ValidationError{Field: "path", Reason: "must not contain control characters"}
		}
	}

	for _, segment := range strings.Split(raw, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return &ValidationError{
				Field:  "path",
				Reason: "must be a relative path without empty, '.' or '..' segments",
			}
		}
	}

	return nil
}
