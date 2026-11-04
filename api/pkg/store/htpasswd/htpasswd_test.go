package htpasswd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/mmhk/svn-admin/pkg/model"
)

// APR1 vectors generated with OpenSSL (same apr_md5_encode as Apache
// htpasswd -m): openssl passwd -apr1 -salt <salt> <password>.
func TestVerifyAPR1Vectors(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
	}{
		{"plain", "myPassword", "$apr1$r31.....$HqJZimcKQFAMYayBlzkrA/"},
		{"simple", "testpass", "$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z."},
		{"specials", "p@ss:w0rd!#$%&,", "$apr1$Zz9.aB3c$d.FskSwM1kQAKAXSZtmmH0"},
		{"spaces", " spaced pass ", "$apr1$xY12zAb9$hTjYxo8BdtlL1YgWpu1E.1"},
		{"empty password", "", "$apr1$00000000$6RcJ2QejwQuo6wYyqYPMQ1"},
		{"long password", "correct horse battery staple", "$apr1$12345678$5s8zqQNNXgdW9osfkSNGf0"},
		{"short password", "a", "$apr1$12345678$68ZQVfPkWX/wcXr/41VxQ."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Verify(test.password, test.hash); err != nil {
				t.Fatalf("Verify() error = %v", err)
			}

			if err := Verify(test.password+"x", test.hash); !errors.Is(err, model.ErrInvalidCredentials) {
				t.Fatalf("Verify(wrong) error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestVerifyCryptVectors(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
	}{
		{"md5crypt", "rasmuslerdorf", "$1$rasmusle$rISCgZzpwk3UhDidwXvin0"},
		{"sha256", "testpass", "$5$saltsalt$L9w162ClVuu/Ps66S1saSCvCOCyAfru379gq8pXCj27"},
		{"sha512", "testpass", "$6$saltsalt$UUTr6oVJTBk4azi0BmGqXvVgtbeWdSGG0b.P4dfxLVqqVmSL7RQXfoGFlx0.Jkt3Q2nsKyrc.M/cKE2XOCLN70"},
		{"sha1", "testpass", "{SHA}IGyAQTualsExLMNGt9JRe4RGPt0="},
		{"sha1 specials", "p@ss w0rd", "{SHA}hkh4dSUt9vfhE3nIRwDeOxmhD7w="},
		{"bcrypt 2y", "testpass", "$2y$10$rs14yDbmCWVK7s9ulpVDbuNMUfIPCbg/799ZcB7xJ8naxpoNs78Hy"},
		{"bcrypt 2y specials", "p@ss w0rd", "$2y$10$FR6VXWjNV1dJ2U9HPzm9euj91150Ga3lB4WwE3f4gI7MWLLuzUp.a"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Verify(test.password, test.hash); err != nil {
				t.Fatalf("Verify() error = %v", err)
			}

			if err := Verify(test.password+"x", test.hash); !errors.Is(err, model.ErrInvalidCredentials) {
				t.Fatalf("Verify(wrong) error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

// DES vectors generated with PHP crypt() (glibc crypt(3) DES).
func TestVerifyDESVectors(t *testing.T) {
	tests := []struct {
		password string
		hash     string
	}{
		{"abc", "aaMjTET7rKV4Y"},
		{"abcdefgh", "ZzrUgYb/Yc/Qs"},
		{"abcdefghi", "12cmzKaxhUaiY"},
		{"p@ss w0rd", "..qf5gIKeD1jY"},
		{"", "abmF1QH4PEr.E"},
		{"testpass", "zzHZPGkShzdp."},
		{"12345678", "H0BysyB9BBchs"},
	}

	for _, test := range tests {
		if err := Verify(test.password, test.hash); err != nil {
			t.Errorf("Verify(%q) error = %v", test.password, err)
		}

		// DES only uses the first eight characters, so the wrong password must
		// differ inside that prefix.
		if err := Verify("wrongpass", test.hash); !errors.Is(err, model.ErrInvalidCredentials) {
			t.Errorf("Verify(%q, wrong) error = %v, want ErrInvalidCredentials", test.password, err)
		}
	}
}

func TestVerifyBcryptGenerated(t *testing.T) {
	encoded, err := bcrypt.GenerateFromPassword([]byte("generated-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	if err := Verify("generated-password", string(encoded)); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if err := Verify("other", string(encoded)); !errors.Is(err, model.ErrInvalidCredentials) {
		t.Fatalf("Verify(wrong) error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyRejectsPlaintextAndUnknownFormats(t *testing.T) {
	for _, hash := range []string{
		"s3cr3t",
		"admin",
		"{SSHA}whatever",
		"$9$unsupported$hash",
		"_J9..ccKS.2S19T1PmB7m6pxZb9.",
	} {
		err := Verify("s3cr3t", hash)
		if err == nil {
			t.Errorf("Verify(%q) error = nil, want a rejection", hash)
		}

		if errors.Is(err, model.ErrInvalidCredentials) {
			t.Errorf("Verify(%q) reported bad credentials instead of an unsupported format", hash)
		}

		if strings.Contains(err.Error(), hash) {
			t.Errorf("error leaks the hash: %q", err.Error())
		}
	}
}

func TestParsePreservesRawLines(t *testing.T) {
	data := "# managed by hand\r\n\r\nalice:$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z.\r\nbob:{SHA}IGyAQTualsExLMNGt9JRe4RGPt0=\r\n; trailing comment\r\n"

	file := Parse("test", []byte(data))

	if got := file.Users(); len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Fatalf("Users() = %v", got)
	}

	if len(file.Lines) != 5 {
		t.Fatalf("Lines = %d, want 5", len(file.Lines))
	}

	if file.Lines[0].Raw != "# managed by hand" || file.Lines[0].EOL != "\r\n" {
		t.Errorf("comment line = %+v", file.Lines[0])
	}

	if file.Lines[2].Entry == nil || file.Lines[2].Entry.User != "alice" || file.Lines[2].Entry.Line != 3 {
		t.Errorf("alice line = %+v", file.Lines[2])
	}

	// Reassembling the preserved lines must reproduce the input byte for byte
	// so the P2 writer can be lossless.
	var rebuilt strings.Builder
	for _, line := range file.Lines {
		rebuilt.WriteString(line.Raw)
		rebuilt.WriteString(line.EOL)
	}

	if rebuilt.String() != data {
		t.Errorf("reassembled file differs:\n got %q\nwant %q", rebuilt.String(), data)
	}
}

func TestStoreVerifyPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svn-auth-file")

	content := "alice:$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z.\nbob:{SHA}IGyAQTualsExLMNGt9JRe4RGPt0=\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := New(path)

	if err := store.VerifyPassword(context.Background(), "alice", "testpass"); err != nil {
		t.Fatalf("VerifyPassword(alice) error = %v", err)
	}

	if err := store.VerifyPassword(context.Background(), "bob", "testpass"); err != nil {
		t.Fatalf("VerifyPassword(bob) error = %v", err)
	}

	for _, test := range []struct{ user, password string }{
		{"alice", "wrong"},
		{"nobody", "testpass"},
	} {
		if err := store.VerifyPassword(context.Background(), test.user, test.password); !errors.Is(err, model.ErrInvalidCredentials) {
			t.Errorf("VerifyPassword(%q) error = %v, want ErrInvalidCredentials", test.user, err)
		}
	}
}

func TestStoreMissingFileIsTypedError(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := store.ListUsers(context.Background(), model.PageQuery{})
	if err == nil {
		t.Fatal("ListUsers() error = nil, want a read error")
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *htpasswd.Error", err)
	}

	if typed.Op != "read" {
		t.Errorf("Op = %q, want read", typed.Op)
	}
}

func TestStoreListUsersNeverExposesHashes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svn-auth-file")
	content := "alice:$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z.\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := New(path)

	page, err := store.ListUsers(context.Background(), model.PageQuery{})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if page.Total != 1 || page.Items[0].Name != "alice" {
		t.Fatalf("page = %+v", page)
	}

	if strings.Contains(content, page.Items[0].Name+"$") || strings.Contains(page.Items[0].Comment, "$apr1$") {
		t.Error("user view appears to contain a hash")
	}

	if _, err := store.GetUser(context.Background(), "nobody"); !model.IsNotFound(err) {
		t.Errorf("GetUser(nobody) error = %v, want NotFoundError", err)
	}
}
