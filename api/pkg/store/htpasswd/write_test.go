package htpasswd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

const writerFixture = "# managed by hand\r\n" +
	"\r\n" +
	"alice:$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z.\r\n" +
	"; keep this comment\r\n" +
	"bob:{SHA}IGyAQTualsExLMNGt9JRe4RGPt0=\r\n" +
	"carol:$2y$10$rs14yDbmCWVK7s9ulpVDbuNMUfIPCbg/799ZcB7xJ8naxpoNs78Hy\r\n"

func newWriterStore(t *testing.T, content string) (string, *Store, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "svn-auth-file")

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupDir := filepath.Join(dir, "backup")

	return path, New(path, WithBackupDir(backupDir)), backupDir
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	return string(data)
}

// assertLinesPreserved fails when any line of before (except the excluded
// ones) is missing from after, byte for byte. It is how the "non-target lines
// are identical" requirement is checked.
func assertLinesPreserved(t *testing.T, before, after string, excluded func(string) bool) {
	t.Helper()

	remaining := strings.Split(after, "\n")
	counts := make(map[string]int, len(remaining))

	for _, line := range remaining {
		counts[line]++
	}

	for _, line := range strings.Split(strings.TrimSuffix(before, "\n"), "\n") {
		if excluded != nil && excluded(line) {
			continue
		}

		if counts[line] == 0 {
			t.Fatalf("line was not preserved byte-for-byte: %q", line)
		}

		counts[line]--
	}
}

func TestHashAPR1MatchesOpenSSLVector(t *testing.T) {
	// Vector from `openssl passwd -apr1 -salt abcdefgh testpass`.
	if got := hashAPR1WithSalt("testpass", "abcdefgh"); got != "$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z." {
		t.Fatalf("hashAPR1WithSalt() = %q", got)
	}

	hash, err := HashAPR1("testpass")
	if err != nil {
		t.Fatalf("HashAPR1() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$apr1$") || len(hash) != len("$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z.") {
		t.Errorf("hash = %q, want the APR1 layout", hash)
	}

	if err := Verify("testpass", hash); err != nil {
		t.Errorf("generated hash does not verify: %v", err)
	}

	if err := Verify("wrong", hash); !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("generated hash accepts a wrong password: %v", err)
	}

	// A second call must use a different salt.
	other, err := HashAPR1("testpass")
	if err != nil {
		t.Fatalf("HashAPR1() second call error = %v", err)
	}

	if other == hash {
		t.Error("HashAPR1() reused the salt")
	}
}

func TestCreateAccountPreservesEveryOtherByte(t *testing.T) {
	path, store, backupDir := newWriterStore(t, writerFixture)
	ctx := context.Background()

	if err := store.CreateAccount(ctx, "dave", "s3cr3t-pass", model.WriteOptions{}); err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	content := readFile(t, path)

	assertLinesPreserved(t, writerFixture, content, nil)

	if !strings.HasSuffix(content, "dave:$apr1$") && !strings.Contains(content, "dave:$apr1$") {
		t.Errorf("new account line is missing or not APR1:\n%s", content)
	}

	// CRLF must survive: the appended line uses the file's ending.
	if strings.Count(content, "\r\n") != strings.Count(writerFixture, "\r\n")+1 {
		t.Errorf("CRLF count did not grow by one:\n%q", content)
	}

	if err := store.VerifyPassword(ctx, "dave", "s3cr3t-pass"); err != nil {
		t.Fatalf("VerifyPassword(new account) error = %v", err)
	}

	backups, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir(backup) error = %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("backups = %d, want 1", len(backups))
	}
}

func TestCreateAccountRejectsDuplicateAndWeakInput(t *testing.T) {
	_, store, _ := newWriterStore(t, writerFixture)
	ctx := context.Background()

	err := store.CreateAccount(ctx, "alice", "whatever", model.WriteOptions{})
	var conflict *model.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("CreateAccount(existing) error = %v, want ConflictError", err)
	}

	for _, name := range []string{"", "a b", "a:b", "a\nb", "*", ".leading", strings.Repeat("a", 65)} {
		if err := store.CreateAccount(ctx, name, "password", model.WriteOptions{}); !model.IsValidation(err) {
			t.Errorf("CreateAccount(%q) error = %v, want ValidationError", name, err)
		}
	}

	for _, password := range []string{"", "line\nbreak", "\x00", strings.Repeat("p", model.MaxPasswordLength+1)} {
		if err := store.CreateAccount(ctx, "erin", password, model.WriteOptions{}); !model.IsValidation(err) {
			t.Errorf("CreateAccount(password %q) error = %v, want ValidationError", password, err)
		}
	}
}

