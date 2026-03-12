package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agenthub/internal/db"
)

func newServerTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	if err := database.Migrate(); err != nil {
		t.Fatalf("database.Migrate() error = %v", err)
	}

	return database
}

func newTestServer(t *testing.T) (*Server, *db.DB) {
	t.Helper()

	database := newServerTestDB(t)
	srv := New(database, nil, "admin-secret", Config{
		MaxBundleSize:    1024 * 1024,
		MaxPushesPerHour: 10,
		MaxPostsPerHour:  10,
		ListenAddr:       ":0",
	})

	return srv, database
}

func TestHealthEndpointReturnsOK(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %q, want status ok json", rr.Body.String())
	}
}

func TestAdminCreateAgentRequiresValidAdminKey(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/agents", strings.NewReader(`{"id":"agent-1"}`))
	rr := httptest.NewRecorder()

	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminCreateAgentCreatesAgentWithAPIKey(t *testing.T) {
	srv, database := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/agents", strings.NewReader(`{"id":"agent-1"}`))
	req.Header.Set("Authorization", "Bearer admin-secret")
	rr := httptest.NewRecorder()

	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%q", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["id"] != "agent-1" {
		t.Fatalf("payload id = %q, want %q", payload["id"], "agent-1")
	}
	if payload["api_key"] == "" {
		t.Fatal("expected generated api_key, got empty")
	}

	agent, err := database.GetAgentByID("agent-1")
	if err != nil {
		t.Fatalf("GetAgentByID() error = %v", err)
	}
	if agent == nil {
		t.Fatal("expected created agent to exist in DB")
	}
}

func TestCreateAndListChannelsWithBearerAuth(t *testing.T) {
	srv, database := newTestServer(t)
	if err := database.CreateAgent("agent-1", "token-1"); err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/channels", strings.NewReader(`{"name":"general","description":"General"}`))
	createReq.Header.Set("Authorization", "Bearer token-1")
	createRR := httptest.NewRecorder()

	srv.mux.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body=%q", createRR.Code, http.StatusCreated, createRR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	listReq.Header.Set("Authorization", "Bearer token-1")
	listRR := httptest.NewRecorder()

	srv.mux.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%q", listRR.Code, http.StatusOK, listRR.Body.String())
	}
	if !strings.Contains(listRR.Body.String(), `"name":"general"`) {
		t.Fatalf("list body = %q, expected created channel", listRR.Body.String())
	}
}
