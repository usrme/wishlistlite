package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sshExecutable = "ssh"

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
	versionStyle      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).Render
)

type item struct {
	host, hostname string
}

func (i item) Title() string       { return i.host }
func (i item) Description() string { return i.hostname }
func (i item) FilterValue() string { return i.host }

type keyMap struct {
	Input   key.Binding
	Connect key.Binding
	Cancel  key.Binding
	Sort    key.Binding
}

var defaultKeyMap = keyMap{
	Input: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "input connection"),
	),
	Connect: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "connect"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel input"),
	),
	Sort: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "recently used"),
	),
}

type model struct {
	list         list.Model
	choice       string
	quitting     bool
	connectInput textinput.Model
}

func main() {
	execPath := verifyExecutable(sshExecutable)
	p := tea.NewProgram(New(), tea.WithAltScreen())

	m, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.choice != "" {
		runExecutable(execPath, []string{sshExecutable, m.choice})
	}
}

func New() model {
	sshConfigPath := fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	items, _ := getHostsFromSshConfig(sshConfigPath)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(nordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(dimNordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{defaultKeyMap.Input, defaultKeyMap.Connect, defaultKeyMap.Cancel, defaultKeyMap.Sort}
	}

	hostList := list.New(items, delegate, 0, 0)
	hostList.Title = "Wishlist Lite"
	hostList.Styles.Title = titleStyle
	hostList.FilterInput.PromptStyle = filterPromptStyle
	hostList.FilterInput.CursorStyle = filterCursorStyle
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			defaultKeyMap.Input,
			defaultKeyMap.Connect,
			defaultKeyMap.Cancel,
			defaultKeyMap.Sort,
		}
	}
	input := textinput.New()
	input.Prompt = "Connect to: "
	input.PromptStyle = inputPromptStyle
	input.CursorStyle = inputCursorStyle
	return model{
		list:         hostList,
		connectInput: input,
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
		// Don't match any of the keys below if we're actively filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, defaultKeyMap.Input):
			m.connectInput.Focus()
			cmds = append(cmds, textinput.Blink)

		case key.Matches(msg, defaultKeyMap.Connect):
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i.host)
			}
			return m, tea.Quit
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var view string

	m.list.NewStatusMessage(versionStyle(getPkgVersion()))

	if m.connectInput.Focused() {
		defaultKeyMap.Cancel.SetEnabled(true)
		defaultKeyMap.Input.SetEnabled(false)
		defaultKeyMap.Sort.SetEnabled(false)
		m.list.KeyMap.CursorUp.SetEnabled(false)
		m.list.KeyMap.CursorDown.SetEnabled(false)
		m.list.KeyMap.Filter.SetEnabled(false)
		m.list.KeyMap.Quit.SetEnabled(false)
		m.list.KeyMap.ShowFullHelp.SetEnabled(false)

		m.list.SetShowTitle(false)
		view += m.connectInput.View()
	} else {
		defaultKeyMap.Cancel.SetEnabled(false)
		defaultKeyMap.Input.SetEnabled(true)
		defaultKeyMap.Sort.SetEnabled(true)
		m.list.KeyMap.CursorUp.SetEnabled(true)
		m.list.KeyMap.CursorDown.SetEnabled(true)
		m.list.KeyMap.Filter.SetEnabled(true)
		m.list.KeyMap.Quit.SetEnabled(true)
		m.list.KeyMap.ShowFullHelp.SetEnabled(true)
	}
	view += m.list.View()

	return docStyle.Render(view)
}

func verifyExecutable(execName string) string {
	path, err := exec.LookPath(execName)
	if err != nil {
		panic(err)
	}
	return path
}

func runExecutable(execPath string, args []string) {
	env := os.Environ()
	err := syscall.Exec(execPath, args, env)
	if err != nil {
		panic(err)
	}
}

func userHomeDir() string {
	switch runtime.GOOS {
	case "windows":
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home

	case "linux":
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
	}
	return os.Getenv("HOME")
}

// TODO: Write tests for this
func getHostsFromSshConfig(filePath string) ([]list.Item, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Err")
	}

	pat := regexp.MustCompile(`Host\s([^\*].*)[\r\n]\s+HostName\s(.*)`)
	matches := pat.FindAllStringSubmatch(string(content), -1)
	var items []list.Item
	for _, match := range matches {
		host := item{host: match[1], hostname: match[2]}
		items = append(items, host)
	}

	return items, nil
}

func getPkgVersion() string {
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	return version
}
