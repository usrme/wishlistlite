package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sshExecutable = "ssh"

var (
	defDelegate        = list.NewDefaultDelegate()
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
	sshConfigPath     = fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	recentlyUsedPath  = fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/recent.json")
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
	list          list.Model
	originalItems []list.Item
	sortedItems   []list.Item
	choice        string
	quitting      bool
	connectInput  textinput.Model
	sorted        bool
	conDelegate   list.ItemDelegate
}

func main() {
	execPath := verifyExecutable(sshExecutable)
	p := tea.NewProgram(New(), tea.WithAltScreen())

	m, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("failed to start: %w", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.choice != "" {
		runExecutable(execPath, []string{sshExecutable, m.choice})
	}
}

func New() model {
	items, _ := sshConfigHosts(sshConfigPath)
	itemsToJson(recentlyUsedPath, items, false)

	defDelegate.Styles.SelectedTitle = defDelegate.Styles.SelectedTitle.
		Foreground(nordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)
	defDelegate.Styles.SelectedDesc = defDelegate.Styles.SelectedDesc.
		Foreground(dimNordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)
	defDelegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{defaultKeyMap.Input, defaultKeyMap.Connect, defaultKeyMap.Cancel, defaultKeyMap.Sort}
	}

	hostList := list.New(items, defDelegate, 0, 0)
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

	// Create separate delegate for when active input is present
	conDelegate := defDelegate
	conDelegate.Styles.SelectedTitle = conDelegate.Styles.DimmedTitle
	conDelegate.Styles.SelectedDesc = conDelegate.Styles.DimmedDesc
	conDelegate.Styles.NormalTitle = conDelegate.Styles.DimmedTitle
	conDelegate.Styles.NormalDesc = conDelegate.Styles.DimmedDesc

	sortedItems, err := itemsFromJson(recentlyUsedPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return model{
		list:          hostList,
		connectInput:  input,
		originalItems: items,
		sortedItems:   sortedItems,
		conDelegate:   conDelegate,
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
				m.list.SetDelegate(defDelegate)
			case "enter":
				m.choice = m.connectInput.Value()
				return m, tea.Quit
			}
		}
		var cmd tea.Cmd
		m.connectInput, cmd = m.connectInput.Update(msg)
		return m, cmd
	}

	if m.sorted {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, defaultKeyMap.Sort):
				m.sorted = false
				m.list.SetItems(m.originalItems)
				return m, nil
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
		// Don't match any of the keys below if we're actively filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, defaultKeyMap.Input):
			m.connectInput.Focus()
			m.list.SetDelegate(m.conDelegate)
			cmds = append(cmds, textinput.Blink)

		case key.Matches(msg, defaultKeyMap.Connect):
			i, ok := m.list.SelectedItem().(Item)
			if ok {
				m.choice = string(i.Host)
				items := timestampFirstItem(itemToFront(m, i))
				itemsToJson(recentlyUsedPath, items, true)
			}
			return m, tea.Quit

		case key.Matches(msg, defaultKeyMap.Sort):
			m.sorted = true
			m.list.SetItems(m.sortedItems)
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var view string

	m.list.NewStatusMessage(versionStyle(pkgVersion()))

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
	// using 'panic()' as it's supposedly acceptable during initialization phases
	// https://go.dev/doc/effective_go#panic
	if err != nil {
		panic(err)
	}
	return path
}

func runExecutable(execPath string, args []string) error {
	err := syscall.Exec(execPath, args, os.Environ())
	if err != nil {
		return fmt.Errorf("unable to run executable '%s' with args '%v': %w", execPath, args, err)
	}
	return nil
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

func sshConfigHosts(filePath string) ([]list.Item, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s': %w", filePath, err)
	}

	// grab all 'Host' ('Host' not included) and 'HostName' ('HostName' included)
	pat := regexp.MustCompile(`(?m)^Host\s([^\*][a-zA-Z0-9_\.-]*)[\r\n](\s+HostName.*)?`)
	mainMatches := pat.FindAllStringSubmatch(string(content), -1)

	var items []list.Item
	for _, m := range mainMatches {
		// if 'HostName' was present
		if m[2] != "" {
			// make sure 'HostName' was defined correctly (i.e. followed by a space)
			pat := regexp.MustCompile(`HostName\s(.*)`)
			secMatches := pat.FindAllStringSubmatch(m[2], -1)
			for _, n := range secMatches {
				host := Item{Host: m[1], Hostname: n[1]}
				items = append(items, host)
			}
		} else {
			// if no 'HostName' was found just add the 'Host'
			// value as the description in the 'Item' struct
			host := Item{Host: m[1], Hostname: m[1]}
			items = append(items, host)
		}
	}
	return items, nil
}

func itemsToJson(filePath string, l []list.Item, overwrite bool) error {
	result, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("could not marshal JSON: %w", err)
	}
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) || overwrite {
		ioutil.WriteFile(filePath, result, 0644)
	}
	return nil
}

func itemsFromJson(filePath string) ([]list.Item, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s': %w", filePath, err)
	}
	var payload []Item
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON from '%s': %w", filePath, err)
	}

	var items []list.Item
	for _, item := range payload {
		if item.Timestamp != "" {
			items = append(items, Item{Host: item.Host, Hostname: item.Hostname, Timestamp: item.Timestamp})
		} else {
			items = append(items, Item{Host: item.Host, Hostname: item.Hostname})
		}
	}
	return items, nil
}

func itemToFront(m model, i Item) []list.Item {
	var sortedHostSlice []string
	for _, host := range m.sortedItems {
		sortedHostSlice = append(sortedHostSlice, host.(Item).Title())
	}
	sortedHostSlice = moveToFront(i.Host, sortedHostSlice)

	var items []list.Item
	for _, sortedHost := range sortedHostSlice {
		for n := range m.sortedItems {
			if sortedHost == m.sortedItems[n].(Item).Host {
				if m.sortedItems[n].(Item).Timestamp != "" {
					items = append(items, Item{Host: sortedHost, Hostname: m.sortedItems[n].(Item).Hostname, Timestamp: m.sortedItems[n].(Item).Timestamp})
				} else {
					items = append(items, Item{Host: sortedHost, Hostname: m.sortedItems[n].(Item).Hostname})
				}
			}
		}
	}
	return items
}

// https://github.com/golang/go/wiki/SliceTricks#move-to-front-or-prepend-if-not-present-in-place-if-possible
func moveToFront(needle string, haystack []string) []string {
	if len(haystack) != 0 && haystack[0] == needle {
		return haystack
	}
	prev := needle
	for i, elem := range haystack {
		switch {
		case i == 0:
			haystack[0] = needle
			prev = elem
		case elem == needle:
			haystack[i] = prev
			return haystack
		default:
			haystack[i] = prev
			prev = elem
		}
	}
	return append(haystack, prev)
}

func timestampFirstItem(l []list.Item) []list.Item {
	// This seemed to be the easiest way to directly modify
	// the first element in an already sorted slice
	l[0] = Item{
		Host:      l[0].(Item).Host,
		Hostname:  l[0].(Item).Hostname,
		Timestamp: time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
	}
	return l
}

func pkgVersion() string {
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}
	return version
}
