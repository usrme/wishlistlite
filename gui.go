package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
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
				Foreground(lipgloss.Color("#fffdf5ff"))
	filterPromptStyle = lipgloss.NewStyle().Foreground(nordAuroraYellow)
	filterCursorStyle = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	inputPromptStyle  = lipgloss.NewStyle().Foreground(nordAuroraYellow).Padding(0, 0, 0, 2)
	inputCursorStyle  = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	spinnerStyle      = lipgloss.NewStyle().Foreground(nordAuroraGreen)
	pingSpinnerStyle  = lipgloss.NewStyle().Foreground(nordAuroraYellow)
	versionStyle      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).Render
)

// An Item is an item that appears in the list.
type Item struct {
	Host         string
	Hostname     string
	Timestamp    string
	Extra        string
	SwitchFilter bool
}

// Title returns the Host field for an Item as that is the
// value that will be the primary for basing the selection.
func (i Item) Title() string { return i.Host }

// Description returns the Timestamp field for an Item if
// it is present (i.e. when in the Recently Used view),
// otherwise just the Hostname field.
func (i Item) Description() string {
	if i.Timestamp != "" {
		return i.Timestamp
	}
	if i.Extra != "" {
		return fmt.Sprintf("%s :: %s", i.Hostname, i.Extra)
	}
	return i.Hostname
}

// FilterValue returns the value that is used when
// filtering the list.
func (i Item) FilterValue() string {
	if i.SwitchFilter {
		return i.Hostname
	}
	return i.Host
}

// A connection stores information about a successful
// connection that was made against a chosen host.
type connection struct {
	output      string
	startupTime time.Duration
	state       string
}

// A connectionOutputMsg indicates that something has
// been written to the standard output of a connection.
type connectionOutputMsg []string

// A connectionErrorMsg indicates that something has
// been written to the standard error of a connection.
type connectionErrorMsg []string

type model struct {
	list             list.Model
	originalItems    []list.Item
	sortedItems      []list.Item
	choice           string
	quitting         bool
	connection       connection
	err              string
	errorChan        chan []string
	outputChan       chan []string
	connectInput     textinput.Model
	sorted           bool
	defaultDelegate  list.ItemDelegate
	connectDelegate  list.ItemDelegate
	spinner          spinner.Model
	pingSpinner      spinner.Model
	stopwatch        stopwatch.Model
	recentlyUsedPath string
	pingOpts         []string
	sshOpts          []string
}

func newModel(items, sortedItems []list.Item, path string, pingOpts, sshOpts []string) model {
	// Set up default delegate for styling
	defaultDelegate := list.NewDefaultDelegate()
	defaultDelegate.Styles.SelectedTitle = defaultDelegate.Styles.SelectedTitle.
		Foreground(nordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)
	defaultDelegate.Styles.SelectedDesc = defaultDelegate.Styles.SelectedDesc.
		Foreground(dimNordAuroraGreen).
		BorderLeftForeground(nordAuroraGreen)

	// Create separate delegate for when active input is present
	connectDelegate := defaultDelegate
	connectDelegate.Styles.SelectedTitle = connectDelegate.Styles.DimmedTitle
	connectDelegate.Styles.SelectedDesc = connectDelegate.Styles.DimmedDesc
	connectDelegate.Styles.NormalTitle = connectDelegate.Styles.DimmedTitle
	connectDelegate.Styles.NormalDesc = connectDelegate.Styles.DimmedDesc

	// Set up main list
	hostList := list.New(items, defaultDelegate, 0, 0)
	hostList.Title = "Wishlist Lite"
	hostList.Styles.Title = titleStyle
	hostList.FilterInput.PromptStyle = filterPromptStyle
	hostList.FilterInput.Cursor.Style = filterCursorStyle

	bindings := []key.Binding{
		customKeys.Input,
		customKeys.Connect,
		customKeys.Cancel,
		customKeys.Sort,
		customKeys.Delete,
		customKeys.Ping,
	}
	// Make sure custom keys have help text available
	hostList.AdditionalShortHelpKeys = func() []key.Binding { return bindings }
	hostList.AdditionalFullHelpKeys = func() []key.Binding { return bindings }

	// Set up input prompt for custom connection
	input := textinput.New()
	input.Prompt = "Connect to: "
	input.PromptStyle = inputPromptStyle
	input.Cursor.Style = inputCursorStyle

	sp := spinner.New()
	sp.Spinner = spinner.Pulse
	sp.Style = spinnerStyle

	psp := spinner.New()
	psp.Spinner = spinner.Pulse
	psp.Style = pingSpinnerStyle

	st := stopwatch.NewWithInterval(time.Millisecond)
	return model{
		list:             hostList,
		errorChan:        make(chan []string),
		outputChan:       make(chan []string),
		connectInput:     input,
		originalItems:    items,
		sortedItems:      sortedItems,
		defaultDelegate:  defaultDelegate,
		connectDelegate:  connectDelegate,
		spinner:          sp,
		pingSpinner:      psp,
		stopwatch:        st,
		recentlyUsedPath: path,
		pingOpts:         pingOpts,
		sshOpts:          sshOpts,
	}
}

