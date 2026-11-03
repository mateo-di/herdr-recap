package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
)

// A real process at the provider boundary records what Recap asks of it.
func languageCommand(t *testing.T, body string) (binary, request string) {
	t.Helper()
	dir := t.TempDir()
	binary, request = filepath.Join(dir, "provider"), filepath.Join(dir, "request")
	t.Setenv("RECAP_TEST_REQUEST", request)
	write(t, binary, "#!/bin/sh\nset -eu\n"+body)
	if err := os.Chmod(binary, 0o700); err != nil {
		t.Fatal(err)
	}
	return binary, request
}

func TestClaudeRecapRequestsTheConfiguredLanguageInItsOwnFork(t *testing.T) {
	binary, request := languageCommand(t, `printf '%s\000' "$@" > "$RECAP_TEST_REQUEST"
printf '{"result":"正在修复登录问题，下一步运行测试。"}'
`)
	s := testSessionWith(t, "one\n")
	s.Cwd = t.TempDir()
	r, err := runRecap(context.Background(), withDefaults(config{Claude: binary, Language: "zh-CN"}), s)
	if err != nil || r.Text != "正在修复登录问题，下一步运行测试。" {
		t.Fatalf("%+v, %v", r, err)
	}
	raw, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	for _, want := range []string{s.ID, "--resume", "--fork-session", "--no-session-persistence", "json"} {
		if !slices.Contains(args, want) {
			t.Errorf("request lacks %q: %q", want, args)
		}
	}
	if prompt := args[len(args)-1]; !strings.Contains(prompt, "zh-CN") || !strings.Contains(prompt, "Don't run any commands or use any tools") {
		t.Errorf("language or read-only instruction missing: %q", prompt)
	}
}

func TestCodexRecapRequestsTheConfiguredLanguageWithoutSavingOrWriting(t *testing.T) {
	binary, request := languageCommand(t, `printf '%s\000' "$@" > "$RECAP_TEST_REQUEST"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    printf '正在修复登录问题。' > "$2"
    exit 0
  fi
  shift
done
exit 1
`)
	s := testSessionWith(t, "one\n")
	s.Agent, s.Native = "codex", true
	r, err := runRecap(context.Background(), withDefaults(config{Codex: binary, Language: "zh-CN"}), s)
	if err != nil || r.Text != "正在修复登录问题。" {
		t.Fatalf("%+v, %v", r, err)
	}
	raw, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	for _, want := range []string{"fork", s.ID, "--ephemeral", `sandbox_mode="read-only"`} {
		if !slices.Contains(args, want) {
			t.Errorf("request lacks %q: %q", want, args)
		}
	}
	if prompt := args[len(args)-1]; !strings.Contains(prompt, "zh-CN") || !strings.Contains(prompt, "Don't run any commands or use any tools") {
		t.Errorf("language or read-only instruction missing: %q", prompt)
	}
}

func TestMemexSummarizersReceiveTheLanguageOutsideTheConversation(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name, reply := "default", `printf '{"result":"正在修复登录问题。"}'`
		if custom {
			name, reply = "custom", "printf '正在修复登录问题。'"
		}
		t.Run(name, func(t *testing.T) {
			binary, request := languageCommand(t, "cat > \"$RECAP_TEST_REQUEST\"\n"+reply+"\n")
			memex := filepath.Join(filepath.Dir(binary), "memex")
			write(t, memex, "#!/bin/sh\nprintf '%s' '[{\"record\":{\"role\":\"user\",\"text\":\"fix login\"}}]'\n")
			if err := os.Chmod(memex, 0o700); err != nil {
				t.Fatal(err)
			}
			cfg := withDefaults(config{Claude: binary, Memex: memex, Language: "zh-CN"})
			if custom {
				cfg.Summarizer = []string{binary}
			}
			s := claudeSession{Agent: "hermes", ID: testSession, Transcript: "/session", MessageCount: 1}
			r, err := runRecap(context.Background(), cfg, s)
			if err != nil || r.Text != "正在修复登录问题。" {
				t.Fatalf("%+v, %v", r, err)
			}
			raw, err := os.ReadFile(request)
			if err != nil {
				t.Fatal(err)
			}
			prompt := string(raw)
			start := strings.IndexByte(prompt, '{')
			if start < 0 || !strings.Contains(prompt[:start], "zh-CN") {
				t.Fatalf("language is absent from the trusted instructions: %q", prompt)
			}
			var payload struct{ Agent, Conversation string }
			if err := json.Unmarshal(raw[start:], &payload); err != nil || payload.Agent != "hermes" || payload.Conversation != "USER: fix login" {
				t.Fatalf("conversation envelope changed: %+v, %v", payload, err)
			}
		})
	}
}

