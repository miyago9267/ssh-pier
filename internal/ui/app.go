package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/miyago9267/ssh-pier/internal/config"
)

type mode int

const (
	modeList mode = iota
	modeSearch
	modeEdit
	modeNew
	modeConfirmDelete
)

type Model struct {
	hosts      []config.Host
	configPath string

	// List state
	cursor     int
	flatItems  []listItem // flattened view (groups + hosts)
	collapsed  map[string]bool

	// Search
	searchInput textinput.Model
	searchQuery string

	// Edit / New
	editFields []textinput.Model
	editCursor int
	editHost   *config.Host // nil = new host

	// UI state
	mode    mode
	width   int
	height  int
	status  string

	// Connection
	connectAlias string
}

type listItem struct {
	isGroup bool
	group   string
	host    *config.Host
}

func NewModel(hosts []config.Host, configPath string) Model {
	si := textinput.New()
	si.Placeholder = "Search hosts..."
	si.CharLimit = 64

	m := Model{
		hosts:      hosts,
		configPath: configPath,
		collapsed:  make(map[string]bool),
		searchInput: si,
	}
	m.rebuildList()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeSearch:
		return m.updateSearch(msg)
	case modeEdit, modeNew:
		return m.updateEdit(msg)
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	default:
		return m.updateList(msg)
	}
}

func (m Model) View() string {
	switch m.mode {
	case modeSearch:
		return m.viewSearch()
	case modeEdit, modeNew:
		return m.viewEdit()
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	default:
		return m.viewList()
	}
}

// --- List Mode ---

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "enter":
			if item := m.selectedItem(); item != nil {
				if item.isGroup {
					m.collapsed[item.group] = !m.collapsed[item.group]
					m.rebuildList()
				} else if item.host != nil {
					m.connectAlias = item.host.Alias
					return m, tea.Quit
				}
			}
		case " ":
			// Toggle group with space too
			if item := m.selectedItem(); item != nil && item.isGroup {
				m.collapsed[item.group] = !m.collapsed[item.group]
				m.rebuildList()
			}
		case "/":
			m.mode = modeSearch
			m.searchInput.Focus()
			m.searchQuery = ""
			m.searchInput.SetValue("")
			return m, textinput.Blink
		case "e":
			if item := m.selectedItem(); item != nil && item.host != nil {
				m.startEdit(item.host)
			}
		case "n":
			m.startNew()
		case "d":
			if item := m.selectedItem(); item != nil && item.host != nil {
				m.mode = modeConfirmDelete
			}
		}
	}
	return m, nil
}

func (m Model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Pier"))
	b.WriteString("\n")

	for i, item := range m.flatItems {
		selected := i == m.cursor
		if item.isGroup {
			arrow := "v"
			if m.collapsed[item.group] {
				arrow = ">"
			}
			line := fmt.Sprintf("%s %s", arrow, item.group)
			if selected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(groupStyle.Render(line))
			}
		} else {
			h := item.host
			line := fmt.Sprintf("  %s  %s@%s", h.Alias, h.User, h.Hostname)
			if h.Port != "" && h.Port != "22" {
				line += fmt.Sprintf(":%s", h.Port)
			}
			if selected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(hostStyle.Render(line))
			}
		}
		b.WriteString("\n")
	}

	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.status))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: connect/toggle  /: search  e: edit  n: new  d: delete  q: quit"))

	return b.String()
}

// --- Search Mode ---

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.mode = modeList
			m.searchInput.Blur()
			return m, nil
		case "esc":
			m.mode = modeList
			m.searchQuery = ""
			m.searchInput.Blur()
			m.rebuildList()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchQuery = m.searchInput.Value()
	m.rebuildList()
	m.cursor = 0
	return m, cmd
}

func (m Model) viewSearch() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Pier"))
	b.WriteString("\n")
	b.WriteString(searchStyle.Render("Search: "))
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	for i, item := range m.flatItems {
		selected := i == m.cursor
		if item.isGroup {
			line := fmt.Sprintf("v %s", item.group)
			if selected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(groupStyle.Render(line))
			}
		} else {
			h := item.host
			line := fmt.Sprintf("  %s  %s@%s", h.Alias, h.User, h.Hostname)
			if selected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(hostStyle.Render(line))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: confirm  esc: cancel"))

	return b.String()
}

// --- Edit Mode ---

func (m *Model) startEdit(h *config.Host) {
	m.mode = modeEdit
	m.editHost = h
	m.editCursor = 0
	m.editFields = makeEditFields(h)
	m.editFields[0].Focus()
}

func (m *Model) startNew() {
	m.mode = modeNew
	m.editHost = nil
	m.editCursor = 0
	m.editFields = makeEditFields(&config.Host{Port: "22", Group: "ungrouped"})
	m.editFields[0].Focus()
}

