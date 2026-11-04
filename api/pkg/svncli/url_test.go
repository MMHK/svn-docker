package svncli

import "testing"

func TestFileURL(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/var/svnlocal/project", "file:///var/svnlocal/project"},
		{"/var/svn local/my repo#1", "file:///var/svn%20local/my%20repo%231"},
		{"/srv/資料/repo", "file:///srv/%E8%B3%87%E6%96%99/repo"},
		{`C:\srv\svn\repo`, "file:///C:/srv/svn/repo"},
		{"/var/svnlocal/project/", "file:///var/svnlocal/project"},
		{"", "file://"},
	}

	for _, test := range cases {
		if got := FileURL(test.path); got != test.want {
			t.Errorf("FileURL(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}

func TestJoinURL(t *testing.T) {
	cases := []struct {
		base     string
		segments []string
		want     string
	}{
		{"file:///svn/repo", []string{"trunk"}, "file:///svn/repo/trunk"},
		{"file:///svn/repo/", []string{"a b", "d#1"}, "file:///svn/repo/a%20b/d%231"},
		{"file:///svn/repo", []string{"proj", "trunk"}, "file:///svn/repo/proj/trunk"},
		{"file:///svn/repo", nil, "file:///svn/repo"},
	}

	for _, test := range cases {
		if got := JoinURL(test.base, test.segments...); got != test.want {
			t.Errorf("JoinURL(%q, %v) = %q, want %q", test.base, test.segments, got, test.want)
		}
	}
}
