package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jesseduffield/lazygit/pkg/config"
)

const (
	defaultOllamaGeneratePath = "/api/generate"
	defaultTimeout            = 10 * time.Second
	maxCommitSummaryLength    = 72
)

var (
	ErrDisabled        = errors.New("ai commit message generation is disabled")
	ErrUnsupported     = errors.New("unsupported ai provider")
	ErrEmptyDiff       = errors.New("no staged diff to analyze")
	ErrEmptyResponse   = errors.New("ai provider returned an empty response")
	ErrInvalidEndpoint = errors.New("invalid ai endpoint")
)

type CommitMessageGenerator struct {
	client *http.Client
	cache  map[string]string
	mutex  sync.Mutex
}

func NewCommitMessageGenerator() *CommitMessageGenerator {
	return &CommitMessageGenerator{
		client: &http.Client{},
		cache:  map[string]string{},
	}
}

func (self *CommitMessageGenerator) Generate(ctx context.Context, cfg config.AIConfig, diff string) (string, error) {
	if !cfg.Enabled || !cfg.AutoGenerateCommitMessage {
		return "", ErrDisabled
	}

	if cfg.Provider != "ollama" {
		return "", fmt.Errorf("%w: %s", ErrUnsupported, cfg.Provider)
	}

	truncatedDiff := TruncateDiff(diff, cfg.MaxDiffLines)
	if strings.TrimSpace(truncatedDiff) == "" {
		return "", ErrEmptyDiff
	}

	cacheKey := self.cacheKey(cfg, truncatedDiff)
	if cachedMessage := self.getCached(cacheKey); cachedMessage != "" {
		return cachedMessage, nil
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	message, err := self.generateWithOllama(ctx, cfg, truncatedDiff)
	if err != nil {
		return "", err
	}

	message = normalizeCommitMessage(message)
	if message == "" {
		return "", ErrEmptyResponse
	}

	self.setCached(cacheKey, message)
	return message, nil
}

func (self *CommitMessageGenerator) generateWithOllama(ctx context.Context, cfg config.AIConfig, diff string) (string, error) {
	endpoint, err := ollamaGenerateURL(cfg.Endpoint)
	if err != nil {
		return "", err
	}

	body := ollamaGenerateRequest{
		Model:  cfg.Model,
		Prompt: buildCommitMessagePrompt(diff),
		Stream: false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := self.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var response ollamaGenerateResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", err
	}

	return response.Response, nil
}

func (self *CommitMessageGenerator) cacheKey(cfg config.AIConfig, diff string) string {
	hash := sha256.Sum256([]byte(cfg.Provider + "\x00" + cfg.Model + "\x00" + cfg.Endpoint + "\x00" + diff))
	return hex.EncodeToString(hash[:])
}

func (self *CommitMessageGenerator) getCached(key string) string {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	return self.cache[key]
}

func (self *CommitMessageGenerator) setCached(key, message string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.cache[key] = message
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

func ollamaGenerateURL(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", ErrInvalidEndpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalidEndpoint
	}
	if !isLocalEndpoint(parsed.Hostname()) {
		return "", fmt.Errorf("%w: endpoint must be local", ErrInvalidEndpoint)
	}

	path := strings.TrimRight(parsed.Path, "/")
	if path == "" || path == "/" {
		parsed.Path = defaultOllamaGeneratePath
	} else if path != defaultOllamaGeneratePath {
		parsed.Path = path + defaultOllamaGeneratePath
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func isLocalEndpoint(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func buildCommitMessagePrompt(diff string) string {
	return strings.TrimSpace(`You are an expert software engineer.
Analyze this git diff and generate a concise and precise conventional commit message.
Do not hallucinate functionality.
Keep the title under 72 characters.

Return only the commit message. Use this format:
<type>(optional scope): <title>

<body>

The body is required. Keep it to 1-3 concise lines that explain what changed and why.
Do not include bullets unless the diff clearly needs multiple points.

Git diff:
` + diff)
}

func normalizeCommitMessage(message string) string {
	message = strings.TrimSpace(message)
	message = strings.TrimPrefix(message, "```")
	message = strings.TrimSuffix(message, "```")
	message = strings.TrimSpace(message)

	lines := strings.Split(message, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}

	if len(lines) == 0 {
		return ""
	}

	lines[0] = trimCommitSummary(lines[0])
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func trimCommitSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	if len(summary) <= maxCommitSummaryLength {
		return summary
	}

	truncated := summary[:maxCommitSummaryLength]
	if index := strings.LastIndex(truncated, " "); index >= maxCommitSummaryLength/2 {
		truncated = truncated[:index]
	}

	return strings.TrimRight(truncated, " .,:;")
}

func TruncateDiff(diff string, maxLines int) string {
	if maxLines <= 0 {
		return diff
	}

	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}

	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + "\n\n[Diff truncated: showing first " + fmt.Sprint(maxLines) + " lines]"
}
