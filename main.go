package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sshExecutable = "ssh"

var docStyle = lipgloss.NewStyle().Margin(1, 2)

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

func runExecutable(execPath string, args []string) {
	env := os.Environ()
	err := syscall.Exec(execPath, args, env)
	if err != nil {
		panic(err)
	}
}

func main() {
	execPath := verifyExecutable(sshExecutable)
	sshConfigPath := fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	items, _ := getHostsFromSshConfig(sshConfigPath)

	md := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	md.list.Title = "Wishlist Lite"

	p := tea.NewProgram(md, tea.WithAltScreen())

	m, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.choice != "" {
		runExecutable(execPath, []string{sshExecutable, m.choice})
	}
}
