package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// sshExecutable is the name of the SSH executable present on the local system.
const sshExecutable = "ssh"

var (
	sshConfigPath    = fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	recentlyUsedPath = fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/recent.json")
)

func main() {
	execPath := verifyExecutable(sshExecutable)

	items, err := sshConfigHosts(sshConfigPath)
	if err != nil {
		fmt.Println("failed to read SSH configuration: %w", err)
		os.Exit(1)
	}

	itemsToJson(recentlyUsedPath, items, false)

	sortedItems, err := itemsFromJson(recentlyUsedPath)
	if err != nil {
		fmt.Println("failed to sort items: %w", err)
		os.Exit(1)
	}
	p := tea.NewProgram(newModel(items, sortedItems), tea.WithAltScreen())

	m, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("failed to execute: %w", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.choice != "" {
		runExecutable(execPath, []string{sshExecutable, m.choice})
	}
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
