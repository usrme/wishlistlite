package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sshExecutable = "ssh"

var (
	docStyle         = lipgloss.NewStyle().Margin(1, 2)
	nordAuroraYellow = lipgloss.Color("#ebcb8b")
	nordAuroraOrange = lipgloss.Color("#d08770")
	titleStyle       = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fffdf5ff")).
				Background(lipgloss.Color("#5e81ac")). // Nord Frost dark blue
				Padding(0, 1)
	filterPromptStyle = lipgloss.NewStyle().Foreground(nordAuroraYellow)
	filterCursorStyle = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	inputPromptStyle  = lipgloss.NewStyle().Foreground(nordAuroraYellow).Padding(0, 0, 0, 2)
	inputCursorStyle  = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	versionStyle      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).Render
)

type Item struct {
	Host      string
	Hostname  string
	Timestamp string
}

func (i Item) Title() string { return i.Host }
func (i Item) Description() string {
	if i.Timestamp != "" {
		return i.Timestamp
	}
	return i.Hostname
}
func (i Item) FilterValue() string { return i.Host }

type model struct {
	list          list.Model
	originalItems []list.Item
	sortedItems   []list.Item
	choice        string
	quitting      bool
	connectInput  textinput.Model
	sorted        bool
}

func newModel(items, sortedItems []list.Item) model {
	// set up main list
	hostList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	hostList.Title = "Wishlist Lite"
	hostList.Styles.Title = titleStyle
	hostList.FilterInput.PromptStyle = filterPromptStyle
	hostList.FilterInput.CursorStyle = filterCursorStyle

	// make sure custom keys have help text available
	hostList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{customKeys.Input, customKeys.Connect, customKeys.Cancel, customKeys.Sort}
	}
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			customKeys.Input,
			customKeys.Connect,
			customKeys.Cancel,
			customKeys.Sort,
		}
	}

	// set up input prompt for custom connection
	input := textinput.New()
	input.Prompt = "Connect to: "
	input.PromptStyle = inputPromptStyle
	input.CursorStyle = inputCursorStyle

	return model{
		list:          hostList,
		connectInput:  input,
		originalItems: items,
		sortedItems:   sortedItems,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	if m.connectInput.Focused() {
		return m.updateCustomInput(msg)
	}

	if m.sorted {
		return m.unsort(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
		// don't match any of the keys below if we're actively filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, customKeys.Input):
			m.connectInput.Focus()
			cmds = append(cmds, textinput.Blink)

		case key.Matches(msg, customKeys.Connect):
			return m.recordConnection(msg)

		case key.Matches(msg, customKeys.Sort):
			m.sort(msg)
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) updateCustomInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "esc":
			m.connectInput.Blur()
		case "enter":
			m.choice = m.connectInput.Value()
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.connectInput, cmd = m.connectInput.Update(msg)
	return m, cmd
}

func (m model) unsort(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, customKeys.Sort):
			m.sorted = false
			customKeys.Sort.SetHelp("r", "recently used")
			m.list.SetItems(m.originalItems)
			return m, nil
		}
	}
	return m, nil
}

func (m model) sort(msg tea.Msg) {
	m.sorted = true
	customKeys.Sort.SetHelp("r", "revert to default")
	m.list.SetItems(m.sortedItems)
}

func (m model) recordConnection(msg tea.Msg) (tea.Model, tea.Cmd) {
	i, ok := m.list.SelectedItem().(Item)
	if ok {
		m.choice = string(i.Host)
		items := timestampFirstItem(itemToFront(m.sortedItems, i))
		itemsToJson(recentlyUsedPath, items, true)
	}
	return m, tea.Quit
}

func (m model) View() string {
	var view string

	m.list.NewStatusMessage(versionStyle(pkgVersion()))

	if m.connectInput.Focused() {
		customKeys.Cancel.SetEnabled(true)
		customKeys.Input.SetEnabled(false)
		customKeys.Sort.SetEnabled(false)
		m.list.KeyMap.CursorUp.SetEnabled(false)
		m.list.KeyMap.CursorDown.SetEnabled(false)
		m.list.KeyMap.Filter.SetEnabled(false)
		m.list.KeyMap.Quit.SetEnabled(false)
		m.list.KeyMap.ShowFullHelp.SetEnabled(false)

		m.list.SetShowTitle(false)
		view += m.connectInput.View()
	} else {
		customKeys.Cancel.SetEnabled(false)
		customKeys.Input.SetEnabled(true)
		customKeys.Sort.SetEnabled(true)
		m.list.KeyMap.CursorUp.SetEnabled(true)
		m.list.KeyMap.CursorDown.SetEnabled(true)
		m.list.KeyMap.Filter.SetEnabled(true)
		m.list.KeyMap.Quit.SetEnabled(true)
		m.list.KeyMap.ShowFullHelp.SetEnabled(true)
	}
	view += m.list.View()

	return docStyle.Render(view)
}
