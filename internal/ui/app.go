package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/miyago9267/ssh-pier/internal/config"
	"github.com/miyago9267/ssh-pier/internal/source"
)

type mode int

const (
	modeList mode = iota
	modeSearch
	modeEdit
	modeNew
	modeConfirmDelete
	modeShellInput
)

// tabState holds per-tab list state.
type tabState struct {
	targets      []source.Target
	cursor       int
	scrollOffset int
	flatItems    []listItem
	collapsed    map[string]bool
	fetched      bool
}

type Model struct {
	sources    []source.Source
	configPath string

	// Tab
	activeTab int
	tabs      []tabState

	// Search
	searchInput textinput.Model
	searchQuery string

	// Edit / New (SSH only)
	editFields []textinput.Model
	editCursor int
	editTarget *source.Target

	// Shell input (GKE only)
	shellInput textinput.Model

	// UI state
	mode   mode
	width  int
	height int
	status string

	// Connection result
	connectSource source.Source
	connectTarget *source.Target
}

type listItem struct {
	isGroup bool
	group   string
	target  *source.Target
}

// fetchDoneMsg signals that a source fetch completed.
type fetchDoneMsg struct {
	tabIdx  int
	targets []source.Target
	err     error
}

func NewModel(sources []source.Source, configPath string) Model {
	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 64

	shi := textinput.New()
	shi.Placeholder = "/bin/sh"
	shi.CharLimit = 64

	tabs := make([]tabState, len(sources))
	for i := range tabs {
		tabs[i].collapsed = make(map[string]bool)
	}

	return Model{
		sources:     sources,
		configPath:  configPath,
		tabs:        tabs,
		searchInput: si,
		shellInput:  shi,
	}
}

func (m Model) Init() tea.Cmd {
	// Fetch the first tab on startup
	return m.fetchTab(0)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case fetchDoneMsg:
		return m.handleFetchDone(msg)
	}

	switch m.mode {
	case modeSearch:
		return m.updateSearch(msg)
	case modeEdit, modeNew:
		return m.updateEdit(msg)
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	case modeShellInput:
		return m.updateShellInput(msg)
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
	case modeShellInput:
		return m.viewShellInput()
	default:
		return m.viewList()
	}
}

// --- Tab helpers ---

func (m Model) tab() *tabState {
	return &m.tabs[m.activeTab]
}

func (m Model) currentSource() source.Source {
	return m.sources[m.activeTab]
}

func (m Model) fetchTab(idx int) tea.Cmd {
	s := m.sources[idx]
	return func() tea.Msg {
		targets, err := s.Fetch()
		return fetchDoneMsg{tabIdx: idx, targets: targets, err: err}
	}
}

func (m Model) handleFetchDone(msg fetchDoneMsg) (tea.Model, tea.Cmd) {
	if msg.tabIdx < 0 || msg.tabIdx >= len(m.tabs) {
		return m, nil
	}
	tab := &m.tabs[msg.tabIdx]
	tab.fetched = true
	if msg.err != nil {
		m.status = fmt.Sprintf("%s: %v", m.sources[msg.tabIdx].Name(), msg.err)
		tab.targets = nil
	} else {
		tab.targets = msg.targets
		m.status = fmt.Sprintf("%s: %d targets loaded", m.sources[msg.tabIdx].Name(), len(msg.targets))
	}
	m.rebuildList()
	return m, nil
}

// --- List Mode ---

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "d":
			return m.switchTab(1)
		case "shift+tab", "a":
			return m.switchTab(-1)
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "enter":
			tab := m.tab()
			if item := m.selectedItem(tab); item != nil {
				if item.isGroup {
					tab.collapsed[item.group] = !tab.collapsed[item.group]
					m.rebuildList()
				} else if item.target != nil {
					m.connectSource = m.currentSource()
					m.connectTarget = item.target
					return m, tea.Quit
				}
			}
		case " ":
			tab := m.tab()
			if item := m.selectedItem(tab); item != nil && item.isGroup {
				tab.collapsed[item.group] = !tab.collapsed[item.group]
				m.rebuildList()
			}
		case "/":
			m.mode = modeSearch
			m.searchInput.Focus()
			m.searchQuery = ""
			m.searchInput.SetValue("")
			return m, textinput.Blink
		case "e":
			if m.currentSource().Name() == "SSH" {
				tab := m.tab()
				if item := m.selectedItem(tab); item != nil && item.target != nil && item.target.Editable {
					m.startEdit(item.target)
				}
			}
		case "n":
			if m.currentSource().Name() == "SSH" {
				m.startNew()
			}
		case "x":
			if m.currentSource().Name() == "SSH" {
				tab := m.tab()
				if item := m.selectedItem(tab); item != nil && item.target != nil && item.target.Editable {
					m.mode = modeConfirmDelete
				}
			}
		case "r":
			m.status = fmt.Sprintf("Refreshing %s...", m.currentSource().Name())
			return m, m.fetchTab(m.activeTab)
		case "s":
			if m.currentSource().Name() == "GKE" {
				m.mode = modeShellInput
				m.shellInput.SetValue("/bin/sh")
				m.shellInput.Focus()
				return m, textinput.Blink
			}
		}
	}
	return m, nil
}

