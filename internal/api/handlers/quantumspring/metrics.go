// Package quantumspring provides QuantumSpring-specific API handlers.
// It includes metrics collection, visualization, and usage tracking endpoints.
package quantumspring

import (
	"embed"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/persistence"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
)

//go:embed web/quantumspring/*
var webAssets embed.FS

// MetricsResponse represents the response format for the metrics endpoint.
type MetricsResponse struct {
	Totals       TotalsResponse         `json:"totals"`
	ByModel      []ModelMetricsResponse `json:"by_model"`
	ByAPIKey     []APIKeyResponse       `json:"by_api_key,omitempty"`
	ByProvider   []ProviderResponse     `json:"by_provider,omitempty"`
	Timeseries   []TimeseriesResponse   `json:"timeseries"`
	QueryPeriod  QueryPeriodResponse    `json:"query_period"`
}

// TotalsResponse contains overall aggregated metrics.
type TotalsResponse struct {
	Requests         int64   `json:"requests"`
	Tokens           int64   `json:"tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	ReasoningTokens  int64   `json:"reasoning_tokens,omitempty"`
	CachedTokens     int64   `json:"cached_tokens,omitempty"`
	FailedRequests   int64   `json:"failed_requests"`
	SuccessRate      float64 `json:"success_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
}

