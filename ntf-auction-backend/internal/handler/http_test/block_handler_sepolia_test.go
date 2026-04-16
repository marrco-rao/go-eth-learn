package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"ntf-auction-backend/internal/app"
	"ntf-auction-backend/internal/config"
)

func setupEngineWithSepolia(t *testing.T) http.Handler {
	t.Helper()

	rpcURL := sepoliaRPCURL(t)
	t.Setenv("ETH_RPC_URL", rpcURL)

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

func sepoliaRPCURL(t *testing.T) string {
	t.Helper()

	for _, key := range []string{"ETH_RPC_URL_SEPOLIA", "SEPOLIA_RPC_URL", "ETH_RPC_URL"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}

	t.Skip("skip sepolia integration test: set ETH_RPC_URL_SEPOLIA or SEPOLIA_RPC_URL")
	return ""
}

func TestBlockLatestSepolia(t *testing.T) {
	engine := setupEngineWithSepolia(t)
	token := loginAndGetToken(t, engine)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/blocks/latest", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode latest block response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("response data should be object")
	}

	if data["number"] == nil {
		t.Fatalf("latest block number should not be nil")
	}
	if data["hash"] == "" {
		t.Fatalf("latest block hash should not be empty")
	}
}

func TestBlockByNumberSepolia(t *testing.T) {
	engine := setupEngineWithSepolia(t)
	token := loginAndGetToken(t, engine)

	latestReq := httptest.NewRequest(http.MethodGet, "/api/v1/blocks/latest", nil)
	latestReq.Header.Set("Authorization", "Bearer "+token)
	latestW := httptest.NewRecorder()
	engine.ServeHTTP(latestW, latestReq)
	if latestW.Code != http.StatusOK {
		t.Fatalf("latest status expected 200, got %d, body=%s", latestW.Code, latestW.Body.String())
	}

	var latestResp map[string]any
	if err := json.Unmarshal(latestW.Body.Bytes(), &latestResp); err != nil {
		t.Fatalf("decode latest response: %v", err)
	}
	latestData := latestResp["data"].(map[string]any)
	latestNumber := uint64(latestData["number"].(float64))

	blockReq := httptest.NewRequest(http.MethodGet, "/api/v1/blocks/"+strconv.FormatUint(latestNumber, 10), nil)
	blockReq.Header.Set("Authorization", "Bearer "+token)
	blockW := httptest.NewRecorder()
	engine.ServeHTTP(blockW, blockReq)

	if blockW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", blockW.Code, blockW.Body.String())
	}

	var blockResp map[string]any
	if err := json.Unmarshal(blockW.Body.Bytes(), &blockResp); err != nil {
		t.Fatalf("decode block by number response: %v", err)
	}

	data := blockResp["data"].(map[string]any)
	gotNumber := uint64(data["number"].(float64))
	if gotNumber != latestNumber {
		t.Fatalf("expected block number %d, got %d", latestNumber, gotNumber)
	}
}

func TestBlocksRangeSepolia(t *testing.T) {
	engine := setupEngineWithSepolia(t)
	token := loginAndGetToken(t, engine)

	latestReq := httptest.NewRequest(http.MethodGet, "/api/v1/blocks/latest", nil)
	latestReq.Header.Set("Authorization", "Bearer "+token)
	latestW := httptest.NewRecorder()
	engine.ServeHTTP(latestW, latestReq)
	if latestW.Code != http.StatusOK {
		t.Fatalf("latest status expected 200, got %d, body=%s", latestW.Code, latestW.Body.String())
	}

	var latestResp map[string]any
	if err := json.Unmarshal(latestW.Body.Bytes(), &latestResp); err != nil {
		t.Fatalf("decode latest response: %v", err)
	}
	latestData := latestResp["data"].(map[string]any)
	end := uint64(latestData["number"].(float64))
	if end == 0 {
		t.Fatalf("latest block number should be greater than 0")
	}
	start := end - 1

	rangeReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/blocks?start="+strconv.FormatUint(start, 10)+"&end="+strconv.FormatUint(end, 10)+"&rate_limit=10",
		nil,
	)
	rangeReq.Header.Set("Authorization", "Bearer "+token)
	rangeW := httptest.NewRecorder()
	engine.ServeHTTP(rangeW, rangeReq)

	if rangeW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rangeW.Code, rangeW.Body.String())
	}

	var rangeResp map[string]any
	if err := json.Unmarshal(rangeW.Body.Bytes(), &rangeResp); err != nil {
		t.Fatalf("decode blocks range response: %v", err)
	}

	data := rangeResp["data"].(map[string]any)
	items := data["data"].([]any)
	if len(items) == 0 {
		t.Fatalf("range result should not be empty")
	}

	success := uint64(data["success"].(float64))
	if success == 0 {
		t.Fatalf("success should be > 0")
	}
}