func TestSuggestedTitlesUseTheConfiguredLanguageForBothNativeAgents(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			binary, request := languageCommand(t, `printf '%s\000' "$@" > "$RECAP_TEST_REQUEST"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    printf 'HAL-12 修复登录问题' > "$2"
    exit 0
  fi
  shift
done
printf '{"result":"HAL-12 修复登录问题"}'
`)
			s := testSessionWith(t, "one\n")
			s.Agent, s.Native, s.Cwd = agent, agent == "codex", t.TempDir()
			title, _, err := suggestTitle(context.Background(), withDefaults(config{Claude: binary, Codex: binary, Language: "zh-CN"}), s, "HAL-12 Fix Login")
			if err != nil || title != "HAL-12 修复登录问题" {
				t.Fatalf("%q, %v", title, err)
			}
			raw, err := os.ReadFile(request)
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
			prompt := args[len(args)-1]
			if !strings.Contains(prompt, "zh-CN") || strings.Contains(prompt, "in Title Case") || !strings.Contains(prompt, "already in the requested language") {
				t.Errorf("title request does not honor the chosen language: %q", prompt)
			}
		})
	}
}

func TestChangingLanguageRefreshesAnUnchangedConversation(t *testing.T) {
	calls := withClaude(t, okReply)
	s := testSessionWith(t, "one\n")
	if _, _, err := ensureRecap(context.Background(), withDefaults(config{}), s, "Fix login", false); err != nil {
		t.Fatal(err)
	}
	// An old cache without language metadata still works with the default.
	if _, current := (liveSource{withDefaults(config{})}).cached(s); !current {
		t.Fatal("the default no longer accepts its existing cache")
	}
	for i, language := range []string{"zh-CN", "Japanese", ""} {
		cfg := withDefaults(config{Language: language})
		if _, current := (liveSource{cfg}).cached(s); current {
			t.Fatalf("old-language cache is current after switching to %q", language)
		}
		if _, ran, err := ensureRecap(context.Background(), cfg, s, "Fix login", false); err != nil || !ran {
			t.Fatalf("switch to %q: ran %v, error %v", language, ran, err)
		}
		if _, current := (liveSource{cfg}).cached(s); !current {
			t.Fatalf("new-language cache is stale for %q", language)
		}
		if _, ran, err := ensureRecap(context.Background(), cfg, s, "Fix login", false); err != nil || ran || calls.Load() != int32(i+2) {
			t.Fatalf("repeated %q: ran %v, error %v, calls %d", language, ran, err, calls.Load())
		}
	}
}

func languageReplies(t *testing.T) *atomic.Int32 {
	t.Helper()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", t.TempDir())
	var titles atomic.Int32
	old := runCommand
	runCommand = func(_ context.Context, cfg config, _ claudeSession, args []string) ([]byte, error) {
		if strings.Contains(args[len(args)-1], "Suggest a name for this session") {
			titles.Add(1)
			if cfg.Language == "zh-CN" {
				return []byte(`{"result":"HAL-12 修复登录问题"}`), nil
			}
			return []byte(titleReply), nil
		}
		return []byte(okReply), nil
	}
	t.Cleanup(func() { runCommand = old })
	return &titles
}

