package source

import (
	"os"
	"os/exec"
	"path/filepath"
)

// findCLI looks up a CLI tool by name, checking PATH first then common locations.
func findCLI(name string) (string, error) {
	// Try standard PATH first
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	// Try common locations
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "google-cloud-sdk", "bin", name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/opt/homebrew/bin", name),
		filepath.Join(home, ".local", "bin", name),
		filepath.Join(home, "bin", name),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}
