package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func getBinaryPath(t *testing.T) string {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	path := "../../tmp/mcp-alertmanager"
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(absPath); err == nil {
		return absPath
	}

	t.Fatal("mcp-alertmanager binary not found. Run 'task build' first or use 'task test:e2e'")
	return ""
}

// newFakeAlertmanager creates a test HTTP server simulating Alertmanager v2 API.
func newFakeAlertmanager(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"labels":      map[string]string{"alertname": "TestAlert", "severity": "critical"},
				"annotations": map[string]string{"summary": "Test alert"},
				"status":      map[string]string{"state": "active"},
			},
		})
	})

	mux.HandleFunc("/api/v2/silences", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"silenceID": "test-silence-id"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":     "silence-1",
				"status": map[string]string{"state": "active"},
				"matchers": []map[string]any{
					{"name": "alertname", "value": "TestAlert", "isRegex": false, "isEqual": true},
				},
			},
		})
	})

	mux.HandleFunc("/api/v2/silence/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v2/silence/")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     id,
			"status": map[string]string{"state": "active"},
		})
	})

	return httptest.NewServer(mux)
}

func newTestClient(t *testing.T, alertmanagerURL string) *client.Client {
	t.Helper()

	binaryPath := getBinaryPath(t)
	c, err := client.NewStdioMCPClient(binaryPath, nil, "-mode", "stdio", "-url", alertmanagerURL)
	if err != nil {
		t.Fatalf("failed to create MCP client: %v", err)
	}

	t.Cleanup(func() {
		_ = c.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcp-alertmanager-e2e-test",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize MCP client: %v", err)
	}

	return c
}

func TestE2E_ListAlerts(t *testing.T) {
	srv := newFakeAlertmanager(t)
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_alerts",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	content := getTextContent(t, result)
	if !strings.Contains(content, "TestAlert") {
		t.Errorf("expected result to contain 'TestAlert', got: %s", content)
	}
}

func TestE2E_ListSilences(t *testing.T) {
	srv := newFakeAlertmanager(t)
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_silences",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	content := getTextContent(t, result)
	if !strings.Contains(content, "silence-1") {
		t.Errorf("expected result to contain 'silence-1', got: %s", content)
	}
}

func TestE2E_GetSilence(t *testing.T) {
	srv := newFakeAlertmanager(t)
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_silence",
			Arguments: map[string]any{
				"id": "test-123",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	content := getTextContent(t, result)
	if !strings.Contains(content, "test-123") {
		t.Errorf("expected result to contain 'test-123', got: %s", content)
	}
}

func TestE2E_CreateSilence(t *testing.T) {
	srv := newFakeAlertmanager(t)
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "create_silence",
			Arguments: map[string]any{
				"matchers": []any{"alertname=TestAlert"},
				"author":   "test-user",
				"comment":  "e2e test silence",
				"duration": "1h",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	content := getTextContent(t, result)
	if !strings.Contains(content, "test-silence-id") {
		t.Errorf("expected result to contain silence ID, got: %s", content)
	}
}

func TestE2E_DeleteSilence(t *testing.T) {
	srv := newFakeAlertmanager(t)
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "delete_silence",
			Arguments: map[string]any{
				"id": "silence-to-delete",
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	content := getTextContent(t, result)
	if !strings.Contains(content, "expired successfully") {
		t.Errorf("expected success message, got: %s", content)
	}
}

func TestE2E_RequiredParameterValidation(t *testing.T) {
	srv := newFakeAlertmanager(t)
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// get_silence without id
	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_silence",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Errorf("expected error for missing required parameter, got success")
	}
}

func getTextContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	for _, c := range result.Content {
		if textContent, ok := c.(mcp.TextContent); ok {
			return textContent.Text
		}
	}
	t.Fatalf("expected TextContent, got: %T", result.Content[0])
	return ""
}
