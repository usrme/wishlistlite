package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestSshConfigHosts(t *testing.T) {
	cases := []struct {
		Description, FilePath string
		Want                  int
	}{
		{"good", "testdata/good", 11},
		{"commented", "testdata/commented", 2},
		{"invalid", "testdata/invalid", 0},
		{"includedTop", "testdata/includedTop", 3},
	}
	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			hosts, err := sshConfigHosts(test.FilePath)

			if err != nil {
				t.Fatal(err)
			}

			got := len(hosts)
			if got != test.Want {
				log.Printf("%v", hosts)
				t.Errorf("got %d, wanted %d", got, test.Want)
			}
		})
	}
	t.Run("expected hosts 'good'", func(t *testing.T) {
		expected := []list.Item{
			Item{Host: "darkstar", Hostname: "darkstar.local"},
			Item{Host: "supernova", Hostname: "supernova.local"},
			Item{Host: "app1", Hostname: "app.foo.local"},
			Item{Host: "app2", Hostname: "app.foo.local"},
			Item{Host: "multiple1", Hostname: "multi1.foo.local"},
			Item{Host: "multiple2", Hostname: "multi2.foo.local"},
			Item{Host: "multiple3", Hostname: "multi3.foo.local"},
			Item{Host: "no.hostname", Hostname: "no.hostname"},
			Item{Host: "req.tty", Hostname: "req.tty"},
			Item{Host: "remote.cmd", Hostname: "remote.cmd"},
			Item{Host: "only.host", Hostname: "only.host"},
		}
		hosts, err := sshConfigHosts("testdata/good")

		if err != nil {
			t.Fatal(err)
		}

		for i := range hosts {
			if hosts[i].(Item).Host != expected[i].(Item).Host && hosts[i].(Item).Hostname != expected[i].(Item).Hostname {
				t.Errorf("got %s, wanted %d", hosts[i], expected[i])
			}
		}
	})
	t.Run("expected hosts 'includedTopLevel'", func(t *testing.T) {
		var hosts []list.Item
		expected := []list.Item{
			Item{Host: "saturday", Hostname: "saturday.local"},
			Item{Host: "sunday", Hostname: "sunday.local"},
			Item{Host: "lodestar", Hostname: "lodestar.local"},
		}
		hosts, err := sshConfigHosts("testdata/includedTop")

		if err != nil {
			t.Fatal(err)
		}
		if len(hosts) != len(expected) {
			t.Fatalf("got %d, wanted %d", len(hosts), len(expected))
		}
		for i := range hosts {
			if hosts[i] != expected[i] {
				t.Errorf("got %s, wanted %d", hosts[i], expected[i])
			}
		}
	})
	t.Run("expected hosts 'inifile'", func(t *testing.T) {
		var hosts []list.Item
		expected := []list.Item{
			Item{Host: "chat.local", Hostname: "chat"},
			Item{Host: "turn.local", Hostname: "turn"},
			Item{Host: "lieu.local", Hostname: "lieu.local"},
			Item{Host: "vt.local", Hostname: "vt.local"},
			Item{Host: "graph.local", Hostname: "graph"},
		}
		hosts, err := iniHosts("testdata/inifile", false)

		if err != nil {
			t.Fatal(err)
		}
		if len(hosts) != len(expected) {
			t.Fatalf("got %d, wanted %d", len(hosts), len(expected))
		}
		for i := range hosts {
			if hosts[i] != expected[i] {
				t.Errorf("got %s, wanted %d", hosts[i], expected[i])
			}
		}
	})
}

func TestMoveToFront(t *testing.T) {
	cases := []struct {
		Description, Needle string
		Haystack, Want      []string
	}{
		{"add if empty", "a", []string{}, []string{"a"}},
		{"return same", "a", []string{"a"}, []string{"a"}},
		{"move to front", "c", []string{"a", "b", "c", "d", "e"}, []string{"c", "a", "b", "d", "e"}},
		{"prepend if missing", "f", []string{"a", "b", "c", "d", "e"}, []string{"f", "a", "b", "c", "d", "e"}},
	}
	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			got := moveToFront(test.Needle, test.Haystack)

			if len(got) != len(test.Want) {
				t.Errorf("got %d, wanted %d", len(got), len(test.Want))
			}

			for i, v := range got {
				if v != test.Want[i] {
					log.Println(got)
					t.Errorf("got %s, wanted %s", v, test.Want[i])
				}
			}
		})
	}
}

