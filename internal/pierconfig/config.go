package pierconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	GCE GCEConfig `json:"gce"`
	GKE GKEConfig `json:"gke"`
}

type GCEConfig struct {
	Projects []string `json:"projects"` // empty = auto-discover
}

type GKEConfig struct {
	Shell string `json:"shell"` // default "/bin/sh"
}

// DefaultPath returns ~/.config/pier/config.json
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pier", "config.json")
}

// Load reads config from path. Returns zero config if file doesn't exist.
func Load(path string) Config {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}
