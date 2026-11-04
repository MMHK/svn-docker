package fsx

import "strings"

// TextLine is one physical line: Raw excludes the line ending, EOL is "" for
// the last line of a file that does not end with a newline. Editing code only
// replaces the lines it must, so comments, ordering, indentation and CRLF
// survive a write (plan sections 5.1-5.3).
type TextLine struct {
	Raw string
	EOL string
}

// SplitLines splits data into raw lines, preserving each line ending. The
// result is lossless: JoinLines(SplitLines(data)) == data for any input.
func SplitLines(data []byte) []TextLine {
	text := string(data)
	lines := make([]TextLine, 0, strings.Count(text, "\n")+1)

	for len(text) > 0 {
		var raw, eol string

		if index := strings.IndexByte(text, '\n'); index >= 0 {
			raw, text = text[:index], text[index+1:]
			eol = "\n"

			if strings.HasSuffix(raw, "\r") {
				raw = raw[:len(raw)-1]
				eol = "\r\n"
			}
		} else {
			raw, text = text, ""
		}

		lines = append(lines, TextLine{Raw: raw, EOL: eol})
	}

	return lines
}

// JoinLines is the inverse of SplitLines.
func JoinLines(lines []TextLine) []byte {
	var builder strings.Builder

	for _, line := range lines {
		builder.WriteString(line.Raw)
		builder.WriteString(line.EOL)
	}

	return []byte(builder.String())
}

// Trimmed returns the line without surrounding whitespace.
func (l TextLine) Trimmed() string {
	return strings.TrimSpace(l.Raw)
}

// Blank reports whether the line has no content.
func (l TextLine) Blank() bool {
	return l.Trimmed() == ""
}

// Comment reports whether the line is a `#` or `;` comment.
func (l TextLine) Comment() bool {
	trimmed := l.Trimmed()

	return strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";")
}

// IsSectionHeader reports whether the line is an INI section header and, when
// it is, returns the trimmed name.
func (l TextLine) IsSectionHeader() (string, bool) {
	trimmed := l.Trimmed()
	if len(trimmed) < 2 || !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return "", false
	}

	return strings.TrimSpace(trimmed[1 : len(trimmed)-1]), true
}

// SplitKeyValue splits an `key = value` line on the first '='.
func (l TextLine) SplitKeyValue() (key, value string, ok bool) {
	key, value, ok = strings.Cut(l.Raw, "=")
	if !ok {
		return "", "", false
	}

	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

// Newline returns the line ending this file uses, defaulting to "\n".
func Newline(lines []TextLine) string {
	for _, line := range lines {
		if line.EOL != "" {
			return line.EOL
		}
	}

	return "\n"
}

// EnsureTrailingNewline gives the last line an ending when it lacks one. It is
// only needed when appending to a file that does not end with a newline; it
// touches formatting, never content.
func EnsureTrailingNewline(lines []TextLine) []TextLine {
	if len(lines) == 0 {
		return lines
	}

	if lines[len(lines)-1].EOL == "" {
		lines[len(lines)-1].EOL = Newline(lines[:len(lines)-1])
	}

	return lines
}