func (m Model) switchTab(delta int) (tea.Model, tea.Cmd) {
	m.activeTab = (m.activeTab + delta + len(m.sources)) % len(m.sources)
	m.searchQuery = ""
	m.status = ""
	tab := m.tab()
	if !tab.fetched {
		m.status = fmt.Sprintf("Loading %s...", m.currentSource().Name())
		return m, m.fetchTab(m.activeTab)
	}
	m.rebuildList()
	return m, nil
}

func (m Model) viewList() string {
	var b strings.Builder

	// Title + tabs
	b.WriteString(titleStyle.Render("Pier"))
	b.WriteString("  ")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	tab := m.tab()
	if !tab.fetched {
		b.WriteString(helpStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if len(tab.flatItems) == 0 {
		b.WriteString(helpStyle.Render("No targets found"))
		b.WriteString("\n")
	} else {
		vh := m.listViewHeight()
		end := tab.scrollOffset + vh
		if end > len(tab.flatItems) {
			end = len(tab.flatItems)
		}

		// Scroll indicator (top)
		if tab.scrollOffset > 0 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ... %d more above", tab.scrollOffset)))
			b.WriteString("\n")
		}

		for i := tab.scrollOffset; i < end; i++ {
			item := tab.flatItems[i]
			selected := i == tab.cursor
			if item.isGroup {
				arrow := "v"
				if tab.collapsed[item.group] {
					arrow = ">"
				}
				line := fmt.Sprintf("%s %s", arrow, item.group)
				if selected {
					b.WriteString(selectedStyle.Render(line))
				} else {
					b.WriteString(groupStyle.Render(line))
				}
			} else {
				t := item.target
				line := fmt.Sprintf("  %s  %s", t.Alias, t.Detail)
				if selected {
					b.WriteString(selectedStyle.Render(line))
				} else {
					b.WriteString(hostStyle.Render(line))
				}
			}
			b.WriteString("\n")
		}

		// Scroll indicator (bottom)
		if end < len(tab.flatItems) {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ... %d more below", len(tab.flatItems)-end)))
			b.WriteString("\n")
		}
	}

	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.status))
	}

	b.WriteString("\n")
	help := "enter: connect  /: search  a/d: switch tab  r: refresh  q: quit"
	if m.currentSource().Name() == "SSH" {
		help = "enter: connect  /: search  e: edit  n: new  x: delete  a/d: switch tab  r: refresh  q: quit"
	} else if m.currentSource().Name() == "GKE" {
		help = "enter: exec  /: search  s: shell  a/d: switch tab  r: refresh  q: quit"
	}
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) renderTabs() string {
	var parts []string
	for i, s := range m.sources {
		name := s.Name()
		if i == m.activeTab {
			parts = append(parts, activeTabStyle.Render(name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(name))
		}
	}
	return strings.Join(parts, " ")
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
	tab := m.tab()
	tab.cursor = 0
	tab.scrollOffset = 0
	return m, cmd
}

func (m Model) viewSearch() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Pier"))
	b.WriteString("  ")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")
	b.WriteString(searchStyle.Render("Search: "))
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	tab := m.tab()
	vh := m.listViewHeight() - 2 // extra lines for search input
	end := tab.scrollOffset + vh
	if end > len(tab.flatItems) {
		end = len(tab.flatItems)
	}
	for i := tab.scrollOffset; i < end; i++ {
		item := tab.flatItems[i]
		selected := i == tab.cursor
		if item.isGroup {
			line := fmt.Sprintf("v %s", item.group)
			if selected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(groupStyle.Render(line))
			}
		} else {
			t := item.target
			line := fmt.Sprintf("  %s  %s", t.Alias, t.Detail)
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

// --- Shell Input (GKE) ---

func (m Model) updateShellInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			shell := m.shellInput.Value()
			if shell == "" {
				shell = "/bin/sh"
			}
			if gke, ok := m.currentSource().(*source.GKESource); ok {
				gke.SetShell(shell)
			}
			m.status = fmt.Sprintf("Shell set to: %s", shell)
			m.mode = modeList
			m.shellInput.Blur()
			return m, nil
		case "esc":
			m.mode = modeList
			m.shellInput.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.shellInput, cmd = m.shellInput.Update(msg)
	return m, cmd
}

func (m Model) viewShellInput() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Set Shell"))
	b.WriteString("\n\n")
	b.WriteString(detailLabelStyle.Render("Shell:"))
	b.WriteString(m.shellInput.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter: confirm  esc: cancel"))
	return b.String()
}

// --- Edit Mode (SSH only) ---

func (m *Model) startEdit(t *source.Target) {
	m.mode = modeEdit
	m.editTarget = t
	m.editCursor = 0
	h := source.HostFromTarget(*t)
	m.editFields = makeEditFields(&h)
	m.editFields[0].Focus()
}

func (m *Model) startNew() {
	m.mode = modeNew
	m.editTarget = nil
	m.editCursor = 0
	h := config.Host{Port: "22", Group: "ungrouped"}
	m.editFields = makeEditFields(&h)
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

	// Read current hosts from SSH source targets
	tab := m.tab()
	hosts := targetsToHosts(tab.targets)

	if m.mode == modeEdit && m.editTarget != nil {
		if m.editTarget.Alias != h.Alias {
			hosts = config.DeleteHost(hosts, m.editTarget.Alias)
		}
	}

	hosts = config.UpdateHost(hosts, h)

	if err := config.WriteFile(m.configPath, hosts); err != nil {
		m.status = fmt.Sprintf("Write error: %v", err)
	} else {
		m.status = fmt.Sprintf("Saved: %s", h.Alias)
	}

	m.mode = modeList
	// Refresh SSH tab
	return m, m.fetchTab(m.activeTab)
}

func (m Model) viewEdit() string {
	var b strings.Builder

	title := "Edit Host"
	if m.mode == modeNew {
		title = "New Host"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

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

// --- Confirm Delete (SSH only) ---

func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			tab := m.tab()
			if item := m.selectedItem(tab); item != nil && item.target != nil {
				alias := item.target.Alias
				hosts := targetsToHosts(tab.targets)
				hosts = config.DeleteHost(hosts, alias)
				if err := config.WriteFile(m.configPath, hosts); err != nil {
					m.status = fmt.Sprintf("Write error: %v", err)
				} else {
					m.status = fmt.Sprintf("Deleted: %s", alias)
				}
			}
			m.mode = modeList
			return m, m.fetchTab(m.activeTab)
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
	tab := m.tab()
	if item := m.selectedItem(tab); item != nil && item.target != nil {
		alias = item.target.Alias
	}

	b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/N)", alias)))
	return b.String()
}

// --- Helpers ---

func (m *Model) rebuildList() {
	tab := m.tab()
	filtered := tab.targets
	if m.searchQuery != "" {
		filtered = source.FilterTargets(tab.targets, m.searchQuery)
	}

	groups := source.GroupTargets(filtered)
	tab.flatItems = nil

	for _, g := range groups {
		tab.flatItems = append(tab.flatItems, listItem{isGroup: true, group: g.Name})
		if !tab.collapsed[g.Name] {
			for i := range g.Targets {
				tab.flatItems = append(tab.flatItems, listItem{target: &g.Targets[i]})
			}
		}
	}
}

func (m *Model) moveCursor(delta int) {
	tab := m.tab()
	if len(tab.flatItems) == 0 {
		return
	}
	tab.cursor += delta
	if tab.cursor < 0 {
		tab.cursor = 0
	}
	if tab.cursor >= len(tab.flatItems) {
		tab.cursor = len(tab.flatItems) - 1
	}

	// Adjust scroll to keep cursor visible
	vh := m.listViewHeight()
	if vh <= 0 {
		return
	}
	if tab.cursor < tab.scrollOffset {
		tab.scrollOffset = tab.cursor
	}
	if tab.cursor >= tab.scrollOffset+vh {
		tab.scrollOffset = tab.cursor - vh + 1
	}
}

// listViewHeight returns how many list rows fit on screen.
// Reserve lines for: title+tabs(2) + status(2) + help(2) = 6
func (m Model) listViewHeight() int {
	h := m.height - 6
	if h < 1 {
		h = 20 // fallback before first WindowSizeMsg
	}
	return h
}

func (m Model) selectedItem(tab *tabState) *listItem {
	if tab.cursor < 0 || tab.cursor >= len(tab.flatItems) {
		return nil
	}
	return &tab.flatItems[tab.cursor]
}

func targetsToHosts(targets []source.Target) []config.Host {
	hosts := make([]config.Host, 0, len(targets))
	for _, t := range targets {
		if t.Source == "ssh" {
			hosts = append(hosts, source.HostFromTarget(t))
		}
	}
	return hosts
}

// ConnectResult returns the source and target to connect to after TUI exits.
func (m Model) ConnectResult() (source.Source, *source.Target) {
	return m.connectSource, m.connectTarget
}

var _ tea.Model = Model{}
