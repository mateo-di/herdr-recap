package main

import (
	"path/filepath"
	"testing"
)

func TestLanguageCanBeConfiguredWithoutChangingTheDefault(t *testing.T) {
	for _, c := range []struct {
		name, input, want string
	}{
		{"absent", `{}`, ""},
		{"empty", `{"language":""}`, ""},
		{"whitespace", `{"language":" \t "}`, ""},
		{"locale", `{"language":" zh-CN "}`, "zh-CN"},
		{"language name", `{"language":"简体中文"}`, "简体中文"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)
			write(t, filepath.Join(dir, "config.json"), c.input)
			cfg, err := loadConfig()
			if err != nil || cfg.Language != c.want {
				t.Fatalf("language %q, error %v; want %q", cfg.Language, err, c.want)
			}
		})
	}
}
