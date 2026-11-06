package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWarmIsWithinThePromptCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.jsonl")
	write(t, path, "{}\n")
	s := claudeSession{Transcript: path}
	if !warm(s) {
		t.Error("just written")
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if warm(s) {
		t.Error("two hours old")
	}
	if warm(claudeSession{}) || warm(claudeSession{Transcript: path + ".gone"}) {
		t.Error("no transcript")
	}
}
