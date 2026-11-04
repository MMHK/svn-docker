package svncli

import (
	"encoding/xml"
	"fmt"
	"time"
)

// commitXML is the <commit> element shared by `svn list --xml` and
// `svn info --xml`.
type commitXML struct {
	Revision int64  `xml:"revision,attr"`
	Author   string `xml:"author"`
	Date     string `xml:"date"`
}

// listDocument is `svn list --xml`.
type listDocument struct {
	XMLName xml.Name         `xml:"lists"`
	Lists   []listElementXML `xml:"list"`
}

type listElementXML struct {
	Path    string         `xml:"path,attr"`
	Entries []listEntryXML `xml:"entry"`
}

type listEntryXML struct {
	Kind   string    `xml:"kind,attr"`
	Name   string    `xml:"name"`
	Size   *int64    `xml:"size"`
	Commit commitXML `xml:"commit"`
}

// infoDocument is `svn info --xml`.
type infoDocument struct {
	XMLName xml.Name       `xml:"info"`
	Entries []infoEntryXML `xml:"entry"`
}

type infoEntryXML struct {
	Kind     string    `xml:"kind,attr"`
	Path     string    `xml:"path,attr"`
	Revision int64     `xml:"revision,attr"`
	URL      string    `xml:"url"`
	Commit   commitXML `xml:"commit"`
}

// logDocument is `svn log --xml --limit 1`.
type logDocument struct {
	XMLName xml.Name   `xml:"log"`
	Entries []logEntry `xml:"logentry"`
}

type logEntry struct {
	Revision int64  `xml:"revision,attr"`
	Author   string `xml:"author"`
	Date     string `xml:"date"`
}

// parseSVNDate converts Subversion's ISO-8601 timestamp, which may carry six
// fractional digits.
func parseSVNDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("svncli: parse date %q: %w", value, err)
	}

	return parsed.UTC(), nil
}
