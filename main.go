package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"syscall"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sshExecutable = "ssh"

var (
	docStyle         = lipgloss.NewStyle().Margin(1, 2)
	offWhiteColor    = lipgloss.Color("#fffdf5ff")
	nordAuroraYellow = lipgloss.Color("#ebcb8b")
	nordAuroraOrange = lipgloss.Color("#d08770")
	titleStyle       = lipgloss.NewStyle().
				Foreground(offWhiteColor).
				Background(lipgloss.Color("#5e81ac")). // Nord Frost dark blue
				Padding(0, 1)
	selectedItemColor = lipgloss.Color("#a3be8c")                        // Nord Aurora green
	selectedDescColor = lipgloss.Color("#7a8e69")                        // Dimmed Nord Aurora green
	filterPromptStyle = lipgloss.NewStyle().Foreground(nordAuroraYellow) // Nord Aurora yellow
	filterCursorStyle = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	inputPromptStyle  = lipgloss.NewStyle().Foreground(nordAuroraYellow)
	inputCursorStyle  = lipgloss.NewStyle().Foreground(nordAuroraOrange)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

type item struct {
	host, hostname string
}

func (i item) Title() string       { return i.host }
func (i item) Description() string { return i.hostname }
func (i item) FilterValue() string { return i.host }

type model struct {
	list         list.Model
	items        []item
	choice       string
	quitting     bool
	keys         *listKeyMap
	connectInput textinput.Model
}

type listKeyMap struct {
	input key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		input: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "input connection"),
		),
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

func getHostsFromSshConfig(filePath string) ([]list.Item, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Err")
	}

	pat := regexp.MustCompile("Host\\s([^\\*].*)[\\r\\n]\\s+HostName\\s(.*)")
	matches := pat.FindAllStringSubmatch(string(content), -1)
	var items []list.Item
	for _, match := range matches {
		host := item{host: match[1], hostname: match[2]}
		items = append(items, host)
	}

	return items, nil
}

func New() model {
	var (
		listKeys = newListKeyMap()
	)

	sshConfigPath := fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	items, _ := getHostsFromSshConfig(sshConfigPath)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(selectedItemColor).
		BorderLeftForeground(selectedItemColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(selectedDescColor).
		BorderLeftForeground(selectedItemColor)
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{listKeys.input}
	}

	hostList := list.New(items, delegate, 0, 0)
	hostList.Title = "Wishlist Lite"
	hostList.Styles.Title = titleStyle
	hostList.FilterInput.PromptStyle = filterPromptStyle
	hostList.FilterInput.CursorStyle = filterCursorStyle
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.input,
		}
	}
	input := textinput.New()
	input.Prompt = "Connect to: "
	input.PromptStyle = inputPromptStyle
	input.CursorStyle = inputCursorStyle
	return model{
		list:         hostList,
		keys:         listKeys,
		connectInput: input,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, tea.Quit
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

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i.host)
			}
			return m, tea.Quit
		}
		// Don't match any of the keys below if we're actively filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, m.keys.input):
			m.connectInput.Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.connectInput.Focused() {
		// TODO: Find way to co-opt 'FilterInput' when 'ConnectInput' is focused
		return docStyle.Render(m.connectInput.View())
	}
	return docStyle.Render(m.list.View())
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
