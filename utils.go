package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

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

func expandTilde(filePath string) string {
	if filePath[0] == '~' {
		return fmt.Sprintf("%s/%s", userHomeDir(), filePath[1:])
	}
	return filePath
}

var allItems [][]list.Item

func sshConfigHosts(filePath string) ([]list.Item, error) {
	filePath = expandTilde(filePath)

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s': %w", filePath, err)
	}

	filePaths, includeCount := findIncludedFiles(content)

	for _, i := range filePaths {
		j, err := sshConfigHosts(i)
		if err == nil {
			allItems = append(allItems, j)
		} else {
			continue
		}
	}

	items := findHosts(content)

	// this should only trigger when at the top level, thus
	// causing a return of all found hosts from the main file
	// and from any included files if there were any
	if len(allItems) == includeCount && includeCount != 0 {
		allItems = append(allItems, items)
		return flatten(allItems), nil
	}

	// this should trigger when looking inside of an included
	// file or when there were no included files
	return items, nil
}

func findIncludedFiles(content []byte) ([]string, int) {
	var (
		pat          *regexp.Regexp
		filePaths    []string
		includeCount int
	)

	pat = regexp.MustCompile(`(?m)^Include\s([a-zA-Z0-9_\-\.\~\*\/]*)`)
	includeMatches := pat.FindAllStringSubmatch(string(content), -1)

	for _, i := range includeMatches {
		// if an 'Include' value's (i[1]) last character (i[1][len(i[1])-1]) is a wildcard
		if i[1][len(i[1])-1] == '*' {
			i[1] = expandTilde(i[1])
			matches, _ := filepath.Glob(i[1])
			// add all the globbed matches
			filePaths = append(filePaths, matches...)
			includeCount += len(matches)
		} else {
			// add whatever was the 'Include' value
			filePaths = append(filePaths, i[1])
			includeCount += 1
		}
	}

	return filePaths, includeCount
}

func findHosts(content []byte) []list.Item {
	// grab all 'Host' ('Host' not included) and 'HostName' ('HostName' included)
	pat := regexp.MustCompile(`(?m)^Host\s([^\*][a-zA-Z0-9_\.-]*)[\r\n](\s+HostName.*)?`)
	mainMatches := pat.FindAllStringSubmatch(string(content), -1)

	var items []list.Item
	for _, m := range mainMatches {
		// if 'HostName' was present
		if m[2] != "" {
			// make sure 'HostName' was defined correctly (i.e. followed by a space)
			pat := regexp.MustCompile(`HostName\s(.*)`)
			for _, n := range pat.FindAllStringSubmatch(m[2], -1) {
				items = append(items, Item{Host: m[1], Hostname: n[1]})
			}
		} else {
			items = append(items, Item{Host: m[1], Hostname: m[1]})
		}
	}

	return items
}

func flatten[T any](lists [][]T) []T {
	var res []T
	for _, list := range lists {
		res = append(res, list...)
	}
	return res
}

func itemsToJson(filePath string, l []list.Item, overwrite bool) error {
	result, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("could not marshal JSON: %w", err)
	}
	// Only write file if it doesn't already exist or an explicit overwrite was requested
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

func itemToFront(sorted []list.Item, i Item) []list.Item {
	var sortedHostSlice []string         // slice for sorting
	hostMapBool := make(map[string]bool) // map for checking whether element already exists
	hostMap := make(map[string]Item)     // map for quickly getting attributes of a single item

	for _, host := range sorted {
		hostShort := host.(Item).Host
		hostMapBool[hostShort] = true
		hostMap[hostShort] = host.(Item)
	}

	// if in ad hoc connections the input host is an already known host,
	// then rewrite the incoming Item's fields to match existing host's
	if _, ok := hostMapBool[i.Host]; ok {
		existing := hostMap[i.Host]
		i.Hostname = existing.Hostname
	}

	for _, host := range sorted {
		hostShort := host.(Item).Host
		sortedHostSlice = append(sortedHostSlice, hostShort)
	}
	sortedHostSlice = moveToFront(i.Host, sortedHostSlice)

	var (
		items []list.Item
		item  list.Item
	)

	for _, hostShort := range sortedHostSlice {
		if _, ok := hostMapBool[hostShort]; !ok && hostShort == i.Host {
			item = Item{Host: i.Host, Hostname: i.Hostname, Timestamp: i.Timestamp}
		} else {
			c := hostMap[hostShort]
			item = Item{Host: c.Host, Hostname: c.Hostname, Timestamp: c.Timestamp}
		}
		items = append(items, item)
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
