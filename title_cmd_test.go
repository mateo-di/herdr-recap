package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

// The app's ctrl+r on an agent with no suggestion: a copy of the program
// asks for the title in the background, and the call answers at once.
func TestTitleCommandAsksInTheBackground(t *testing.T) {
	old := spawn
	var spawned [][]string
	spawn = func(args ...string) error { spawned = append(spawned, args); return nil }
	t.Cleanup(func() { spawn = old })

	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	err := runTitle(context.Background(), config{}, []string{`[{"pid": 42, "title": "fix-thing"}]`})
	w.Close()
	os.Stdout = stdout
	out, _ := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(spawned) != 1 || strings.Join(spawned[0], " ") != `title --now [{"pid": 42, "title": "fix-thing"}]` {
		t.Fatalf("spawned %q", spawned)
	}
	if strings.TrimSpace(string(out)) != `{"asked":1}` {
		t.Fatalf("answered %q", out)
	}
	if err := runTitle(context.Background(), config{}, nil); err == nil {
		t.Fatal("no argument should be an error")
	}
}
