package svncli

import (
	"net/url"
	"path/filepath"
	"strings"
)

// FileURL renders a local filesystem path as a file:// URL, escaping each path
// segment with url.PathEscape (plan T4.2). Segments are escaped individually so
// a space, '#', '%' or non-ASCII byte in the parent path stays valid.
func FileURL(path string) string {
	slashed := filepath.ToSlash(strings.TrimSpace(path))
	if slashed == "" {
		return "file://"
	}

	absolute := strings.HasPrefix(slashed, "/") || hasDrivePrefix(slashed)

	segments := strings.Split(strings.Trim(slashed, "/"), "/")
	escaped := make([]string, 0, len(segments))

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		escaped = append(escaped, url.PathEscape(segment))
	}

	prefix := "file://"
	if absolute {
		prefix = "file:///"
	}

	return prefix + strings.Join(escaped, "/")
}

// JoinURL appends path segments to an existing Subversion URL. Each segment is
// escaped separately, which matches Subversion's own URL layout and keeps a
// segment that contains a slash from silently creating a nested path.
func JoinURL(base string, segments ...string) string {
	out := strings.TrimSuffix(base, "/")

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		out += "/" + url.PathEscape(segment)
	}

	return out
}

// hasDrivePrefix reports whether a slash-normalised path starts with a Windows
// drive letter such as "C:/".
func hasDrivePrefix(path string) bool {
	if len(path) < 2 || path[1] != ':' {
		return false
	}

	letter := path[0]

	return (letter >= 'a' && letter <= 'z') || (letter >= 'A' && letter <= 'Z')
}
