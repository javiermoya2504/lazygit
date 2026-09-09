package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/integrii/flaggy"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestModelCommands(t *testing.T) {
	failPull := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"small:latest","size":3000000000}]}`))
		case "/api/pull":
			if failPull {
				_, _ = w.Write([]byte(`{"error":"download failed"}`))
			} else {
				_, _ = w.Write([]byte(`{"status":"success"}`))
			}
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	t.Setenv("CONFIG_DIR", dir)
	t.Setenv("LG_CONFIG_FILE", "")
	path := filepath.Join(dir, config.ConfigFilename)
	original := []byte("ai:\n  endpoint: " + server.URL + "\n")
	assert.NoError(t, os.WriteFile(path, original, 0o600))
	var out bytes.Buffer
	assert.NoError(t, runModelCommand("list", "", &out))
	assert.Contains(t, out.String(), "small:latest")
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, original, data)
	assert.ErrorContains(t, runModelCommand("use", "missing", &out), "not installed")
	failPull = true
	assert.ErrorContains(t, runModelCommand("pull", "small", &out), "download failed")
	data, err = os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, original, data)
	failPull = false
	assert.NoError(t, runModelCommand("pull", "small", &out))
	saved, err := config.LoadModelSettings()
	assert.NoError(t, err)
	assert.Equal(t, "small:latest", saved.AI.Model)
	assert.True(t, saved.AI.Enabled)
	assert.NoError(t, runModelCommand("use", "small", &out))
	out.Reset()
	assert.NoError(t, runModelCommand("list", "", &out))
	assert.Contains(t, out.String(), "*")
	assert.Contains(t, out.String(), "AI enabled: true")
}

func TestModelAndPanelCLIParsing(t *testing.T) {
	original := os.Args
	defer func() { os.Args = original; flaggy.ResetParser() }()
	for _, tc := range []struct {
		args                       []string
		command, model, panel, dir string
	}{
		{[]string{"pull", "small:3b"}, "pull", "small:3b", "", ""},
		{[]string{"use", "small:3b", "--use-config-dir", "custom"}, "use", "small:3b", "", "custom"},
		{[]string{"--use-config-dir", "custom", "list"}, "list", "", "", "custom"},
		{[]string{"status"}, "", "", "status", ""},
		{[]string{"branch"}, "", "", "branch", ""},
		{[]string{"log"}, "", "", "log", ""},
		{[]string{"stash"}, "", "", "stash", ""},
		{[]string{}, "", "", "", ""},
	} {
		t.Run(tc.command+tc.panel+tc.dir, func(t *testing.T) {
			os.Args = append([]string{"lazygit-ai"}, tc.args...)
			flaggy.ResetParser()
			result := parseCliArgsAndEnvVars()
			assert.Equal(t, tc.command, result.ModelCommand)
			assert.Equal(t, tc.model, result.ModelName)
			assert.Equal(t, tc.panel, result.GitArg)
			assert.Equal(t, tc.dir, result.UseConfigDir)
		})
	}
}
