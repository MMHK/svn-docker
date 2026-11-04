package svncli

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Entry is one child of a directory listing.
type Entry struct {
	Name     string
	Kind     string
	Size     int64
	Revision int64
	Author   string
	Date     time.Time
}

// Listing is one `svn list --xml` result.
type Listing struct {
	Path    string
	Entries []Entry
}

// Info is the subset of `svn info --xml` the admin UI needs.
type Info struct {
	URL               string
	Revision          int64
	LastChangedRev    int64
	LastChangedAuthor string
	LastChangedAt     time.Time
}

// List runs `svn list --xml` against target. revision <= 0 means HEAD.
func (c *Client) List(ctx context.Context, target string, revision int64) (Listing, error) {
	args := []string{"list", "--xml", "--non-interactive"}

	if revision > 0 {
		args = append(args, "-r", strconv.FormatInt(revision, 10))
	}

	args = append(args, target)

	stdout, err := c.run(ctx, "list", c.timeout, c.svn, args...)
	if err != nil {
		return Listing{}, err
	}

	return parseListing(stdout)
}

// HeadRevision resolves the youngest revision of a repository. The entry
// revision returned by `svn info` is the last change to that node, not the
// repository HEAD, so HEAD comes from `svn log -r HEAD --limit 1` (plan T4.6).
// An empty repository (no commits yet) has HEAD 0.
func (c *Client) HeadRevision(ctx context.Context, target string) (int64, error) {
	stdout, err := c.run(
		ctx, "log", c.timeout, c.svn,
		"log", "--xml", "--limit", "1", "-r", "HEAD", "--non-interactive", target,
	)
	if err != nil {
		var typed *Error
		if errors.As(err, &typed) {
			switch typed.Code {
			case "E160006", "E195002":
				// "No such revision" for a repository with no commits.
				return 0, nil
			}
		}

		return 0, err
	}

	var document logDocument
	if err := xml.Unmarshal(stdout, &document); err != nil {
		return 0, fmt.Errorf("svncli: parse log xml: %w", err)
	}

	if len(document.Entries) == 0 {
		return 0, nil
	}

	return document.Entries[0].Revision, nil
}

// Info runs `svn info --xml` against target.
func (c *Client) Info(ctx context.Context, target string) (Info, error) {
	stdout, err := c.run(ctx, "info", c.timeout, c.svn, "info", "--xml", "--non-interactive", target)
	if err != nil {
		return Info{}, err
	}

	return parseInfo(stdout)
}

// Create runs `svnadmin create --fs-type fsfs path`.
//
// `--non-interactive` is deliberately not passed here: svnadmin create never
// prompts for credentials and some redistributed builds (TortoiseSVN's 1.14,
// for example) reject the flag. Every `svn` invocation does pass it, which is
// where an interactive auth prompt could actually happen.
func (c *Client) Create(ctx context.Context, path string) error {
	_, err := c.run(
		ctx, "create", c.longTimeout, c.svnAdmin,
		"create", "--fs-type", "fsfs", path,
	)

	return err
}

// MkDirs creates the standard layout directories below target with one
// `svn mkdir --parents` commit. An empty dirs slice is a no-op.
func (c *Client) MkDirs(ctx context.Context, target string, dirs []string) error {
	if len(dirs) == 0 {
		return nil
	}

	args := make([]string, 0, len(dirs)+5)
	args = append(args, "mkdir", "--parents", "--non-interactive", "-m", "create standard layout")

	for _, dir := range dirs {
		args = append(args, JoinURL(target, strings.Split(dir, "/")...))
	}

	_, err := c.run(ctx, "mkdir", c.longTimeout, c.svn, args...)

	return err
}

func parseListing(payload []byte) (Listing, error) {
	var document listDocument
	if err := xml.Unmarshal(payload, &document); err != nil {
		return Listing{}, fmt.Errorf("svncli: parse list xml: %w", err)
	}

	listing := Listing{}

	if len(document.Lists) > 0 {
		listing.Path = document.Lists[0].Path

		for _, entry := range document.Lists[0].Entries {
			date, err := parseSVNDate(entry.Commit.Date)
			if err != nil {
				return Listing{}, err
			}

			item := Entry{
				Name:     entry.Name,
				Kind:     entry.Kind,
				Revision: entry.Commit.Revision,
				Author:   entry.Commit.Author,
				Date:     date,
			}

			if entry.Size != nil {
				item.Size = *entry.Size
			}

			listing.Entries = append(listing.Entries, item)
		}
	}

	return listing, nil
}

func parseInfo(payload []byte) (Info, error) {
	var document infoDocument
	if err := xml.Unmarshal(payload, &document); err != nil {
		return Info{}, fmt.Errorf("svncli: parse info xml: %w", err)
	}

	if len(document.Entries) == 0 {
		return Info{}, errors.New("svncli: info xml contains no entry")
	}

	entry := document.Entries[0]

	date, err := parseSVNDate(entry.Commit.Date)
	if err != nil {
		return Info{}, err
	}

	return Info{
		URL:               entry.URL,
		Revision:          entry.Revision,
		LastChangedRev:    entry.Commit.Revision,
		LastChangedAuthor: entry.Commit.Author,
		LastChangedAt:     date,
	}, nil
}
