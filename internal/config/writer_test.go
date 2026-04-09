package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteRoundTrip(t *testing.T) {
	input := `# @group: company
Host prod-server
    Hostname 10.0.1.100
    User deploy
    Port 22
    IdentityFile ~/.ssh/id_company

# @group: personal
Host my-vps
    Hostname 203.0.113.50
    User miyago
    Port 2222
`

	hosts, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	if err := WriteFile(path, hosts); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Re-parse written file
	got, err := ParseFile(path)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	if len(got) != len(hosts) {
		t.Fatalf("round-trip: got %d hosts, want %d", len(got), len(hosts))
	}

	for i, h := range hosts {
		g := got[i]
		if g.Alias != h.Alias || g.Hostname != h.Hostname || g.User != h.User ||
			g.Port != h.Port || g.Group != h.Group {
			t.Errorf("host[%d] mismatch:\n  got:  %+v\n  want: %+v", i, g, h)
		}
	}
}

func TestWriteBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	// Write original file
	original := "# original content\nHost test\n    Hostname 1.2.3.4\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	hosts := []Host{{Alias: "new-host", Hostname: "5.6.7.8", User: "root", Port: "22", Group: "test"}}
	if err := WriteFile(path, hosts); err != nil {
		t.Fatal(err)
	}

	// Backup should exist
	bakPath := path + ".bak"
	bak, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(bak) != original {
		t.Errorf("backup content mismatch")
	}
}

func TestWriteGroupOrdering(t *testing.T) {
	hosts := []Host{
		{Alias: "a", Hostname: "1.1.1.1", Group: "alpha"},
		{Alias: "b", Hostname: "2.2.2.2", Group: "alpha"},
		{Alias: "c", Hostname: "3.3.3.3", Group: "beta"},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	if err := WriteFile(path, hosts); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(path)
	s := string(content)

	// Group annotations should appear
	if !strings.Contains(s, "# @group: alpha") {
		t.Error("missing @group: alpha annotation")
	}
	if !strings.Contains(s, "# @group: beta") {
		t.Error("missing @group: beta annotation")
	}

	// alpha should come before beta
	alphaIdx := strings.Index(s, "# @group: alpha")
	betaIdx := strings.Index(s, "# @group: beta")
	if alphaIdx > betaIdx {
		t.Error("alpha group should come before beta group")
	}
}