func TestItemToFront(t *testing.T) {
	cases := []struct {
		Description string
		Item        Item
		Have, Want  []list.Item
	}{
		{
			"without timestamp",
			Item{Host: "supernova", Hostname: "supernova.local"},
			[]list.Item{
				Item{Host: "darkstar", Hostname: "darkstar.local"},
				Item{Host: "supernova", Hostname: "supernova.local"},
			},
			[]list.Item{
				Item{Host: "supernova", Hostname: "supernova.local"},
				Item{Host: "darkstar", Hostname: "darkstar.local"},
			},
		},
		{
			"with timestamp",
			Item{Host: "supernova", Hostname: "supernova.local"},
			[]list.Item{
				Item{Host: "darkstar", Hostname: "darkstar.local"},
				Item{Host: "supernova", Hostname: "supernova.local", Timestamp: "Sun, 12 Jun 2022 14:59:28 EEST"},
			},
			[]list.Item{
				Item{Host: "supernova", Hostname: "supernova.local", Timestamp: "Sun, 12 Jun 2022 14:59:28 EEST"},
				Item{Host: "darkstar", Hostname: "darkstar.local"},
			},
		},
		{
			"new host",
			Item{Host: "battlestar", Hostname: "battlestar.local"},
			[]list.Item{
				Item{Host: "darkstar", Hostname: "darkstar.local"},
				Item{Host: "supernova", Hostname: "supernova.local", Timestamp: "Sun, 12 Jun 2022 14:59:28 EEST"},
			},
			[]list.Item{
				Item{Host: "battlestar", Hostname: "battlestar.local"},
				Item{Host: "supernova", Hostname: "supernova.local", Timestamp: "Sun, 12 Jun 2022 14:59:28 EEST"},
				Item{Host: "darkstar", Hostname: "darkstar.local"},
			},
		},
	}
	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			got := itemToFront(test.Have, test.Item)

			if len(got) != len(test.Want) {
				t.Errorf("got %d, wanted %d", len(got), len(test.Want))
			}

			if got[0] != test.Want[0] {
				log.Println(got)
				t.Errorf("got %s, wanted %s", got[0], test.Want[0])
			}
		})
	}
}

func TestItemsFromJson(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := itemsFromJson("testdata/missing.json")
		if !strings.HasPrefix(fmt.Sprint(err), "could not read file") {
			t.Fatal(err)
		}
	})
	t.Run("invalid JSON", func(t *testing.T) {
		_, err := itemsFromJson("testdata/invalid")
		if !strings.HasPrefix(fmt.Sprint(err), "could not unmarshal JSON") {
			t.Fatal(err)
		}
	})
	t.Run("expected", func(t *testing.T) {
		expected := []list.Item{
			Item{Host: "supernova", Hostname: "supernova.local", Timestamp: "Sun, 12 Jun 2022 14:59:28 EEST"},
			Item{Host: "darkstar", Hostname: "darkstar.local"},
			Item{Host: "app1", Hostname: "app.foo.local"},
		}
		sorted, err := itemsFromJson("testdata/recent.json")
		if err != nil {
			t.Fatal(err)
		}
		for i := range sorted {
			if sorted[i] != expected[i] {
				t.Errorf("got %s, wanted %d", sorted[i], expected[i])
			}
		}
	})
}

func TestFindHosts(t *testing.T) {
	t.Run("no duplicates", func(t *testing.T) {
		filePath := "testdata/duplicate"
		filePath = expandTilde(filePath)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}
		expected := []list.Item{
			Item{Host: "saturday1", Hostname: "saturday1.local"},
			Item{Host: "saturday2", Hostname: "saturday.local"},
			Item{Host: "sunday", Hostname: "sunday.local"},
		}
		items := findHosts(content)
		if len(items) != len(expected) {
			t.Fatalf("got %d, wanted %d", len(items), len(expected))
		}
		for i := range items {
			if items[i].(Item).Host != expected[i].(Item).Host && items[i].(Item).Hostname != expected[i].(Item).Hostname {
				t.Errorf("got %s, wanted %d", items[i], expected[i])
			}
		}
	})
}
