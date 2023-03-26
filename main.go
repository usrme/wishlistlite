package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// sshExecutableName is the name of the SSH executable present on the local system.
const sshExecutableName = "ssh"

var (
	sshConfigPath        = fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/config")
	recentlyUsedPath     = fmt.Sprintf("%s/%s", userHomeDir(), ".ssh/recent.json")
	sshControlPath       = "/dev/shm/control:%h:%p:%r"
	sshControlChildOpts  = []string{"-S", sshControlPath}
	sshControlParentOpts = []string{"-o", "ControlMaster=yes", "-o", "ControlPersist=5s", "-o", fmt.Sprintf("ControlPath=%s", sshControlPath)}
)

func main() {
	sshExecutablePath := verifyExecutable(sshExecutableName)

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

	m, err := p.Run()
	if err != nil {
		fmt.Println("failed to execute: %w", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.choice != "" {
		fmt.Printf("Connected in %v\n", m.connection.startupTime)
		runExecutable(sshExecutablePath, append([]string{sshExecutableName, m.choice}, sshControlChildOpts...))
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
