package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle           = lipgloss.NewStyle().Margin(1, 2)
	nordAuroraYellow   = lipgloss.Color("#ebcb8b")
	nordAuroraOrange   = lipgloss.Color("#d08770")
	nordAuroraGreen    = lipgloss.Color("#a3be8c")
	dimNordAuroraGreen = lipgloss.Color("#7a8e69")
	titleStyle         = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fffdf5ff")).
				Background(lipgloss.Color("#5e81ac")). // Nord Frost dark blue
				Padding(0, 1)
	filterPromptStyle = lipgloss.NewStyle().Foreground(nordAuroraYellow)
	filterCursorStyle = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	inputPromptStyle  = lipgloss.NewStyle().Foreground(nordAuroraYellow).Padding(0, 0, 0, 2)
	inputCursorStyle  = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	spinnerStyle      = lipgloss.NewStyle().Foreground(nordAuroraGreen)
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

type connection struct {
	state string
}

type model struct {
	list            list.Model
	originalItems   []list.Item
	sortedItems     []list.Item
	choice          string
	quitting        bool
	connection      connection
	connectInput    textinput.Model
	sorted          bool
	defaultDelegate list.ItemDelegate
	connectDelegate list.ItemDelegate
	spinner         spinner.Model
	stopwatch       stopwatch.Model
}

