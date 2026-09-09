package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSelectModelPreservesUserConfig(t *testing.T) {
	t.Setenv("CONFIG_DIR", t.TempDir())
	t.Setenv("LG_CONFIG_FILE", "")
	path := filepath.Join(ConfigDir(), ConfigFilename)
	original := "# user preferences\ngui:\n  language: es\ncustomCommands:\n  - key: X\n    command: echo hello\nai:\n  model: old # keep this comment\n  endpoint: http://127.0.0.1:11434\n  timeoutSeconds: 120\n  maxDiffLines: 42\n"
	assert.NoError(t, os.WriteFile(path, []byte(original), 0o600))
	settings, err := LoadModelSettings()
	assert.NoError(t, err)
	assert.NoError(t, settings.SelectModel("qwen2.5-coder:3b"))
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "# user preferences")
	assert.Contains(t, string(data), "# keep this comment")
	var before, after map[string]any
	assert.NoError(t, yaml.Unmarshal([]byte(original), &before))
	assert.NoError(t, yaml.Unmarshal(data, &after))
	assert.Equal(t, before["gui"], after["gui"])
	assert.Equal(t, before["customCommands"], after["customCommands"])
	saved, err := LoadModelSettings()
	assert.NoError(t, err)
	assert.Equal(t, "qwen2.5-coder:3b", saved.AI.Model)
	assert.True(t, saved.AI.Enabled)
	assert.True(t, saved.AI.AutoGenerateCommitMessage)
	assert.Equal(t, 120, saved.AI.TimeoutSeconds)
	assert.Equal(t, 42, saved.AI.MaxDiffLines)
	info, err := os.Stat(path)
	assert.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestSelectModelCreatesConfigAndDetectsConcurrentEdit(t *testing.T) {
	t.Setenv("CONFIG_DIR", filepath.Join(t.TempDir(), "new"))
	t.Setenv("LG_CONFIG_FILE", "")
	settings, err := LoadModelSettings()
	assert.NoError(t, err)
	assert.NoError(t, settings.SelectModel("small:latest"))
	settings, err = LoadModelSettings()
	assert.NoError(t, err)
	assert.Equal(t, 60, settings.AI.TimeoutSeconds)
	assert.NoError(t, os.WriteFile(settings.Path, []byte("# changed\n"), 0o600))
	assert.ErrorContains(t, settings.SelectModel("other:latest"), "config changed")
	data, err := os.ReadFile(settings.Path)
	assert.NoError(t, err)
	assert.Equal(t, "# changed\n", string(data))
}

func TestModelSettingsLayeredConfig(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.yml")
	last := filepath.Join(dir, "last.yml")
	assert.NoError(t, os.WriteFile(first, []byte("ai:\n  endpoint: http://127.0.0.1:12345\n  timeoutSeconds: 90\n"), 0o600))
	assert.NoError(t, os.WriteFile(last, []byte("gui:\n  language: es\n"), 0o600))
	t.Setenv("LG_CONFIG_FILE", first+","+last)
	settings, err := LoadModelSettings()
	assert.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:12345", settings.AI.Endpoint)
	assert.NoError(t, settings.SelectModel("test:latest"))
	data, err := os.ReadFile(first)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(data), "model:"))
	settings, err = LoadModelSettings()
	assert.NoError(t, err)
	assert.Equal(t, "test:latest", settings.AI.Model)
	assert.Equal(t, 90, settings.AI.TimeoutSeconds)
}

func TestModelSettingsRejectsInvalidConfig(t *testing.T) {
	for _, content := range []string{"ai: [invalid]", "gui: [", "[]", "gui: {}\n---\nai: {}", "ai: {}\nai: {}"} {
		t.Run(content, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			t.Setenv("LG_CONFIG_FILE", path)
			assert.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			_, err := LoadModelSettings()
			assert.Error(t, err)
			data, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Equal(t, content, string(data))
		})
	}
}
