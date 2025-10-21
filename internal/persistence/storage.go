// Package persistence provides persistent storage for usage statistics.
// It supports SQLite backend with efficient querying and aggregation.
package persistence

import (
	"context"
	"time"
)

// UsageRecord represents a single API request record with complete metrics.
type UsageRecord struct {
	ID        int64
	Timestamp time.Time
	CreatedAt time.Time

	// Request identification
	RequestID string

	// Authentication & source
	APIKey string
	Source string
	AuthID string

	// Provider information
	Provider string
	Model    string

	// Token metrics
	PromptTokens     int64
	CompletionTokens int64
	ReasoningTokens  int64
	CachedTokens     int64
	TotalTokens      int64

	// Request status
	Status    int
	Failed    bool
	LatencyMs int64
}

// QueryFilter defines filters for querying usage records.
type QueryFilter struct {
	From     *time.Time
	To       *time.Time
	Model    string
	APIKey   string
	Provider string
	Failed   *bool
	Limit    int
	Offset   int
}

// AggregateQuery defines parameters for aggregation queries.
type AggregateQuery struct {
	From     time.Time
	To       time.Time
	Interval string   // "hour", "day", "week", "month"
	GroupBy  []string // "model", "api_key", "provider"
}

// AggregateResult contains aggregated metrics.
type AggregateResult struct {
	Totals       TotalMetrics
	ByModel      []ModelMetrics
	ByAPIKey     []APIKeyMetrics
	ByProvider   []ProviderMetrics
	Timeseries   []TimeseriesPoint
}

// TotalMetrics contains overall aggregated metrics.
type TotalMetrics struct {
	Requests         int64
	Tokens           int64
	PromptTokens     int64
	CompletionTokens int64
	ReasoningTokens  int64
	CachedTokens     int64
	FailedRequests   int64
	SuccessRequests  int64
	AvgLatencyMs     float64
}

// ModelMetrics contains metrics aggregated by model.
type ModelMetrics struct {
	Model            string
	Requests         int64
	Tokens           int64
	PromptTokens     int64
	CompletionTokens int64
	AvgLatencyMs     float64
	FailedRequests   int64
}

// APIKeyMetrics contains metrics aggregated by API key.
type APIKeyMetrics struct {
	APIKey   string
	Requests int64
	Tokens   int64
}

// ProviderMetrics contains metrics aggregated by provider.
type ProviderMetrics struct {
	Provider     string
	Requests     int64
	Tokens       int64
	AvgLatencyMs float64
}

// TimeseriesPoint represents a single point in time-based aggregation.
type TimeseriesPoint struct {
	BucketStart      time.Time
	Requests         int64
	Tokens           int64
	PromptTokens     int64
	CompletionTokens int64
	AvgLatencyMs     float64
	FailedRequests   int64
}

// Storage defines the interface for persistent usage statistics storage.
type Storage interface {
	// Write operations
	Insert(ctx context.Context, record UsageRecord) error
	InsertBatch(ctx context.Context, records []UsageRecord) error

	// Read operations
	Query(ctx context.Context, filter QueryFilter) ([]UsageRecord, error)
	Aggregate(ctx context.Context, query AggregateQuery) (*AggregateResult, error)

	// Statistics
	GetTotals(ctx context.Context, from, to time.Time) (*TotalMetrics, error)
	GetByModel(ctx context.Context, from, to time.Time) ([]ModelMetrics, error)
	GetTimeseries(ctx context.Context, from, to time.Time, interval string) ([]TimeseriesPoint, error)

	// Maintenance
	Cleanup(ctx context.Context, olderThan time.Time) (int64, error)
	GetRecordCount(ctx context.Context) (int64, error)

	// Lifecycle
	Close() error
}