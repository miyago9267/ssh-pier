package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	groupStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			PaddingLeft(1)

	hostStyle = lipgloss.NewStyle().
			PaddingLeft(3)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Width(14)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Padding(0, 1)
)
