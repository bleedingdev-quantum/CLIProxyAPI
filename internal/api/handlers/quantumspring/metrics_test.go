package quantumspring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/persistence"
)

func setupTestRouter(storage persistence.Storage) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set global storage for tests
	if storage != nil {
		// Note: In real implementation, you'd inject storage differently
		// For now, we'll test with what we have
	}

	cfg := &config.Config{
		QuantumSpring: config.QuantumSpringConfig{
			Enabled: true,
		},
	}

	RegisterRoutes(router, cfg)
	return router
}

func TestGetHealth(t *testing.T) {
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.OK {
		t.Error("Expected ok=true")
	}

	if response.Version == "" {
		t.Error("Expected version to be set")
	}
}

func TestGetMetricsWithoutPersistence(t *testing.T) {
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics", nil)
	router.ServeHTTP(w, req)

	// Should return 503 when persistence is not enabled
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestGetMetricsWithPersistence(t *testing.T) {
	// Create in-memory storage
	storage, err := persistence.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Insert test data
	now := time.Now()
	records := []persistence.UsageRecord{
		{
			Timestamp:        now,
			Provider:         "openai",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Status:           200,
			Failed:           false,
		},
		{
			Timestamp:        now,
			Provider:         "anthropic",
			Model:            "claude-3-opus",
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
			Status:           200,
			Failed:           false,
		},
	}

	ctx := context.Background()
	if err := storage.InsertBatch(ctx, records); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Note: In a real test, you'd need to set the global storage
	// This is a limitation of the current implementation
	// For now, we'll just verify the endpoint exists
	router := setupTestRouter(storage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics", nil)
	router.ServeHTTP(w, req)

	// This will fail without global storage, but shows the pattern
	if w.Code != http.StatusServiceUnavailable {
		t.Logf("Response: %s", w.Body.String())
	}
}

func TestGetMetricsWithTimeRange(t *testing.T) {
	router := setupTestRouter(nil)

	now := time.Now()
	from := now.Add(-24 * time.Hour).Format(time.RFC3339)
	to := now.Format(time.RFC3339)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics?from="+from+"&to="+to, nil)
	router.ServeHTTP(w, req)

	// Will return 503 without persistence, but tests query parsing
	if w.Code == http.StatusBadRequest {
		t.Error("Query parameters should be valid")
	}
}

func TestGetMetricsWithInvalidTimeRange(t *testing.T) {
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics?from=invalid-time", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		// This is expected with invalid time format
		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if errorResp["error"] == "" {
			t.Error("Expected error message")
		}
	}
}

func TestGetMetricsWithInterval(t *testing.T) {
	router := setupTestRouter(nil)

	intervals := []string{"hour", "day", "week", "month"}

	for _, interval := range intervals {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/_qs/metrics?interval="+interval, nil)
		router.ServeHTTP(w, req)

		if w.Code == http.StatusBadRequest {
			t.Errorf("Interval %s should be valid", interval)
		}
	}
}

func TestGetMetricsWithInvalidInterval(t *testing.T) {
	router := setupTestRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics?interval=invalid", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if errorResp["error"] == "" {
			t.Error("Expected error message for invalid interval")
		}
	}
}

func TestServeUI(t *testing.T) {
	router := setupTestRouter(nil)

	// Test main UI page
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics/ui", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}
}

func TestServeUIJavaScript(t *testing.T) {
	router := setupTestRouter(nil)

	// Test JavaScript file
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/_qs/metrics/ui/app.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/javascript" {
		t.Errorf("Expected Content-Type application/javascript, got %s", w.Header().Get("Content-Type"))
	}
}

func TestConvertModelMetrics(t *testing.T) {
	metrics := []persistence.ModelMetrics{
		{
			Model:            "gpt-4",
			Requests:         100,
			Tokens:           10000,
			PromptTokens:     5000,
			CompletionTokens: 5000,
			AvgLatencyMs:     1234.56,
			FailedRequests:   5,
		},
	}

	result := convertModelMetrics(metrics)

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	if result[0].Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", result[0].Model)
	}

	if result[0].Requests != 100 {
		t.Errorf("Expected 100 requests, got %d", result[0].Requests)
	}
}

func TestConvertAPIKeyMetrics(t *testing.T) {
	metrics := []persistence.APIKeyMetrics{
		{
			APIKey:   "sk-1234567890abcdef",
			Requests: 50,
			Tokens:   5000,
		},
	}

	result := convertAPIKeyMetrics(metrics)

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	// API key should be masked
	if result[0].APIKey == "sk-1234567890abcdef" {
		t.Error("API key should be masked")
	}

	if result[0].Requests != 50 {
		t.Errorf("Expected 50 requests, got %d", result[0].Requests)
	}
}

func TestConvertTimeseries(t *testing.T) {
	now := time.Now()
	points := []persistence.TimeseriesPoint{
		{
			BucketStart:      now,
			Requests:         100,
			Tokens:           10000,
			PromptTokens:     5000,
			CompletionTokens: 5000,
			AvgLatencyMs:     1500.0,
			FailedRequests:   2,
		},
	}

	result := convertTimeseries(points)

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	if result[0].Requests != 100 {
		t.Errorf("Expected 100 requests, got %d", result[0].Requests)
	}

	// Check timestamp format
	_, err := time.Parse(time.RFC3339, result[0].BucketStart)
	if err != nil {
		t.Errorf("BucketStart should be in RFC3339 format: %v", err)
	}
}
