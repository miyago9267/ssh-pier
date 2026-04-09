package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/miyago/ssh-pier/internal/config"
	"github.com/miyago/ssh-pier/internal/ssh"
	"github.com/miyago/ssh-pier/internal/ui"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find home dir: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(home, ".ssh", "config")
	hosts, err := config.ParseFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", configPath, err)
		os.Exit(1)
	}

	model := ui.NewModel(hosts, configPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	// If user selected a host to connect, exec ssh
	m := finalModel.(ui.Model)
	if alias := m.ConnectAlias(); alias != "" {
		fmt.Printf("Connecting to %s...\n", alias)
		if err := ssh.Connect(alias); err != nil {
			fmt.Fprintf(os.Stderr, "ssh error: %v\n", err)
			os.Exit(1)
		}
	}
}
