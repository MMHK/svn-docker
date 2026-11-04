package svncli

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordRunner records argv and returns canned output.
type recordRunner struct {
	mu     sync.Mutex
	calls  []call
	stdout []byte
	stderr []byte
	err    error
}

type call struct {
	name string
	args []string
}

func (r *recordRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, call{name: name, args: append([]string(nil), args...)})

	return r.stdout, r.stderr, r.err
}

func (r *recordRunner) last(t *testing.T) call {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.calls) == 0 {
		t.Fatal("no command was executed")
	}

	return r.calls[len(r.calls)-1]
}

const listXMLFixture = `<?xml version="1.0" encoding="UTF-8"?>
<lists>
<list path="file:///svn/repo">
<entry kind="dir">
<name>trunk</name>
<commit revision="4">
<author>mixmedia</author>
<date>2026-11-04T10:00:00.000000Z</date>
</commit>
</entry>
<entry kind="file">
<name>a &amp; b.txt</name>
<size>12</size>
<commit revision="3">
<author>viewer</author>
<date>2026-11-03T09:30:00.000000Z</date>
</commit>
</entry>
</list>
</lists>`

const infoXMLFixture = `<?xml version="1.0" encoding="UTF-8"?>
<info>
<entry kind="dir" path="." revision="4">
<url>file:///svn/repo</url>
<commit revision="4">
<author>mixmedia</author>
<date>2026-11-04T10:00:00.000000Z</date>
</commit>
</entry>
</info>`

const logXMLFixture = `<?xml version="1.0" encoding="UTF-8"?>
<log>
<logentry revision="9">
<author>mixmedia</author>
<date>2026-11-04T11:00:00.000000Z</date>
</logentry>
</log>`

