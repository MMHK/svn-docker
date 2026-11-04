package presenter

import (
	"os/exec"
	"strings"
	"testing"
)

// TestPresenterDoesNotImportNetHTTP guards the MVP boundary from plan section
// 7.2 check 14: presenter (and everything it pulls in) must not import
// net/http.
func TestPresenterDoesNotImportNetHTTP(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool not available")
	}

	output, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "net/http" {
			t.Fatal("pkg/presenter depends on net/http; keep HTTP concerns in pkg/transport/httpapi")
		}
	}
}
