package source

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/miyago9267/ssh-pier/internal/config"
)

type SSHSource struct {
	ConfigPath string
}

func (s *SSHSource) Name() string { return "SSH" }

func (s *SSHSource) Fetch() ([]Target, error) {
	hosts, err := config.ParseFile(s.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("parse ssh config: %w", err)
	}

	targets := make([]Target, len(hosts))
	for i, h := range hosts {
		detail := h.User + "@" + h.Hostname
		if h.Port != "" && h.Port != "22" {
			detail += ":" + h.Port
		}
		targets[i] = Target{
			Source:   "ssh",
			Alias:    h.Alias,
			Group:    h.Group,
			Detail:   detail,
			Editable: true,
			Meta: map[string]string{
				"hostname":     h.Hostname,
				"user":         h.User,
				"port":         h.Port,
				"identityFile": h.IdentityFile,
			},
		}
	}
	return targets, nil
}

func (s *SSHSource) Connect(t Target) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found: %w", err)
	}
	return syscall.Exec(sshPath, []string{"ssh", t.Alias}, os.Environ())
}

// HostFromTarget converts a Target back to a config.Host for editing.
func HostFromTarget(t Target) config.Host {
	return config.Host{
		Alias:        t.Alias,
		Hostname:     t.Meta["hostname"],
		User:         t.Meta["user"],
		Port:         t.Meta["port"],
		IdentityFile: t.Meta["identityFile"],
		Group:        t.Group,
	}
}