// ModelMetricsResponse contains metrics for a specific model.
type ModelMetricsResponse struct {
	Model            string  `json:"model"`
	Requests         int64   `json:"requests"`
	Tokens           int64   `json:"tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	FailedRequests   int64   `json:"failed_requests"`
}

// APIKeyResponse contains metrics for a specific API key.
type APIKeyResponse struct {
	APIKey   string `json:"api_key"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// ProviderResponse contains metrics for a specific provider.
type ProviderResponse struct {
	Provider     string  `json:"provider"`
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// TimeseriesResponse represents a single timeseries point.
type TimeseriesResponse struct {
	BucketStart      string  `json:"bucket_start"`
	Requests         int64   `json:"requests"`
	Tokens           int64   `json:"tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	FailedRequests   int64   `json:"failed_requests"`
}

// QueryPeriodResponse shows the actual period queried.
type QueryPeriodResponse struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	OK                    bool   `json:"ok"`
	Version               string `json:"version"`
	Persistence           string `json:"persistence"`
	StatisticsEnabled     bool   `json:"statistics_enabled"`
	PersistenceEnabled    bool   `json:"persistence_enabled"`
	TotalRecordsPersisted int64  `json:"total_records_persisted,omitempty"`
}

// GetHealth handles GET /_qs/health requests.
func GetHealth(c *gin.Context) {
	storage := persistence.GetStorage()

	response := HealthResponse{
		OK:                 true,
		Version:            "1.0.0",
		StatisticsEnabled:  usage.StatisticsEnabled(),
		PersistenceEnabled: storage != nil,
	}

	if storage != nil {
		response.Persistence = "sqlite"
		count, err := storage.GetRecordCount(c.Request.Context())
		if err == nil {
			response.TotalRecordsPersisted = count
		}
	} else {
		response.Persistence = "disabled"
	}

	c.JSON(http.StatusOK, response)
}

// GetMetrics handles GET /_qs/metrics requests.
// Query parameters:
//   - from: ISO 8601 timestamp (default: 24h ago)
//   - to: ISO 8601 timestamp (default: now)
//   - model: filter by model name
//   - interval: timeseries bucket size (hour/day/week/month, default: hour)
func GetMetrics(c *gin.Context) {
	storage := persistence.GetStorage()
	if storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "persistence is not enabled",
		})
		return
	}

	// Parse query parameters
	now := time.Now()
	defaultFrom := now.Add(-24 * time.Hour)

	from, err := parseTime(c.Query("from"), defaultFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid 'from' parameter: " + err.Error(),
		})
		return
	}

	to, err := parseTime(c.Query("to"), now)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid 'to' parameter: " + err.Error(),
		})
		return
	}

	interval := c.DefaultQuery("interval", "hour")
	if interval != "hour" && interval != "day" && interval != "week" && interval != "month" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid interval: must be hour, day, week, or month",
		})
		return
	}

	ctx := c.Request.Context()

	// Get totals
	totals, err := storage.GetTotals(ctx, from, to)
	if err != nil {
		log.WithError(err).Error("Failed to get totals")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve metrics",
		})
		return
	}

	// Calculate success rate
	successRate := 0.0
	if totals.Requests > 0 {
		successRate = float64(totals.SuccessRequests) / float64(totals.Requests) * 100.0
	}

	// Get by model
	byModel, err := storage.GetByModel(ctx, from, to)
	if err != nil {
		log.WithError(err).Error("Failed to get by model")
		byModel = []persistence.ModelMetrics{}
	}

	// Get timeseries
	timeseries, err := storage.GetTimeseries(ctx, from, to, interval)
	if err != nil {
		log.WithError(err).Error("Failed to get timeseries")
		timeseries = []persistence.TimeseriesPoint{}
	}

	// Get aggregate data for by_api_key and by_provider
	aggregateQuery := persistence.AggregateQuery{
		From:     from,
		To:       to,
		Interval: interval,
		GroupBy:  []string{"model", "api_key", "provider"},
	}

	aggregate, err := storage.Aggregate(ctx, aggregateQuery)
	if err != nil {
		log.WithError(err).Error("Failed to get aggregates")
		aggregate = &persistence.AggregateResult{}
	}

	// Build response
	response := MetricsResponse{
		Totals: TotalsResponse{
			Requests:         totals.Requests,
			Tokens:           totals.Tokens,
			PromptTokens:     totals.PromptTokens,
			CompletionTokens: totals.CompletionTokens,
			ReasoningTokens:  totals.ReasoningTokens,
			CachedTokens:     totals.CachedTokens,
			FailedRequests:   totals.FailedRequests,
			SuccessRate:      successRate,
			AvgLatencyMs:     totals.AvgLatencyMs,
		},
		ByModel:     convertModelMetrics(byModel),
		ByAPIKey:    convertAPIKeyMetrics(aggregate.ByAPIKey),
		ByProvider:  convertProviderMetrics(aggregate.ByProvider),
		Timeseries:  convertTimeseries(timeseries),
		QueryPeriod: QueryPeriodResponse{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		},
	}

	c.JSON(http.StatusOK, response)
}

// parseTime parses a time string or returns the default value.
func parseTime(s string, defaultValue time.Time) (time.Time, error) {
	if s == "" {
		return defaultValue, nil
	}
	return time.Parse(time.RFC3339, s)
}

// convertModelMetrics converts persistence model metrics to response format.
func convertModelMetrics(metrics []persistence.ModelMetrics) []ModelMetricsResponse {
	result := make([]ModelMetricsResponse, len(metrics))
	for i, m := range metrics {
		result[i] = ModelMetricsResponse{
			Model:            m.Model,
			Requests:         m.Requests,
			Tokens:           m.Tokens,
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
			AvgLatencyMs:     m.AvgLatencyMs,
			FailedRequests:   m.FailedRequests,
		}
	}
	return result
}

// convertAPIKeyMetrics converts persistence API key metrics to response format.
// Masks API keys for security.
func convertAPIKeyMetrics(metrics []persistence.APIKeyMetrics) []APIKeyResponse {
	result := make([]APIKeyResponse, len(metrics))
	for i, m := range metrics {
		result[i] = APIKeyResponse{
			APIKey:   util.HideAPIKey(m.APIKey),
			Requests: m.Requests,
			Tokens:   m.Tokens,
		}
	}
	return result
}

// convertProviderMetrics converts persistence provider metrics to response format.
func convertProviderMetrics(metrics []persistence.ProviderMetrics) []ProviderResponse {
	result := make([]ProviderResponse, len(metrics))
	for i, m := range metrics {
		result[i] = ProviderResponse{
			Provider:     m.Provider,
			Requests:     m.Requests,
			Tokens:       m.Tokens,
			AvgLatencyMs: m.AvgLatencyMs,
		}
	}
	return result
}

// convertTimeseries converts persistence timeseries to response format.
func convertTimeseries(points []persistence.TimeseriesPoint) []TimeseriesResponse {
	result := make([]TimeseriesResponse, len(points))
	for i, p := range points {
		result[i] = TimeseriesResponse{
			BucketStart:      p.BucketStart.Format(time.RFC3339),
			Requests:         p.Requests,
			Tokens:           p.Tokens,
			PromptTokens:     p.PromptTokens,
			CompletionTokens: p.CompletionTokens,
			AvgLatencyMs:     p.AvgLatencyMs,
			FailedRequests:   p.FailedRequests,
		}
	}
	return result
}

// ServeUI serves the web UI dashboard.
func ServeUI(c *gin.Context) {
	// Determine which file to serve
	path := c.Param("filepath")
	if path == "" || path == "/" {
		path = "index.html"
	} else {
		// Remove leading slash
		if path[0] == '/' {
			path = path[1:]
		}
	}

	// Read file from embedded FS
	filePath := "web/quantumspring/" + path
	content, err := webAssets.ReadFile(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"path":     path,
			"filepath": filePath,
			"error":    err,
		}).Error("Failed to read embedded file")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "file not found: " + path,
		})
		return
	}

	// Determine content type
	contentType := "text/html"
	if len(path) > 3 && path[len(path)-3:] == ".js" {
		contentType = "application/javascript"
	} else if len(path) > 4 && path[len(path)-4:] == ".css" {
		contentType = "text/css"
	}

	c.Data(http.StatusOK, contentType, content)
}

// RegisterRoutes registers QuantumSpring routes to the router.
func RegisterRoutes(router *gin.Engine, cfg *config.Config) {
	if !cfg.QuantumSpring.Enabled {
		log.Info("QuantumSpring metrics API is disabled")
		return
	}

	qs := router.Group("/_qs")
	{
		// Health endpoint (no auth required)
		qs.GET("/health", GetHealth)

		// Metrics endpoint (with auth if configured)
		qs.GET("/metrics", GetMetrics)

		// UI endpoints
		qs.GET("/metrics/ui", ServeUI)
		qs.GET("/metrics/ui/*filepath", ServeUI)
	}

	log.WithFields(log.Fields{
		"prefix":       "/_qs",
		"bind_address": cfg.QuantumSpring.BindAddress,
		"auth_enabled": cfg.QuantumSpring.BasicAuth.Username != "",
	}).Info("QuantumSpring metrics API registered")
}
