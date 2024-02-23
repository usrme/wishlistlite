// Package wishlistlite is a pared down version of Charm's Wishlist.
//
// It leverages SSH-related executables already present on the local system to
// simplify everything and relies on just regular expressions to parse an SSH
// configuration.
//
// Its aim was to provide a more hands-on way to learn about Go and isn't to be
// taken seriously.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// sshExecutableName is the name of the SSH executable present on the local system.
const sshExecutableName = "ssh"

func newPingOpts(count int) []string {
	return []string{"-c", fmt.Sprint(count)}
}

// Paths, 'ping' and SSH control options used by package.
var (
	defaultSshConfigPath    = expandTilde("~/.ssh/config")
	defaultRecentlyUsedPath = expandTilde("~/.ssh/recent.json")
	sshControlPath          = "/dev/shm/control:%h:%p:%r"
	sshControlChildOpts     = []string{"-S", sshControlPath}
	sshControlParentOpts    = []string{"-T", "-o", "ControlMaster=auto", "-o", "ControlPersist=5s", "-o", fmt.Sprintf("ControlPath=%s", sshControlPath)}
	defaultPingCount        = 4
	pingOpts                = newPingOpts(defaultPingCount)
)

func main() {
	sshConfigPath := flag.String("sshconfigpath", defaultSshConfigPath, "Path to SSH configuration file")
	recentlyUsedPath := flag.String("recentlyusedpath", defaultRecentlyUsedPath, "Path to recent SSH connections file")
	iniFilePath := flag.String("inifilepath", "", "Path to INI file path (e.g. Ansible inventory file) in lieu of SSH configuration file")
	switchFilter := flag.Bool("switchfilter", false, "Whether or not to switch filter value from host to hostname")
	pingCount := flag.Int("pingcount", defaultPingCount, "Number of times a host should be pinged")
	sshOpts := flag.String("sshoptions", "", "Additional options passed to SSH. Must be contained in quotes")
	flag.Parse()

	if *pingCount != defaultPingCount {
		pingOpts = newPingOpts(*pingCount)
	}

	sshExecutablePath, err := exec.LookPath(sshExecutableName)
	// Using 'panic()' as it's supposedly acceptable during initialization phases:
	// https://go.dev/doc/effective_go#panic
	if err != nil {
		panic(err)
	}

	var items []list.Item
	if *iniFilePath != "" {
		items, err = iniHosts(*iniFilePath, *switchFilter)
		if err != nil {
			fmt.Println("failed to read input file: %w", err)
			os.Exit(1)
		}
	} else {
		items, err = sshConfigHosts(*sshConfigPath)
		if err != nil {
			fmt.Println("failed to read SSH configuration: %w", err)
			os.Exit(1)
		}
	}

	sortedItems, err := itemsFromJson(*recentlyUsedPath)
	if err != nil {
		sortedItems = []list.Item{}
	}
	sshopts := strings.Split(*sshOpts, " ")
	if *sshOpts == "" {
		sshopts = []string{}
	}
	p := tea.NewProgram(newModel(items, sortedItems, *recentlyUsedPath, pingOpts, sshopts), tea.WithAltScreen())

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
		fmt.Printf("unable to connect: %s", m.err)
		os.Exit(1)
	}
}
