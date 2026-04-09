package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/miyago9267/ssh-pier/internal/source"
	"github.com/miyago9267/ssh-pier/internal/ui"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find home dir: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(home, ".ssh", "config")

	sources := []source.Source{
		&source.SSHSource{ConfigPath: configPath},
		&source.GCESource{},
		&source.GKESource{Shell: "/bin/sh"},
	}

	model := ui.NewModel(sources, configPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	m := finalModel.(ui.Model)
	s, t := m.ConnectResult()
	if s != nil && t != nil {
		fmt.Printf("Connecting to %s (%s)...\n", t.Alias, s.Name())
		if err := s.Connect(*t); err != nil {
			fmt.Fprintf(os.Stderr, "connect error: %v\n", err)
			os.Exit(1)
		}
	}
}