// execCommand returns a command that runs 'name' command with
// 'arg...' in the background when called writing to channels
// 'outChan' and 'errChan' depending on the scenario.
//
// 'lineTail' is for the purposes of storing only the last N
// number of lines from an output.
//
// 'wait' is required for commands where the entire output is
// required and the command must waited upon to finish.
func execCommand(outChan chan []string, errChan chan []string, name string, lineTail int, wait bool, arg ...string) tea.Cmd {
	return func() tea.Msg {
		c := exec.Command(name, arg...)
		stdout, _ := c.StdoutPipe()
		stderr, _ := c.StderrPipe()

		c.Start()

		slurp, _ := io.ReadAll(stderr)
		if len(slurp) > 0 {
			slurp := strings.Split(string(slurp), "")
			errChan <- connectionErrorMsg(slurp)
			return tea.Quit
		}

		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanLines)

		var out []string
		out = append(out, scanner.Text())
		for scanner.Scan() {
			out = append(out, scanner.Text())
		}
		if lineTail > 0 {
			out = out[len(out)-lineTail:]
		}
		outChan <- connectionOutputMsg(out)
		if wait {
			c.Wait()
		}
		return nil
	}
}

// waitForCommandError returns a tea.Cmd that waits for
// standard error activity on a channel.
func waitForCommandError(c chan []string) tea.Cmd {
	return func() tea.Msg {
		return connectionErrorMsg(<-c)
	}
}

// waitForCommandOutput returns a tea.Cmd that waits for
// standard output activity on a channel.
func waitForCommandOutput(c chan []string) tea.Cmd {
	return func() tea.Msg {
		return connectionOutputMsg(<-c)
	}
}

// Init initializes the model by returning commands through
// tea.Batch. In this case it sets up the model in a way
// that there are two commands - one for standard error and
// one for standard output - that will immediately be
// waited upon.
func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForCommandError(m.errorChan),
		waitForCommandOutput(m.outputChan),
	)
}