func makeEditFields(h *config.Host) []textinput.Model {
	labels := []string{"Alias", "Hostname", "User", "Port", "IdentityFile", "Group"}
	values := []string{h.Alias, h.Hostname, h.User, h.Port, h.IdentityFile, h.Group}

	fields := make([]textinput.Model, len(labels))
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.SetValue(values[i])
		ti.CharLimit = 128
		fields[i] = ti
	}
	return fields
}

func (m Model) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.mode = modeList
			return m, nil
		case "tab", "down":
			m.editFields[m.editCursor].Blur()
			m.editCursor = (m.editCursor + 1) % len(m.editFields)
			m.editFields[m.editCursor].Focus()
			return m, textinput.Blink
		case "shift+tab", "up":
			m.editFields[m.editCursor].Blur()
			m.editCursor = (m.editCursor - 1 + len(m.editFields)) % len(m.editFields)
			m.editFields[m.editCursor].Focus()
			return m, textinput.Blink
		case "enter":
			return m.saveEdit()
		}
	}

	var cmd tea.Cmd
	m.editFields[m.editCursor], cmd = m.editFields[m.editCursor].Update(msg)
	return m, cmd
}

func (m Model) saveEdit() (tea.Model, tea.Cmd) {
	h := config.Host{
		Alias:        m.editFields[0].Value(),
		Hostname:     m.editFields[1].Value(),
		User:         m.editFields[2].Value(),
		Port:         m.editFields[3].Value(),
		IdentityFile: m.editFields[4].Value(),
		Group:        m.editFields[5].Value(),
	}

	if h.Alias == "" || h.Hostname == "" {
		m.status = "Alias and Hostname are required"
		return m, nil
	}

	if m.mode == modeEdit && m.editHost != nil {
		// Remove old entry if alias changed
		if m.editHost.Alias != h.Alias {
			m.hosts = config.DeleteHost(m.hosts, m.editHost.Alias)
		}
	}

	m.hosts = config.UpdateHost(m.hosts, h)

	if err := config.WriteFile(m.configPath, m.hosts); err != nil {
		m.status = fmt.Sprintf("Write error: %v", err)
	} else {
		m.status = fmt.Sprintf("Saved: %s", h.Alias)
	}

	m.mode = modeList
	m.rebuildList()
	return m, nil
}

func (m Model) viewEdit() string {
	var b strings.Builder

	title := "Edit Host"
	if m.mode == modeNew {
		title = "New Host"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	labels := []string{"Alias", "Hostname", "User", "Port", "IdentityFile", "Group"}
	for i, f := range m.editFields {
		label := detailLabelStyle.Render(labels[i] + ":")
		b.WriteString(label)
		b.WriteString(f.View())
		b.WriteString("\n")
	}

	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.status))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("tab/shift+tab: navigate  enter: save  esc: cancel"))

	return b.String()
}

// --- Confirm Delete ---

func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			if item := m.selectedItem(); item != nil && item.host != nil {
				alias := item.host.Alias
				m.hosts = config.DeleteHost(m.hosts, alias)
				if err := config.WriteFile(m.configPath, m.hosts); err != nil {
					m.status = fmt.Sprintf("Write error: %v", err)
				} else {
					m.status = fmt.Sprintf("Deleted: %s", alias)
				}
				m.rebuildList()
				if m.cursor >= len(m.flatItems) {
					m.cursor = len(m.flatItems) - 1
				}
			}
			m.mode = modeList
		case "n", "N", "esc":
			m.mode = modeList
		}
	}
	return m, nil
}

func (m Model) viewConfirmDelete() string {
	var b strings.Builder
	b.WriteString(m.viewList())
	b.WriteString("\n")

	alias := ""
	if item := m.selectedItem(); item != nil && item.host != nil {
		alias = item.host.Alias
	}

	b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/N)", alias)))
	return b.String()
}

// --- Helpers ---

func (m *Model) rebuildList() {
	filtered := m.hosts
	if m.searchQuery != "" {
		filtered = config.FilterHosts(m.hosts, m.searchQuery)
	}

	groups := config.GroupHosts(filtered)
	m.flatItems = nil

	for _, g := range groups {
		m.flatItems = append(m.flatItems, listItem{isGroup: true, group: g.Name})
		if !m.collapsed[g.Name] {
			for i := range g.Hosts {
				m.flatItems = append(m.flatItems, listItem{host: &g.Hosts[i]})
			}
		}
	}
}

func (m *Model) moveCursor(delta int) {
	if len(m.flatItems) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.flatItems) {
		m.cursor = len(m.flatItems) - 1
	}
}

func (m Model) selectedItem() *listItem {
	if m.cursor < 0 || m.cursor >= len(m.flatItems) {
		return nil
	}
	return &m.flatItems[m.cursor]
}

// ConnectAlias returns the alias to connect to (empty if none selected).
func (m Model) ConnectAlias() string {
	return m.connectAlias
}

// Hosts returns the current host list.
func (m Model) Hosts() []config.Host {
	return m.hosts
}

// SetSize sets the terminal dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// ensure Model implements tea.Model
var _ tea.Model = Model{}
// lipgloss import used for styles
var _ = lipgloss.NewStyle