func newModel(items, sortedItems []list.Item) model {
	// set up default delegate for styling
	defaultDelegate := list.NewDefaultDelegate()
	defaultDelegate.Styles.SelectedTitle = defaultDelegate.Styles.SelectedTitle.
		Foreground(nordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)
	defaultDelegate.Styles.SelectedDesc = defaultDelegate.Styles.SelectedDesc.
		Foreground(dimNordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)

	// create separate delegate for when active input is present
	connectDelegate := defaultDelegate
	connectDelegate.Styles.SelectedTitle = connectDelegate.Styles.DimmedTitle
	connectDelegate.Styles.SelectedDesc = connectDelegate.Styles.DimmedDesc
	connectDelegate.Styles.NormalTitle = connectDelegate.Styles.DimmedTitle
	connectDelegate.Styles.NormalDesc = connectDelegate.Styles.DimmedDesc

	// set up main list
	hostList := list.New(items, defaultDelegate, 0, 0)
	hostList.Title = "Wishlist Lite"
	hostList.Styles.Title = titleStyle
	hostList.FilterInput.PromptStyle = filterPromptStyle
	hostList.FilterInput.CursorStyle = filterCursorStyle

	// make sure custom keys have help text available
	hostList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{customKeys.Input, customKeys.Connect, customKeys.Cancel, customKeys.Sort, customKeys.Delete}
	}
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			customKeys.Input,
			customKeys.Connect,
			customKeys.Cancel,
			customKeys.Sort,
			customKeys.Delete,
		}
	}

	// set up input prompt for custom connection
	input := textinput.New()
	input.Prompt = "Connect to: "
	input.PromptStyle = inputPromptStyle
	input.CursorStyle = inputCursorStyle

	sp := spinner.New()
	sp.Spinner = spinner.Pulse
	sp.Style = spinnerStyle

	st := stopwatch.NewWithInterval(time.Millisecond)
	return model{
		list:            hostList,
		connectInput:    input,
		originalItems:   items,
		sortedItems:     sortedItems,
		defaultDelegate: defaultDelegate,
		connectDelegate: connectDelegate,
		spinner:         sp,
		stopwatch:       st,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

var (
	stdOutChan = make(chan string)
	stdErrChan = make(chan string)
)

func runBackgroundProcess(stdOutChan chan<- string, stdErrChan chan<- string, executableName string, arg ...string) {
	c := exec.Command(executableName, arg...)
	stdout, _ := c.StdoutPipe()
	stderr, _ := c.StderrPipe()
	c.Start()

	go func() {
		slurp, _ := io.ReadAll(stderr)
		stdErrChan <- fmt.Sprint(slurp)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanRunes)
	for scanner.Scan() {
		stdOutChan <- string(scanner.Text())
	}
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
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, customKeys.Delete):
				index := m.list.Index()
				m.list.RemoveItem(index)
				itemsToJson(recentlyUsedPath, m.list.Items(), true)
				m.sortedItems = m.list.Items()
				return m, nil
			case key.Matches(msg, customKeys.Sort):
				return m.unsort(msg)
			}
		}
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
			m.list.SetDelegate(m.connectDelegate)
			cmds = append(cmds, textinput.Blink)

		case key.Matches(msg, customKeys.Connect):
			i, ok := m.list.SelectedItem().(Item)
			if ok {
				m.connection.state = "Connecting"
				m.choice = string(i.Host)
				cmds = append(cmds, m.spinner.Tick)
				cmds = append(cmds, m.stopwatch.Init())
				// extremely hack-y way to prepend 'm.choice' to 'sshControlParentOpts'
				go runBackgroundProcess(stdOutChan, stdErrChan, sshExecutableName, append([]string{m.choice}, sshControlParentOpts...)...)
			}

		case key.Matches(msg, customKeys.Sort):
			return m.sort(msg)
		}
	}

	select {
	case <-stdOutChan:
		m.connection.state = "Connected"
		return m.recordConnection(m.list.SelectedItem().(Item))
	case <-stdErrChan:
		return m, tea.Quit
	default:
		break
	}

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	m.stopwatch, cmd = m.stopwatch.Update(msg)
	cmds = append(cmds, cmd)
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var (
		view     string
		sections []string
		style    lipgloss.Style
	)

	if m.connection.state == "Connecting" {
		return fmt.Sprintf("\n\n   %s Connecting... %s\n\n", m.spinner.View(), m.stopwatch.View())
	} else if m.connection.state == "Connected" {
		return style.Render(view)
	}

	m.list.NewStatusMessage(versionStyle(pkgVersion()))
	m.list.Styles.HelpStyle.Padding(0, 0, 0, 2)
	style = docStyle

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

		m.list.Styles.HelpStyle.Padding(0, 0, 1, 2)
		style = lipgloss.NewStyle().Margin(1, 0, 0, 2)
		sections = append(sections, m.connectInput.View())
	} else {
		customKeys.Cancel.SetEnabled(false)
		customKeys.Input.SetEnabled(true)
		customKeys.Sort.SetEnabled(true)
		customKeys.Delete.SetEnabled(false)
		m.list.KeyMap.CursorUp.SetEnabled(true)
		m.list.KeyMap.CursorDown.SetEnabled(true)
		m.list.KeyMap.Filter.SetEnabled(true)
		m.list.KeyMap.Quit.SetEnabled(true)
		m.list.KeyMap.ShowFullHelp.SetEnabled(true)
	}

	if m.sorted {
		customKeys.Delete.SetEnabled(true)
	}

	sections = append(sections, m.list.View())
	view = lipgloss.JoinVertical(lipgloss.Left, sections...)
	return style.Render(view)
}

func (m model) updateCustomInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "esc":
			m.connectInput.Blur()
			m.list.SetDelegate(m.defaultDelegate)
		case "enter":
			m.choice = m.connectInput.Value()
			i := Item{Host: m.choice, Hostname: m.choice}
			return m.recordConnection(i)
		}
	}
	var cmd tea.Cmd
	m.connectInput, cmd = m.connectInput.Update(msg)
	return m, cmd
}

func (m model) unsort(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.sorted = false
	customKeys.Sort.SetHelp("r", "recently used")
	m.list.SetItems(m.originalItems)
	return m, nil
}

func (m model) sort(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.sorted = true
	customKeys.Sort.SetHelp("r", "revert to default")
	m.list.SetItems(m.sortedItems)
	return m, nil
}

func (m model) recordConnection(i Item) (tea.Model, tea.Cmd) {
	items := timestampFirstItem(itemToFront(m.sortedItems, i))
	itemsToJson(recentlyUsedPath, items, true)
	return m, tea.Quit
}