func TestSetPasswordOnlyTouchesTheTargetLine(t *testing.T) {
	path, store, _ := newWriterStore(t, writerFixture)
	ctx := context.Background()

	if err := store.SetPassword(ctx, "alice", "brand-new", model.WriteOptions{}); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}

	content := readFile(t, path)

	assertLinesPreserved(t, writerFixture, content, func(line string) bool {
		return strings.HasPrefix(line, "alice:")
	})

	if strings.Contains(content, "$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z.") {
		t.Error("the old alice hash is still present")
	}

	if !strings.HasPrefix(extractLine(t, content, "alice:"), "alice:$apr1$") {
		t.Errorf("alice line = %q, want an APR1 hash", extractLine(t, content, "alice:"))
	}

	if err := store.VerifyPassword(ctx, "alice", "brand-new"); err != nil {
		t.Fatalf("VerifyPassword(new) error = %v", err)
	}

	if err := store.VerifyPassword(ctx, "alice", "testpass"); !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("old password still verifies: %v", err)
	}

	// Untouched accounts keep their original hash format.
	if !strings.Contains(content, "bob:{SHA}IGyAQTualsExLMNGt9JRe4RGPt0=") {
		t.Error("bob's line was modified")
	}

	if err := store.SetPassword(ctx, "nobody", "x", model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("SetPassword(unknown) error = %v, want NotFoundError", err)
	}
}

func TestDeleteUserRemovesOnlyThatAccount(t *testing.T) {
	path, store, _ := newWriterStore(t, writerFixture)
	ctx := context.Background()

	if err := store.DeleteUser(ctx, "bob", model.WriteOptions{}); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	content := readFile(t, path)

	if strings.Contains(content, "bob:") {
		t.Error("bob's line survived the delete")
	}

	assertLinesPreserved(t, writerFixture, content, func(line string) bool {
		return strings.HasPrefix(line, "bob:")
	})

	if err := store.DeleteUser(ctx, "bob", model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("second DeleteUser() error = %v, want NotFoundError", err)
	}

	if err := store.VerifyPassword(ctx, "carol", "testpass"); err != nil {
		t.Errorf("carol no longer verifies after deleting bob: %v", err)
	}
}

func TestWriteHonoursExpectedHash(t *testing.T) {
	path, store, _ := newWriterStore(t, writerFixture)
	ctx := context.Background()

	file, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	_ = file

	stale := model.WriteOptions{ExpectedHash: strings.Repeat("0", 64)}

	if err := store.SetPassword(ctx, "alice", "newpass", stale); !model.IsConflict(err) {
		t.Fatalf("SetPassword(stale hash) error = %v, want ConflictError", err)
	}

	if readFile(t, path) != writerFixture {
		t.Error("a stale hash still modified the file")
	}
}

func TestBackupRotationKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svn-auth-file")

	if err := os.WriteFile(path, []byte(writerFixture), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupDir := filepath.Join(dir, "backup")
	store := New(path, WithBackupDir(backupDir), WithBackupKeep(2))
	ctx := context.Background()

	for _, name := range []string{"dave", "erin", "frank"} {
		if err := store.CreateAccount(ctx, name, "password", model.WriteOptions{}); err != nil {
			t.Fatalf("CreateAccount(%s) error = %v", name, err)
		}
	}

	backups, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("backups = %d, want 2", len(backups))
	}
}

func TestConcurrentCreatesKeepEveryAccount(t *testing.T) {
	_, store, _ := newWriterStore(t, writerFixture)
	ctx := context.Background()

	const count = 20

	var (
		wait   sync.WaitGroup
		mu     sync.Mutex
		failed []error
	)

	for i := 0; i < count; i++ {
		wait.Add(1)

		go func(index int) {
			defer wait.Done()

			name := "user" + string(rune('a'+index))
			if err := store.CreateAccount(ctx, name, "password", model.WriteOptions{}); err != nil {
				mu.Lock()
				failed = append(failed, err)
				mu.Unlock()
			}
		}(i)
	}

	wait.Wait()

	if len(failed) > 0 {
		t.Fatalf("concurrent creates failed: %v", failed)
	}

	file, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(file.Users()) != count+3 {
		t.Fatalf("users = %d, want %d", len(file.Users()), count+3)
	}

	for i := 0; i < count; i++ {
		name := "user" + string(rune('a'+i))
		if err := store.VerifyPassword(ctx, name, "password"); err != nil {
			t.Errorf("VerifyPassword(%s) error = %v", name, err)
		}
	}

	// The fixture accounts must still verify: no line was corrupted.
	for _, account := range []struct{ user, password string }{
		{"alice", "testpass"},
		{"bob", "testpass"},
		{"carol", "testpass"},
	} {
		if err := store.VerifyPassword(ctx, account.user, account.password); err != nil {
			t.Errorf("VerifyPassword(%s) error = %v", account.user, err)
		}
	}
}

func TestRealHtpasswdAgreesWhenAvailable(t *testing.T) {
	binary, err := exec.LookPath("htpasswd")
	if err != nil {
		t.Skip("htpasswd is not installed; the OpenSSL vector above covers the format")
	}

	path, store, _ := newWriterStore(t, writerFixture)

	if err := store.CreateAccount(context.Background(), "dave", "s3cr3t-pass", model.WriteOptions{}); err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	cmd := exec.Command(binary, "-vb", path, "dave", "s3cr3t-pass")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("htpasswd -vb rejected the generated hash: %v (%s)", err, output)
	}
}

func extractLine(t *testing.T, content, prefix string) string {
	t.Helper()

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSuffix(line, "\r")
		}
	}

	t.Fatalf("no line with prefix %q in %q", prefix, content)

	return ""
}