func TestListArguments(t *testing.T) {
	runner := &recordRunner{stdout: []byte(listXMLFixture)}
	client := New(Options{Runner: runner})

	listing, err := client.List(context.Background(), "file:///svn/repo", 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	got := runner.last(t)
	want := []string{"list", "--xml", "--non-interactive", "file:///svn/repo"}
	if got.name != "svn" || strings.Join(got.args, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %s %v, want svn %v", got.name, got.args, want)
	}

	if len(listing.Entries) != 2 {
		t.Fatalf("entries = %+v", listing.Entries)
	}

	// XML entities are decoded and sizes are kept for files only.
	if listing.Entries[1].Name != "a & b.txt" || listing.Entries[1].Size != 12 {
		t.Errorf("file entry = %+v", listing.Entries[1])
	}

	if listing.Entries[0].Kind != "dir" || listing.Entries[0].Size != 0 {
		t.Errorf("dir entry = %+v", listing.Entries[0])
	}

	if listing.Entries[0].Date.UTC().Format(time.RFC3339) != "2026-11-04T10:00:00Z" {
		t.Errorf("date = %v", listing.Entries[0].Date)
	}
}

func TestListWithRevision(t *testing.T) {
	runner := &recordRunner{stdout: []byte(listXMLFixture)}
	client := New(Options{Runner: runner})

	if _, err := client.List(context.Background(), "file:///svn/repo", 3); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	got := runner.last(t)
	want := "list --xml --non-interactive -r 3 file:///svn/repo"
	if strings.Join(got.args, " ") != want {
		t.Errorf("args = %v, want %s", got.args, want)
	}
}

func TestListEmptyDirectory(t *testing.T) {
	runner := &recordRunner{stdout: []byte(`<?xml version="1.0"?><lists><list path="file:///svn/repo"/></lists>`)}
	client := New(Options{Runner: runner})

	listing, err := client.List(context.Background(), "file:///svn/repo", 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(listing.Entries) != 0 || listing.Path != "file:///svn/repo" {
		t.Errorf("listing = %+v", listing)
	}
}

func TestInfoAndHeadRevision(t *testing.T) {
	runner := &recordRunner{stdout: []byte(infoXMLFixture)}
	client := New(Options{Runner: runner})

	info, err := client.Info(context.Background(), "file:///svn/repo")
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.LastChangedRev != 4 || info.LastChangedAuthor != "mixmedia" || info.URL != "file:///svn/repo" {
		t.Errorf("info = %+v", info)
	}

	if got := runner.last(t); got.name != "svn" || got.args[0] != "info" {
		t.Errorf("argv = %s %v", got.name, got.args)
	}

	runner.stdout = []byte(logXMLFixture)

	head, err := client.HeadRevision(context.Background(), "file:///svn/repo")
	if err != nil {
		t.Fatalf("HeadRevision() error = %v", err)
	}

	if head != 9 {
		t.Errorf("head = %d, want 9", head)
	}

	last := runner.last(t)
	if last.args[0] != "log" || !strings.Contains(strings.Join(last.args, " "), "-r HEAD") {
		t.Errorf("log argv = %v", last.args)
	}
}

func TestHeadRevisionEmptyRepository(t *testing.T) {
	empty := &recordRunner{
		stderr: []byte("svn: E160006: No such revision 0\n"),
		err:    errors.New("exit status 1"),
	}

	if head, err := New(Options{Runner: empty}).HeadRevision(context.Background(), "file:///svn/repo"); err != nil || head != 0 {
		t.Errorf("HeadRevision() = %d, %v; want 0, nil", head, err)
	}

	// A failure without an SVN code stays a failure.
	broken := &recordRunner{
		stderr: []byte("svn: unable to open repository\n"),
		err:    errors.New("exit status 1"),
	}

	if _, err := New(Options{Runner: broken}).HeadRevision(context.Background(), "file:///svn/repo"); err == nil {
		t.Error("HeadRevision() error = nil, want a failure")
	}
}

func TestCreateArguments(t *testing.T) {
	runner := &recordRunner{}
	client := New(Options{Runner: runner})

	if err := client.Create(context.Background(), "/var/svnlocal/repo"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := runner.last(t)
	want := []string{"create", "--fs-type", "fsfs", "/var/svnlocal/repo"}
	if got.name != "svnadmin" || strings.Join(got.args, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %s %v, want svnadmin %v", got.name, got.args, want)
	}
}

func TestMkDirsArguments(t *testing.T) {
	runner := &recordRunner{}
	client := New(Options{Runner: runner})

	if err := client.MkDirs(context.Background(), "file:///svn/repo", []string{"proj/trunk", "proj/branches"}); err != nil {
		t.Fatalf("MkDirs() error = %v", err)
	}

	got := runner.last(t)
	if got.name != "svn" {
		t.Errorf("command = %q, want svn", got.name)
	}

	want := []string{
		"mkdir", "--parents", "--non-interactive", "-m", "create standard layout",
		"file:///svn/repo/proj/trunk",
		"file:///svn/repo/proj/branches",
	}
	if strings.Join(got.args, " ") != strings.Join(want, " ") {
		t.Errorf("args = %v, want %v", got.args, want)
	}

	// An empty layout does not spawn a process.
	before := len(runner.calls)
	if err := client.MkDirs(context.Background(), "file:///svn/repo", nil); err != nil {
		t.Fatalf("MkDirs(nil) error = %v", err)
	}

	if len(runner.calls) != before {
		t.Errorf("MkDirs(nil) executed a command")
	}
}

func TestCommandFailureIsTypedAndNeverLeaksStderr(t *testing.T) {
	const stderrText = "svn: E160013: File not found: /secret/location"

	runner := &recordRunner{
		stderr: []byte(stderrText),
		err:    errors.New("exit status 1"),
	}
	client := New(Options{Runner: runner})

	_, err := client.List(context.Background(), "file:///svn/repo/missing", 0)
	if err == nil {
		t.Fatal("List() error = nil, want a failure")
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *svncli.Error", err)
	}

	if typed.Code != "E160013" || !IsNotFound(err) {
		t.Errorf("typed error = %+v", typed)
	}

	if strings.Contains(err.Error(), stderrText) || strings.Contains(err.Error(), "secret") {
		t.Errorf("error message leaks stderr: %q", err.Error())
	}

	if strings.Contains(typed.Error(), "E160013") {
		t.Errorf("typed message leaks the svn diagnostic: %q", typed.Error())
	}
}

// TestIsNotFoundClassifiesEveryDiagnostic covers the P4 leftover: the Windows
// CLI prints a W160013 warning before the E200009 error, so both must be
// scanned, with the E series taking precedence.
func TestIsNotFoundClassifiesEveryDiagnostic(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "windows warning only",
			stderr: "svn: warning: W160013: path not found\n",
			want:   true,
		},
		{
			name:   "windows warning then error",
			stderr: "svn: warning: W160013: path not found\nsvn: E200009: Could not list all targets\n",
			want:   true,
		},
		{
			name:   "unix error only",
			stderr: "svn: E160013: File not found\n",
			want:   true,
		},
		{
			name:   "no code at all",
			stderr: "svn: unable to open repository\n",
			want:   false,
		},
		{
			name:   "warning plus unrelated error",
			stderr: "svn: warning: W160013: path not found\nsvn: E170013: Unable to connect\n",
			want:   false,
		},
		{
			name:   "unrelated error only",
			stderr: "svn: E200033: Another process is blocking\n",
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &recordRunner{
				stderr: []byte(tc.stderr),
				err:    errors.New("exit status 1"),
			}

			_, err := New(Options{Runner: runner}).List(context.Background(), "file:///svn/repo/x", 0)
			if err == nil {
				t.Fatal("List() error = nil, want a failure")
			}

			if got := IsNotFound(err); got != tc.want {
				t.Errorf("IsNotFound() = %v, want %v (stderr %q)", got, tc.want, tc.stderr)
			}
		})
	}
}

// TestErrorKeepsEveryCode checks the raw classification: the primary code is
// the first error code, and every warning stays available for the mapper.
func TestErrorKeepsEveryCode(t *testing.T) {
	runner := &recordRunner{
		stderr: []byte("svn: warning: W160013: x\nsvn: E200009: y\nsvn: W155010: z\n"),
		err:    errors.New("exit status 1"),
	}

	_, err := New(Options{Runner: runner}).List(context.Background(), "file:///svn/repo/x", 0)
	if err == nil {
		t.Fatal("List() error = nil, want a failure")
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *svncli.Error", err)
	}

	want := []string{"W160013", "E200009", "W155010"}
	if strings.Join(typed.Codes, ",") != strings.Join(want, ",") {
		t.Errorf("Codes = %v, want %v", typed.Codes, want)
	}

	if typed.Code != "E200009" {
		t.Errorf("Code = %q, want the first E series code E200009", typed.Code)
	}
}

// blockingRunner waits for the deadline so the timeout branch is exercised.
type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, _ string, _ ...string) ([]byte, []byte, error) {
	<-ctx.Done()

	return nil, []byte("svn: E170013: timed out"), ctx.Err()
}

func TestTimeoutIsReported(t *testing.T) {
	client := New(Options{Runner: blockingRunner{}, Timeout: 5 * time.Millisecond})

	_, err := client.List(context.Background(), "file:///svn/repo", 0)
	if err == nil {
		t.Fatal("List() error = nil, want a timeout")
	}

	var typed *Error
	if !errors.As(err, &typed) || !typed.TimedOut {
		t.Fatalf("error = %v (%T), want a timed out *svncli.Error", err, err)
	}
}
