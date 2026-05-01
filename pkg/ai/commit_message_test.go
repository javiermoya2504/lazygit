package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCommitMessageWithOllama(t *testing.T) {
	var request ollamaGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/generate", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		_, _ = w.Write([]byte(`{"response":"feat(auth): add token refresh"}`))
	}))
	defer server.Close()

	generator := NewCommitMessageGenerator()
	message, err := generator.Generate(context.Background(), config.AIConfig{
		Enabled:                   true,
		AutoGenerateCommitMessage: true,
		Provider:                  "ollama",
		Model:                     "qwen2.5-coder",
		Endpoint:                  server.URL,
		MaxDiffLines:              300,
		TimeoutSeconds:            10,
	}, "diff --git a/auth.go b/auth.go\n+refresh()")

	require.NoError(t, err)
	assert.Equal(t, "feat(auth): add token refresh", message)
	assert.Equal(t, "qwen2.5-coder", request.Model)
	assert.False(t, request.Stream)
	assert.Contains(t, request.Prompt, "Keep title under 72 chars.")
}

func TestTruncateDiff(t *testing.T) {
	diff := "one\ntwo\nthree"

	assert.Equal(t, "one\ntwo\n\n[Diff truncated: showing first 2 lines]", TruncateDiff(diff, 2))
	assert.Equal(t, diff, TruncateDiff(diff, 3))
	assert.Equal(t, diff, TruncateDiff(diff, 0))
}

func TestOllamaGenerateURLAcceptsRootOrGenerateEndpoint(t *testing.T) {
	root, err := ollamaGenerateURL("http://localhost:11434")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:11434/api/generate", root)

	generateEndpoint, err := ollamaGenerateURL("http://localhost:11434/api/generate")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:11434/api/generate", generateEndpoint)
}

func TestOllamaGenerateURLRejectsRemoteEndpoint(t *testing.T) {
	_, err := ollamaGenerateURL("https://example.com")

	assert.ErrorIs(t, err, ErrInvalidEndpoint)
}

func TestGenerateRejectsEmptyDiff(t *testing.T) {
	generator := NewCommitMessageGenerator()
	_, err := generator.Generate(context.Background(), config.AIConfig{
		Enabled:                   true,
		AutoGenerateCommitMessage: true,
		Provider:                  "ollama",
		Model:                     "qwen2.5-coder",
		Endpoint:                  "http://localhost:11434",
		MaxDiffLines:              300,
		TimeoutSeconds:            10,
	}, "   ")

	assert.ErrorIs(t, err, ErrEmptyDiff)
}
