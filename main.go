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
	sshConfigPath        = expandTilde("~/.ssh/config")
	recentlyUsedPath     = expandTilde("~/.ssh/recent.json")
	sshControlPath       = "/dev/shm/control:%h:%p:%r"
	sshControlChildOpts  = []string{"-S", sshControlPath}
	sshControlParentOpts = []string{"-T", "-o", "ControlMaster=yes", "-o", "ControlPersist=5s", "-o", fmt.Sprintf("ControlPath=%s", sshControlPath)}
)

func main() {
	sshExecutablePath, err := exec.LookPath(sshExecutableName)
	// Using 'panic()' as it's supposedly acceptable during initialization phases:
	// https://go.dev/doc/effective_go#panic
	if err != nil {
		panic(err)
	}

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
		fmt.Println(m.connection.output)

		args := append([]string{sshExecutableName, m.choice}, sshControlChildOpts...)
		err := syscall.Exec(sshExecutablePath, args, os.Environ())
		if err != nil {
			fmt.Println("unable to run executable: %w", err)
			os.Exit(1)
		}
	} else if m.err != "" {
		fmt.Println("unable to connect: %w", m.err)
		os.Exit(1)
	}
}