// Update returns the updated model and an optional command.
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

	// When the custom connection input is focused
	// adjust the model accordingly
	if m.connectInput.Focused() {
		return m.updateCustomInput(msg)
	}

	if m.sorted {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			// When the delete key was pressed remove the item
			// from both the list of items and from the file
			case key.Matches(msg, customKeys.Delete):
				index := m.list.Index()
				m.list.RemoveItem(index)
				itemsToJson(m.recentlyUsedPath, m.list.Items(), true)
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
			return m.quitProgram()
		}
		// Don't match any of the keys below if we're actively filtering,
		// but the above Ctrl+C and Q should still quit when filtering.
		// In any other mode Esc should still quit unless specified otherwise.
		if m.list.FilterState() == list.Filtering {
			break
		} else {
			switch keypress := msg.String(); keypress {
			case "esc":
				return m.quitProgram()
			}
		}
		switch {
		// When the key for initiating a custom connection was pressed,
		// focus the input, change the styling through a different
		// delegate and start blinking the input cursor
		case key.Matches(msg, customKeys.Input):
			m.connectInput.Focus()
			m.list.SetDelegate(m.connectDelegate)
			cmds = append(cmds, textinput.Blink)

		case key.Matches(msg, customKeys.Ping):
			i, ok := m.list.SelectedItem().(Item)
			if ok {
				m.connection.state = "Pinging"
				m.choice = i.Hostname
				cmds = append(cmds, m.pingSpinner.Tick)
				cmds = append(cmds, execCommand(m.outputChan, m.errorChan, "ping", 2, true, append([]string{m.choice}, m.pingOpts...)...))
			}

		case key.Matches(msg, customKeys.Connect):
			i, ok := m.list.SelectedItem().(Item)
			if ok {
				m.connection.state = "Connecting"
				m.choice = i.Host
				cmds = append(cmds, m.spinner.Tick)
				cmds = append(cmds, m.stopwatch.Init())
				// Extremely hack-y way to prepend 'm.choice'
				opts := append([]string{m.choice}, m.sshOpts...)
				opts = append(opts, sshControlParentOpts...)
				cmds = append(cmds, execCommand(m.outputChan, m.errorChan, sshExecutableName, 0, false, opts...))
			}

		case key.Matches(msg, customKeys.Sort):
			return m.sort(msg)
		}
	case connectionErrorMsg:
		if m.connection.state == "Pinging" {
			m.connection.state = "Pinged"
			m.connection.output = strings.Split(strings.Join(msg, ""), "\n")[0]
			return m, nil
		}
		// When something was received as 'connectionErrorMsg'
		// clear the choice from the model as the logic in
		// 'main.go' checks it to be present
		m.choice = ""
		m.err = strings.Join(msg, "")
		return m, tea.Quit
	// When something was received as 'connectionOutputMsg'
	// store what was received and stop all processing
	case connectionOutputMsg:
		if m.connection.state == "Pinging" {
			m.connection.state = "Pinged"
			last := msg[len(msg)-1]
			// The last line of the output is only empty when the ping did not succeed
			if last == "" {
				m.connection.output = fmt.Sprintf("%q could not ping", m.list.SelectedItem().(Item).Host)
			} else {
				m.connection.output = fmt.Sprintf("%q %s", m.list.SelectedItem().(Item).Host, last)
			}
			cmds = append(cmds, waitForCommandOutput(m.outputChan)) // Continue waiting for new output
		} else {
			m.connection.output = strings.Join(msg, "\n")
			m.connection.startupTime = m.stopwatch.Elapsed()
			m.connection.state = "Connected"
			return m.recordConnection(m.list.SelectedItem().(Item))
		}
	case spinner.TickMsg:
		m.pingSpinner, cmd = m.pingSpinner.Update(msg)
		cmds = append(cmds, cmd)
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

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

	if m.connection.state == "Pinging" {
		m.list.NewStatusMessage(fmt.Sprintf("%s %s", m.pingSpinner.View(), versionStyle(fmt.Sprintf("Pinging %q %s times", m.list.SelectedItem().(Item).Host, m.pingOpts[len(m.pingOpts)-1]))))
	} else if m.connection.state == "Pinged" {
		m.list.NewStatusMessage(versionStyle(m.connection.output))
	} else {
		m.list.NewStatusMessage(versionStyle(pkgVersion()))
	}

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

func (m model) quitProgram() (tea.Model, tea.Cmd) {
	// Clear the output just in case something was stored
	m.connection.output = ""
	m.choice = ""
	m.quitting = true
	return m, tea.Quit
}

// updateCustomInput updates the model's state based on a
// different set of keypresses.
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

// unsort updates the model's state to the original list of items.
func (m model) unsort(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.sorted = false
	customKeys.Sort.SetHelp("r", "recently used")
	m.list.SetItems(m.originalItems)
	m.list.ResetSelected()
	return m, nil
}

// sort updates the model's state to the sorted list of items.
// The sorted list is based off of what was stored on disk.
func (m model) sort(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.sorted = true
	customKeys.Sort.SetHelp("r", "revert to default")
	m.list.SetItems(m.sortedItems)
	m.list.ResetSelected()
	return m, nil
}

// recordConnection adjusts the sorted list of items to bring
// to the front the most recently chosen item and writes the
// result to disk.
func (m model) recordConnection(i Item) (tea.Model, tea.Cmd) {
	items := timestampFirstItem(itemToFront(m.sortedItems, i))
	itemsToJson(m.recentlyUsedPath, items, true)
	return m, tea.Quit
}
