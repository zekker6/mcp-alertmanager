package alertmanager_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustNewClient(t *testing.T, baseURL string, opts ...ClientOption) *AlertmanagerClient {
	t.Helper()
	c, err := NewClient(baseURL, opts...)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return c
}

func TestNewClient(t *testing.T) {
	c := mustNewClient(t, "http://localhost:9093")
	if c.baseURL != "http://localhost:9093" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:9093")
	}
}

func TestNewClientWithBasicAuth(t *testing.T) {
	c := mustNewClient(t, "http://localhost:9093", WithBasicAuth("user", "pass"))
	if c.options.username != "user" || c.options.password != "pass" {
		t.Errorf("basic auth not set correctly")
	}
}

func TestNewClientWithHeaders(t *testing.T) {
	headers := map[string]string{"X-Custom": "value", "Authorization": "Bearer tok"}
	c := mustNewClient(t, "http://localhost:9093", WithHeaders(headers))
	if c.options.headers["X-Custom"] != "value" {
		t.Errorf("custom header not set")
	}
	if c.options.headers["Authorization"] != "Bearer tok" {
		t.Errorf("auth header not set")
	}
}

func TestNewClientInvalidCAFile(t *testing.T) {
	_, err := NewClient("http://localhost:9093", WithTLSCA("/nonexistent/ca.pem"))
	if err == nil {
		t.Fatal("expected error for nonexistent CA file")
	}
}

func TestGetAlerts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/alerts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"labels": map[string]string{"alertname": "TestAlert"}},
		})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	alerts, err := c.GetAlerts(nil)
	if err != nil {
		t.Fatalf("GetAlerts failed: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
}

func TestGetAlertsWithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("active") != "true" {
			t.Errorf("expected active=true, got %q", q.Get("active"))
		}
		if q.Get("receiver") != "slack" {
			t.Errorf("expected receiver=slack, got %q", q.Get("receiver"))
		}
		filters := q["filter"]
		if len(filters) != 1 || filters[0] != "severity=critical" {
			t.Errorf("expected filter=[severity=critical], got %v", filters)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	active := true
	_, err := c.GetAlerts(&AlertFilter{
		Filter:   []string{"severity=critical"},
		Active:   &active,
		Receiver: "slack",
	})
	if err != nil {
		t.Fatalf("GetAlerts with filters failed: %v", err)
	}
}

func TestGetAlertsBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("expected basic auth admin:secret, got %s:%s (ok=%v)", user, pass, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL, WithBasicAuth("admin", "secret"))
	_, err := c.GetAlerts(nil)
	if err != nil {
		t.Fatalf("GetAlerts with basic auth failed: %v", err)
	}
}

func TestGetAlertsCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Scope-OrgID") != "tenant1" {
			t.Errorf("expected X-Scope-OrgID=tenant1, got %q", r.Header.Get("X-Scope-OrgID"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL, WithHeaders(map[string]string{"X-Scope-OrgID": "tenant1"}))
	_, err := c.GetAlerts(nil)
	if err != nil {
		t.Fatalf("GetAlerts with custom headers failed: %v", err)
	}
}

func TestGetSilences(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/silences" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "silence-1", "status": map[string]string{"state": "active"}},
			{"id": "silence-2", "status": map[string]string{"state": "expired"}},
		})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	silences, err := c.GetSilences(nil)
	if err != nil {
		t.Fatalf("GetSilences failed: %v", err)
	}
	if len(silences) != 2 {
		t.Fatalf("expected 2 silences, got %d", len(silences))
	}
}

func TestGetSilence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/silence/abc-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "abc-123"})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	silence, err := c.GetSilence("abc-123")
	if err != nil {
		t.Fatalf("GetSilence failed: %v", err)
	}
	if silence["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %v", silence["id"])
	}
}

func TestCreateSilence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/silences" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"silenceID": "new-silence-id"})
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	id, err := c.CreateSilence(map[string]any{
		"matchers":  []map[string]any{{"name": "env", "value": "prod", "isRegex": false, "isEqual": true}},
		"createdBy": "test",
		"comment":   "test silence",
		"startsAt":  "2026-01-01T00:00:00Z",
		"endsAt":    "2026-01-02T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}
	if id != "new-silence-id" {
		t.Errorf("expected id=new-silence-id, got %q", id)
	}
}

func TestDeleteSilence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/silence/del-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	err := c.DeleteSilence("del-123")
	if err != nil {
		t.Fatalf("DeleteSilence failed: %v", err)
	}
}

func TestHTTPErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("silence not found"))
	}))
	defer srv.Close()

	c := mustNewClient(t, srv.URL)
	_, err := c.GetSilence("nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