func TestLanguageChangesRefreshSuggestionsButKeepTitleDecisions(t *testing.T) {
	for _, state := range []string{"suggested", "settled", "pending"} {
		t.Run(state, func(t *testing.T) {
			titles := languageReplies(t)
			s := testSessionWith(t, "one\n")
			if _, _, err := ensureRecap(context.Background(), withDefaults(config{}), s, "fix-login-bug", false); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "settled":
				if err := settleTitle(s.ID); err != nil {
					t.Fatal(err)
				}
			case "pending":
				if err := askRename("w1:p1", s.ID); err != nil {
					t.Fatal(err)
				}
			}
			cfg := withDefaults(config{Language: "zh-CN"})
			cached, _ := (liveSource{cfg}).cached(s)
			if state == "suggested" && cached.Title != "" {
				t.Fatalf("an old-language suggestion is still offered: %q", cached.Title)
			}
			r, ran, err := ensureRecap(context.Background(), cfg, s, "fix-login-bug", false)
			if err != nil || !ran {
				t.Fatalf("ran %v, error %v", ran, err)
			}
			if state == "suggested" {
				if r.Title != "HAL-12 修复登录问题" || titles.Load() != 2 {
					t.Fatalf("old suggestion was kept: %+v, %d titles", r, titles.Load())
				}
			} else if r.Title != "HAL-12 Fix the Login Bug" || titles.Load() != 1 || r.Settled != (state == "settled") || r.Renaming != (state == "pending") {
				t.Fatalf("the user's title decision changed: %+v, %d titles", r, titles.Load())
			}
		})
	}
}

func TestAskingOnlyForATitleDoesNotChangeTheRecapsLanguage(t *testing.T) {
	titles := languageReplies(t)
	s := testSessionWith(t, "one\n")
	if _, _, err := ensureRecap(context.Background(), withDefaults(config{}), s, "Fix login", false); err != nil {
		t.Fatal(err)
	}
	cfg := withDefaults(config{Language: "zh-CN"})
	if _, err := ensureTitle(context.Background(), cfg, s, "Fix login"); err != nil {
		t.Fatal(err)
	}
	if _, current := (liveSource{cfg}).cached(s); current {
		t.Fatal("a Chinese title made the old-language recap current")
	}
	if _, current := (liveSource{withDefaults(config{})}).cached(s); !current {
		t.Fatal("asking for a title changed the recap's provenance")
	}
	r, ran, err := ensureRecap(context.Background(), cfg, s, "fix-login-bug", false)
	if err != nil || !ran || r.Title != "HAL-12 修复登录问题" || titles.Load() != 1 {
		t.Fatalf("the new-language suggestion wasn't kept: %+v, ran %v, error %v, titles %d", r, ran, err, titles.Load())
	}
}

func TestTheAppBackendSchedulesUnchangedSessionsWhenLanguageChanges(t *testing.T) {
	languageReplies(t)
	dir := claudeDir(t, "4242", t.TempDir(), true)
	withEnv(t, map[string]string{"CLAUDE_CONFIG_DIR": dir}, nil)
	s, err := sessionForPID(4242, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ensureRecap(context.Background(), withDefaults(config{}), s, "Fix login", false); err != nil {
		t.Fatal(err)
	}
	oldSpawn := spawn
	var spawned [][]string
	spawn = func(args ...string) error { spawned = append(spawned, args); return nil }
	t.Cleanup(func() { spawn = oldSpawn })
	out, err := os.CreateTemp(t.TempDir(), "sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	stdout := os.Stdout
	os.Stdout = out
	t.Cleanup(func() { os.Stdout = stdout })
	err = runSessions(context.Background(), withDefaults(config{Language: "zh-CN"}), []string{"--write", `[{"pid":4242,"title":"Fix login"}]`})
	os.Stdout = stdout
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	var sessions []sessionInfo
	if err := json.NewDecoder(out).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Fresh || len(spawned) != 1 || len(spawned[0]) != 3 || spawned[0][0] != "sessions" || spawned[0][1] != "--now" {
		t.Fatalf("app did not schedule the old-language recap: %+v, spawned %q", sessions, spawned)
	}
}
