package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Model struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type ModelProgress struct {
	Status    string `json:"status"`
	Total     int64  `json:"total"`
	Completed int64  `json:"completed"`
	Error     string `json:"error"`
}

// ModelClient uses the same loopback-only endpoint as commit generation.
type ModelClient struct{ Endpoint string }

func (c ModelClient) request(ctx context.Context, method, action string, body io.Reader) (*http.Response, error) {
	endpoint, err := ollamaGenerateURL(c.Endpoint)
	if err != nil {
		return nil, err
	}
	endpoint = strings.TrimSuffix(endpoint, defaultOllamaGeneratePath) + "/api/" + action
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach Ollama at %s; open Ollama or run 'ollama serve': %w", c.Endpoint, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("Ollama HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	return resp, nil
}

func (c ModelClient) List(ctx context.Context) ([]Model, error) {
	resp, err := c.request(ctx, http.MethodGet, "tags", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Models []Model `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Models, nil
}

func (c ModelClient) Pull(ctx context.Context, model string, progress func(ModelProgress)) error {
	payload, err := json.Marshal(struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}{model, true})
	if err != nil {
		return err
	}
	resp, err := c.request(ctx, http.MethodPost, "pull", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	for {
		var update ModelProgress
		if err := decoder.Decode(&update); err != nil {
			if err == io.EOF {
				return fmt.Errorf("Ollama download ended before success; model selection unchanged")
			}
			return err
		}
		if update.Error != "" {
			return fmt.Errorf("Ollama: %s", update.Error)
		}
		if progress != nil {
			progress(update)
		}
		if update.Status == "success" {
			return nil
		}
	}
}

func ModelName(name string) string {
	if !strings.Contains(name[strings.LastIndex(name, "/")+1:], ":") {
		return name + ":latest"
	}
	return name
}
