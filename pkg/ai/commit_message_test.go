package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
)

func TestGenerateCommitMessageWithOllama(t *testing.T) {
	var request ollamaGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("expected path /api/generate, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"response":"feat(auth): add token refresh\n\nExplain refresh token validation for API sessions."}`))
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

	if err != nil {
		t.Fatal(err)
	}
	expected := "feat(auth): add token refresh\n\nExplain refresh token validation for API sessions."
	if message != expected {
		t.Fatalf("expected generated message, got %q", message)
	}
	if request.Model != "qwen2.5-coder" {
		t.Fatalf("expected model qwen2.5-coder, got %s", request.Model)
	}
	if request.Stream {
		t.Fatal("expected stream to be false")
	}
	if !strings.Contains(request.Prompt, "The body is required.") {
		t.Fatalf("expected prompt to contain instruction, got %q", request.Prompt)
	}
}

func TestNormalizeCommitMessageTrimsLongSummary(t *testing.T) {
	message := normalizeCommitMessage("feat(api): add dataset preview and export endpoints for auditing ML datasets\n\nDescribe the new dataset audit endpoints.")
	lines := strings.Split(message, "\n")

	if len(lines[0]) > maxCommitSummaryLength {
		t.Fatalf("expected summary under %d chars, got %d: %q", maxCommitSummaryLength, len(lines[0]), lines[0])
	}
	assertEqual(t, "Describe the new dataset audit endpoints.", lines[2])
}

func TestTruncateDiff(t *testing.T) {
	diff := "one\ntwo\nthree"

	assertEqual(t, "one\ntwo\n\n[Diff truncated: showing first 2 lines]", TruncateDiff(diff, 2))
	assertEqual(t, diff, TruncateDiff(diff, 3))
	assertEqual(t, diff, TruncateDiff(diff, 0))
}

func TestOllamaGenerateURLAcceptsRootOrGenerateEndpoint(t *testing.T) {
	root, err := ollamaGenerateURL("http://localhost:11434")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "http://localhost:11434/api/generate", root)

	generateEndpoint, err := ollamaGenerateURL("http://localhost:11434/api/generate")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "http://localhost:11434/api/generate", generateEndpoint)
}

func TestOllamaGenerateURLRejectsRemoteEndpoint(t *testing.T) {
	_, err := ollamaGenerateURL("https://example.com")

	if !errors.Is(err, ErrInvalidEndpoint) {
		t.Fatalf("expected ErrInvalidEndpoint, got %v", err)
	}
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

	if !errors.Is(err, ErrEmptyDiff) {
		t.Fatalf("expected ErrEmptyDiff, got %v", err)
	}
}

func assertEqual(t *testing.T, expected, actual string) {
	t.Helper()

	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
