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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sshExecutable = "ssh"

var (
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fffdf5ff")).
			Background(lipgloss.Color("#5e81ac")).
			Padding(0, 1)

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
	list     list.Model
	items    []item
	choice   string
	quitting bool
	keys     *listKeyMap
}

type listKeyMap struct {
	input key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		input: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "input"),
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

func newModel() model {
	var (
		listKeys = newListKeyMap()
	)

	sshConfigPath := fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	items, _ := getHostsFromSshConfig(sshConfigPath)

	delegate := newItemDelegate()
	hostList := list.New(items, delegate, 0, 0)
	hostList.Title = "Wishlist Lite"
	hostList.Styles.Title = titleStyle
	// TODO: Figure out why styling 'hostList.Styles.FilterPrompt' doesn't work like 'hostList.Styles.Title'
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.input,
		}
	}

	return model{
		list: hostList,
		keys: listKeys,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i.host)
			}
			return m, tea.Quit
		}
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, m.keys.input):
			statusCmd := m.list.NewStatusMessage(statusMessageStyle("Pressed"))
			return m, statusCmd
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
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
	p := tea.NewProgram(newModel(), tea.WithAltScreen())

	m, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.choice != "" {
		runExecutable(execPath, []string{sshExecutable, m.choice})
	}
}
