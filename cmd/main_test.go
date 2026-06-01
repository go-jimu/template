package main

import (
	"os"
	"testing"
)

// Default config must parse before fx startup so YAML duration strings do not abort boot.
func TestParseOptionLoadsDefaultConfig(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err = parseOption(); err != nil {
		t.Fatal(err)
	}
}
