package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelPullStream(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"success", "{\"status\":\"pulling\",\"total\":100,\"completed\":50}\n{\"status\":\"success\"}\n", false},
		{"stream error", "{\"error\":\"model not found\"}\n", true},
		{"truncated", "{\"status\":\"pulling\"}\n", true},
		{"invalid", "not json", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/pull", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				var payload map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				assert.Equal(t, "small:3b", payload["model"])
				assert.Equal(t, true, payload["stream"])
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			var updates []ModelProgress
			err := (ModelClient{Endpoint: server.URL}).Pull(context.Background(), "small:3b", func(p ModelProgress) { updates = append(updates, p) })
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, updates, 2)
				assert.Equal(t, int64(50), updates[0].Completed)
			}
		})
	}
}

func TestModelListAndEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tags", r.URL.Path)
		_, _ = w.Write([]byte(`{"models":[{"name":"small:latest","size":1234}]}`))
	}))
	defer server.Close()
	models, err := (ModelClient{Endpoint: server.URL + "/api/generate"}).List(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []Model{{Name: "small:latest", Size: 1234}}, models)
	_, err = (ModelClient{Endpoint: "http://192.168.1.2:11434"}).List(context.Background())
	assert.ErrorContains(t, err, "local")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = (ModelClient{Endpoint: server.URL}).List(ctx)
	assert.Error(t, err)
}
