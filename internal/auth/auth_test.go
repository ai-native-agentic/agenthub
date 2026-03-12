package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agenthub/internal/db"
)

func newAuthTestDB(t *testing.T) *db.DB {
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

func TestMiddlewareValidBearerSetsAgent(t *testing.T) {
	database := newAuthTestDB(t)
	if err := database.CreateAgent("agent-1", "token-1"); err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	called := false
	handler := Middleware(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		agent := AgentFromContext(r.Context())
		if agent == nil {
			t.Fatal("expected agent in context, got nil")
		}
		if agent.ID != "agent-1" {
			t.Fatalf("agent.ID = %q, want %q", agent.ID, "agent-1")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected wrapped handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	database := newAuthTestDB(t)
	if err := database.CreateAgent("agent-1", "token-1"); err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	handler := Middleware(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wrapped handler should not be called for invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareRejectsMissingAuthorizationHeader(t *testing.T) {
	database := newAuthTestDB(t)

	handler := Middleware(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wrapped handler should not be called when auth header is missing")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminMiddlewareAllowsCorrectKey(t *testing.T) {
	handler := AdminMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAdminMiddlewareRejectsWrongKey(t *testing.T) {
	handler := AdminMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wrapped handler should not be called for wrong admin key")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
