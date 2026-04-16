package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ntf-auction-backend/internal/app"
	"ntf-auction-backend/internal/config"
)

func setupEngine(t *testing.T) http.Handler {
	t.Helper()

	cfg := config.App{
		Server: config.ServerConfig{Port: 8080},
		Database: config.DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: "file::memory:?cache=shared",
		},
		JWT: config.JWTConfig{Secret: "test-secret", ExpireHours: 1},
		DefaultUser: config.DefaultUserConfig{
			Username: "admin",
			Password: "admin123",
		},
	}

	application, err := app.New(cfg)
	if err != nil {
		t.Fatalf("init app failed: %v", err)
	}
	return application.Engine
}

func TestLoginSuccess(t *testing.T) {
	engine := setupEngine(t)

	body, _ := json.Marshal(map[string]any{
		"username": "admin",
		"password": "admin123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	if int(resp["code"].(float64)) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]any)
	if data["token"] == "" {
		t.Fatalf("token should not be empty")
	}
}

func TestLogsNeedAuth(t *testing.T) {
	engine := setupEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListLogsWithTokenAndPagination(t *testing.T) {
	engine := setupEngine(t)

	// Build log records via health endpoint.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
	}

	token := loginAndGetToken(t, engine)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?page=1&page_size=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list logs response: %v", err)
	}

	data := resp["data"].(map[string]any)
	if int(data["page"].(float64)) != 1 {
		t.Fatalf("expected page 1, got %v", data["page"])
	}
	if int(data["page_size"].(float64)) != 2 {
		t.Fatalf("expected page_size 2, got %v", data["page_size"])
	}
	if len(data["items"].([]any)) > 2 {
		t.Fatalf("expected item size <= 2, got %d", len(data["items"].([]any)))
	}
}

func loginAndGetToken(t *testing.T, engine http.Handler) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"username": "admin",
		"password": "admin123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login status: %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login body: %v", err)
	}
	return resp["data"].(map[string]any)["token"].(string)
}
