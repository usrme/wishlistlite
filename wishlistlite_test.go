package main

import (
	"log"
	"testing"
)

func TestSshConfigHosts(t *testing.T) {
	cases := []struct {
		Description, FilePath string
		Want                  int
	}{
		{"good", "testdata/good", 11},
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
}
