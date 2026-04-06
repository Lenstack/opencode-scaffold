package hub

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_New(t *testing.T) {
	client := NewClient("https://hub.example.com", "test-api-key")

	if client.server != "https://hub.example.com" {
		t.Errorf("expected server 'https://hub.example.com', got %q", client.server)
	}
	if client.apiKey != "test-api-key" {
		t.Errorf("expected apiKey 'test-api-key', got %q", client.apiKey)
	}
}

func TestClient_Health(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Errorf("expected /api/health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	result, err := client.Health()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", result["status"])
	}
}

func TestClient_PushKnowledge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/knowledge/push" {
			t.Errorf("expected /api/knowledge/push, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	push := KnowledgePush{
		ProjectID: "project-1",
		Workspace: "/workspace",
		Stack:     "go",
	}

	err := client.PushKnowledge(push)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestClient_PullKnowledge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("stack") != "go" {
			t.Errorf("expected stack 'go', got %s", r.URL.Query().Get("stack"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"stack":"go","semantic":[],"heuristics":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	pull, err := client.PullKnowledge("go", "/workspace")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pull.Stack != "go" {
		t.Errorf("expected stack 'go', got %q", pull.Stack)
	}
}

func TestClient_GetGlobalHeuristics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"heuristics":[{"rule":"test","confidence":0.8}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	query := GlobalHeuristicQuery{
		Stack:     "go",
		Workspace: "/workspace",
		MinConf:   0.5,
		Limit:     10,
	}

	heuristics, err := client.GetGlobalHeuristics(query)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Got %d heuristics", len(heuristics))
}

func TestClient_PushSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions/sync" {
			t.Errorf("expected /api/sessions/sync, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	outcome := SessionOutcome{
		ID:        "outcome-1",
		SessionID: "session-1",
		Outcome:   "success",
	}

	err := client.PushSession(outcome)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestClient_GetWorkspaceKnowledge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"knowledge":[{"id":"1","type":"lesson","title":"Test"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	entries, err := client.GetWorkspaceKnowledge("/workspace")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Got %d knowledge entries", len(entries))
}

func TestClient_postJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	err := client.postJSON("/test", map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestClient_setAuth(t *testing.T) {
	client := NewClient("https://example.com", "test-api-key")

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	client.setAuth(req)

	if req.Header.Get("Authorization") != "Bearer test-api-key" {
		t.Errorf("expected 'Bearer test-api-key', got %s", req.Header.Get("Authorization"))
	}
}

func TestClient_setAuthNoKey(t *testing.T) {
	client := NewClient("https://example.com", "")

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	client.setAuth(req)

	if req.Header.Get("Authorization") != "" {
		t.Errorf("expected empty Authorization, got %s", req.Header.Get("Authorization"))
	}
}

func TestClient_parseError(t *testing.T) {
	client := NewClient("https://example.com", "")

	// Test with JSON error
	resp := httptest.NewRecorder()
	resp.Code = 400
	resp.Body.Write([]byte(`{"error":"bad request"}`))

	err := client.parseError(resp.Result())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_parseErrorPlain(t *testing.T) {
	client := NewClient("https://example.com", "")

	// Test with plain text error
	resp := httptest.NewRecorder()
	resp.Code = 500
	resp.Body.Write([]byte("internal server error"))

	err := client.parseError(resp.Result())
	if err == nil {
		t.Fatal("expected error")
	}
}
