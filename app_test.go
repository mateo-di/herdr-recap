package main

import (
	"os"
	"regexp"
	"testing"
)

// The Herdr app's mod fetches the herdr-recap release it names: it has to
// name this one, or it fetches a binary that isn't this code.
func TestAppModNamesThisRelease(t *testing.T) {
	toml, err := os.ReadFile("herdr-plugin.toml")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile("app/mods/recap/main.js")
	if err != nil {
		t.Fatal(err)
	}
	want := regexp.MustCompile(`(?m)^version *= *"([^"]+)"`).FindSubmatch(toml)
	got := regexp.MustCompile(`RECAP_VERSION = "([^"]+)"`).FindSubmatch(mod)
	if want == nil || got == nil {
		t.Fatalf("version not found: toml %q, mod %q", want, got)
	}
	if string(got[1]) != string(want[1]) {
		t.Fatalf("the app mod fetches herdr-recap %s, but this is %s: set RECAP_VERSION in app/mods/recap/main.js", got[1], want[1])
	}
}
