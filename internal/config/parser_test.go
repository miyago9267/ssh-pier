package config

import (
	"strings"
	"testing"
)

const testConfig = `# Global defaults
Host *
    ServerAliveInterval 60

# @group: company
Host prod-server
    Hostname 10.0.1.100
    User deploy
    Port 22
    IdentityFile ~/.ssh/id_company

Host staging
    Hostname 10.0.1.200
    User deploy

# @group: personal
Host my-vps
    Hostname 203.0.113.50
    User miyago
    Port 2222
    IdentityFile ~/.ssh/id_ed25519

Host raspberry
    Hostname 192.168.1.100
    User pi

# @group: ungrouped
Host orphan-box
    Hostname 172.16.0.5
    User admin
`

func TestParseHosts(t *testing.T) {
	hosts, err := Parse(strings.NewReader(testConfig))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Wildcard host should be excluded
	for _, h := range hosts {
		if h.Alias == "*" {
			t.Error("wildcard host (*) should be excluded from results")
		}
	}

	if len(hosts) != 5 {
		t.Fatalf("expected 5 hosts, got %d", len(hosts))
	}
}

func TestParseHostFields(t *testing.T) {
	hosts, _ := Parse(strings.NewReader(testConfig))

	prod := findHost(hosts, "prod-server")
	if prod == nil {
		t.Fatal("prod-server not found")
	}
	if prod.Hostname != "10.0.1.100" {
		t.Errorf("Hostname = %q, want %q", prod.Hostname, "10.0.1.100")
	}
	if prod.User != "deploy" {
		t.Errorf("User = %q, want %q", prod.User, "deploy")
	}
	if prod.Port != "22" {
		t.Errorf("Port = %q, want %q", prod.Port, "22")
	}
	if prod.IdentityFile != "~/.ssh/id_company" {
		t.Errorf("IdentityFile = %q, want %q", prod.IdentityFile, "~/.ssh/id_company")
	}
}

func TestParseGroups(t *testing.T) {
	hosts, _ := Parse(strings.NewReader(testConfig))

	tests := []struct {
		alias string
		group string
	}{
		{"prod-server", "company"},
		{"staging", "company"},
		{"my-vps", "personal"},
		{"raspberry", "personal"},
		{"orphan-box", "ungrouped"},
	}

	for _, tt := range tests {
		h := findHost(hosts, tt.alias)
		if h == nil {
			t.Errorf("host %q not found", tt.alias)
			continue
		}
		if h.Group != tt.group {
			t.Errorf("host %q group = %q, want %q", tt.alias, h.Group, tt.group)
		}
	}
}

func TestParseDefaultPort(t *testing.T) {
	hosts, _ := Parse(strings.NewReader(testConfig))

	staging := findHost(hosts, "staging")
	if staging == nil {
		t.Fatal("staging not found")
	}
	if staging.Port != "22" {
		t.Errorf("default port = %q, want %q", staging.Port, "22")
	}
}

func TestParseEmpty(t *testing.T) {
	hosts, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse empty failed: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseCommentsOnly(t *testing.T) {
	input := "# just a comment\n# another one\n"
	hosts, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse comments-only failed: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func findHost(hosts []Host, alias string) *Host {
	for i := range hosts {
		if hosts[i].Alias == alias {
			return &hosts[i]
		}
	}
	return nil
}
