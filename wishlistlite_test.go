package main

import (
	"log"
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
	t.Run("expected hosts", func(t *testing.T) {
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
