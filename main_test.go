package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
)

func TestSshConfigHosts(t *testing.T) {
	cases := []struct {
		Description, FilePath string
		Want                  int
	}{
		{"good", "testdata/good", 11},
		{"commented", "testdata/commented", 2},
		{"invalid", "testdata/invalid", 1},
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
			if hosts[i].(Item).Host != expected[i].(Item).Host || hosts[i].(Item).Hostname != expected[i].(Item).Hostname {
				t.Errorf("got %s, wanted %s", hosts[i], expected[i])
			}
		}
	})
	t.Run("expected hosts 'includedTopLevel'", func(t *testing.T) {
		var hosts []list.Item
		expected := []list.Item{
			Item{Host: "saturday", Hostname: "saturday.local", SourceFile: "testdata/included1", SourceLine: 1},
			Item{Host: "sunday", Hostname: "sunday.local", SourceFile: "testdata/included1", SourceLine: 4},
			Item{Host: "lodestar", Hostname: "lodestar.local", SourceFile: "testdata/included1", SourceLine: 7},
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
			Item{Host: "chat.local", Hostname: "chat", SourceFile: "testdata/inifile"},
			Item{Host: "turn.local", Hostname: "turn", SourceFile: "testdata/inifile"},
			Item{Host: "lieu.local", Hostname: "lieu.local", SourceFile: "testdata/inifile"},
			Item{Host: "vt.local", Hostname: "vt.local", SourceFile: "testdata/inifile"},
			Item{Host: "graph.local", Hostname: "graph", SourceFile: "testdata/inifile"},
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

func TestFindIncludedFiles(t *testing.T) {
	cases := []struct {
		Description, Content string
		Want                 int
	}{
		{"regular", "Include testdata/included1\n", 1},
		{"case-insensitive", "include testdata/included1\n", 1},
		{"without a value", "Include\nHost foo\n", 0},
		{"with only trailing whitespace", "Include \nHost foo\n", 0},
	}
	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			filePaths, count := findIncludedFiles([]byte(test.Content))

			if len(filePaths) != test.Want || count != test.Want {
				t.Errorf("got %d paths and a count of %d, wanted %d", len(filePaths), count, test.Want)
			}
		})
	}
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
		items := findHosts(content, "")
		if len(items) != len(expected) {
			t.Fatalf("got %d, wanted %d", len(items), len(expected))
		}
		for i := range items {
			if items[i].(Item).Host != expected[i].(Item).Host || items[i].(Item).Hostname != expected[i].(Item).Hostname {
				t.Errorf("got %s, wanted %s", items[i], expected[i])
			}
		}
	})
	cases := []struct {
		Description, Content string
		Want                 []Item
	}{
		{
			"adjacent blocks without blank lines",
			"Host web\n\tHostName web.example.com\nHost db\n\tHostName db.example.com\n",
			[]Item{
				{Host: "web", Hostname: "web.example.com", SourceLine: 1},
				{Host: "db", Hostname: "db.example.com", SourceLine: 3},
			},
		},
		{
			"consecutive hosts without options",
			"Host foo\nHost bar\nHost baz\n",
			[]Item{
				{Host: "foo", Hostname: "foo", SourceLine: 1},
				{Host: "bar", Hostname: "bar", SourceLine: 2},
				{Host: "baz", Hostname: "baz", SourceLine: 3},
			},
		},
		{
			"wildcard and negated patterns are skipped",
			"Host *\n\tUser admin\n\nHost prod-*\n\nHost ?db\n\nHost !negated\n",
			nil,
		},
		{
			"multiple aliases on one line",
			"Host web1 web2 web3\n\tHostName web.example.com\n",
			[]Item{
				{Host: "web1", Hostname: "web.example.com", SourceLine: 1},
				{Host: "web2", Hostname: "web.example.com", SourceLine: 1},
				{Host: "web3", Hostname: "web.example.com", SourceLine: 1},
			},
		},
		{
			"hostname from a later block",
			"Host multi\n\tUser someone\n\nHost multi\n\tHostName multi.local\n",
			[]Item{
				{Host: "multi", Hostname: "multi.local", SourceLine: 1},
			},
		},
		{
			"missing separator falls back to host",
			"Host invalid\n  HostNameinvalid-because-no-spaces\n",
			[]Item{
				{Host: "invalid", Hostname: "invalid", SourceLine: 1},
			},
		},
		{
			"windows line endings",
			"Host a\r\n\tHostName same.local\r\n\tPort 22\r\n\r\nHost b\r\n\tHostName same.local\r\n\tPort 23\r\n",
			[]Item{
				{Host: "a", Hostname: "same.local", Extra: "Port 22", SourceLine: 1},
				{Host: "b", Hostname: "same.local", Extra: "Port 23", SourceLine: 5},
			},
		},
		{
			"case-insensitive keywords",
			"host foo\n\thostname foo.local\n",
			[]Item{
				{Host: "foo", Hostname: "foo.local", SourceLine: 1},
			},
		},
		{
			"keyword and value separated by equals sign",
			"Host=foo\n\tHostName = foo.local\n",
			[]Item{
				{Host: "foo", Hostname: "foo.local", SourceLine: 1},
			},
		},
	}
	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			items := findHosts([]byte(test.Content), "")

			if len(items) != len(test.Want) {
				t.Fatalf("got %d, wanted %d", len(items), len(test.Want))
			}
			for i := range items {
				if items[i].(Item) != test.Want[i] {
					t.Errorf("got %v, wanted %v", items[i], test.Want[i])
				}
			}
		})
	}
}
