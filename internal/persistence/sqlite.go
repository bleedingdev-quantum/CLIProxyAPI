package persistence

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// SQLiteStorage implements Storage interface using SQLite database.
type SQLiteStorage struct {
	db   *sql.DB
	path string
}

// NewSQLiteStorage creates a new SQLite storage instance.
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	storage := &SQLiteStorage{
		db:   db,
		path: path,
	}

	log.WithField("path", path).Info("SQLite storage initialized")
	return storage, nil
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Insert inserts a single usage record.
func (s *SQLiteStorage) Insert(ctx context.Context, record UsageRecord) error {
	query := `
		INSERT INTO usage_records (
			timestamp, request_id, api_key, source, auth_id,
			provider, model,
			prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, total_tokens,
			status, failed, latency_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Format timestamp as ISO string for SQLite compatibility with strftime()
	timestampStr := record.Timestamp.Format("2006-01-02 15:04:05")

	_, err := s.db.ExecContext(ctx, query,
		timestampStr, record.RequestID, record.APIKey, record.Source, record.AuthID,
		record.Provider, record.Model,
		record.PromptTokens, record.CompletionTokens, record.ReasoningTokens, record.CachedTokens, record.TotalTokens,
		record.Status, record.Failed, record.LatencyMs,
	)

	if err != nil {
		return fmt.Errorf("failed to insert record: %w", err)
	}

	return nil
}

// InsertBatch inserts multiple usage records in a single transaction.
func (s *SQLiteStorage) InsertBatch(ctx context.Context, records []UsageRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO usage_records (
			timestamp, request_id, api_key, source, auth_id,
			provider, model,
			prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, total_tokens,
			status, failed, latency_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		// Format timestamp as ISO string for SQLite compatibility with strftime()
		timestampStr := record.Timestamp.Format("2006-01-02 15:04:05")

		_, err := stmt.ExecContext(ctx,
			timestampStr, record.RequestID, record.APIKey, record.Source, record.AuthID,
			record.Provider, record.Model,
			record.PromptTokens, record.CompletionTokens, record.ReasoningTokens, record.CachedTokens, record.TotalTokens,
			record.Status, record.Failed, record.LatencyMs,
		)
		if err != nil {
			return fmt.Errorf("failed to insert record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Query performs a filtered query on usage records.
func (s *SQLiteStorage) Query(ctx context.Context, filter QueryFilter) ([]UsageRecord, error) {
	query := `
		SELECT
			id, timestamp, request_id, api_key, source, auth_id,
			provider, model,
			prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, total_tokens,
			status, failed, latency_ms, created_at
		FROM usage_records
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.From != nil && !filter.From.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.From.Format("2006-01-02 15:04:05"))
	}

	if filter.To != nil && !filter.To.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.To.Format("2006-01-02 15:04:05"))
	}

	if filter.Provider != "" {
		query += " AND provider = ?"
		args = append(args, filter.Provider)
	}

	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}

	if filter.APIKey != "" {
		query += " AND api_key = ?"
		args = append(args, filter.APIKey)
	}

	if filter.Failed != nil {
		query += " AND failed = ?"
		args = append(args, *filter.Failed)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []UsageRecord
	for rows.Next() {
		var r UsageRecord
		var timestampStr, createdAtStr sql.NullString

		err := rows.Scan(
			&r.ID, &timestampStr, &r.RequestID, &r.APIKey, &r.Source, &r.AuthID,
			&r.Provider, &r.Model,
			&r.PromptTokens, &r.CompletionTokens, &r.ReasoningTokens, &r.CachedTokens, &r.TotalTokens,
			&r.Status, &r.Failed, &r.LatencyMs, &createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		// Parse timestamp
		if timestampStr.Valid {
			r.Timestamp, _ = time.Parse("2006-01-02 15:04:05", timestampStr.String)
		}

		// Parse created_at
		if createdAtStr.Valid {
			r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr.String)
		}

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return records, nil
}

// GetTotals retrieves aggregated totals for the given time range.
func (s *SQLiteStorage) GetTotals(ctx context.Context, from, to time.Time) (*TotalMetrics, error) {
	query := `
		SELECT
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(reasoning_tokens), 0) as reasoning_tokens,
			COALESCE(SUM(cached_tokens), 0) as cached_tokens,
			COALESCE(SUM(CASE WHEN failed = 1 THEN 1 ELSE 0 END), 0) as failed_requests,
			COALESCE(SUM(CASE WHEN failed = 0 THEN 1 ELSE 0 END), 0) as success_requests,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms
		FROM usage_records
		WHERE timestamp >= ? AND timestamp <= ?
	`

	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")

	var totals TotalMetrics
	err := s.db.QueryRowContext(ctx, query, fromStr, toStr).Scan(
		&totals.Requests,
		&totals.Tokens,
		&totals.PromptTokens,
		&totals.CompletionTokens,
		&totals.ReasoningTokens,
		&totals.CachedTokens,
		&totals.FailedRequests,
		&totals.SuccessRequests,
		&totals.AvgLatencyMs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get totals: %w", err)
	}

	return &totals, nil
}

// GetByModel retrieves metrics aggregated by model.
func (s *SQLiteStorage) GetByModel(ctx context.Context, from, to time.Time) ([]ModelMetrics, error) {
	query := `
		SELECT
			model,
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
			COALESCE(SUM(CASE WHEN failed = 1 THEN 1 ELSE 0 END), 0) as failed_requests
		FROM usage_records
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY model
		ORDER BY total_tokens DESC
	`

	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")

	rows, err := s.db.QueryContext(ctx, query, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query by model: %w", err)
	}
	defer rows.Close()

	var metrics []ModelMetrics
	for rows.Next() {
		var m ModelMetrics
		err := rows.Scan(
			&m.Model, &m.Requests, &m.Tokens,
			&m.PromptTokens, &m.CompletionTokens,
			&m.AvgLatencyMs, &m.FailedRequests,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model metrics: %w", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetTimeseries retrieves time-based aggregations.
func (s *SQLiteStorage) GetTimeseries(ctx context.Context, from, to time.Time, interval string) ([]TimeseriesPoint, error) {
	var timeFormat string
	switch interval {
	case "hour":
		timeFormat = "%Y-%m-%d %H:00:00"
	case "day":
		timeFormat = "%Y-%m-%d 00:00:00"
	case "week":
		timeFormat = "%Y-W%W"
	case "month":
		timeFormat = "%Y-%m-01 00:00:00"
	default:
		timeFormat = "%Y-%m-%d %H:00:00" // default to hourly
	}

	// Use proper string formatting for SQL query
	query := `
		SELECT
			strftime('` + timeFormat + `', timestamp) as bucket,
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
			COALESCE(SUM(CASE WHEN failed = 1 THEN 1 ELSE 0 END), 0) as failed_requests
		FROM usage_records
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY bucket
		ORDER BY bucket ASC
	`

	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")

	rows, err := s.db.QueryContext(ctx, query, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeseries: %w", err)
	}
	defer rows.Close()

	var points []TimeseriesPoint
	for rows.Next() {
		var bucketStr sql.NullString
		var p TimeseriesPoint

		err := rows.Scan(
			&bucketStr, &p.Requests, &p.Tokens,
			&p.PromptTokens, &p.CompletionTokens,
			&p.AvgLatencyMs, &p.FailedRequests,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeseries point: %w", err)
		}

		// Skip if bucket is NULL (no data)
		if !bucketStr.Valid || bucketStr.String == "" {
			continue
		}

		// Parse bucket timestamp
		if interval == "week" {
			// Week format is special (e.g., "2025-W03")
			p.BucketStart, _ = time.Parse("2006-W02", bucketStr.String)
		} else {
			p.BucketStart, _ = time.Parse("2006-01-02 15:04:05", bucketStr.String)
		}

		points = append(points, p)
	}

	return points, nil
}

// Aggregate performs complex aggregation based on the query parameters.
func (s *SQLiteStorage) Aggregate(ctx context.Context, query AggregateQuery) (*AggregateResult, error) {
	// This is a simplified implementation
	// For more complex aggregations, extend as needed
	totals, err := s.GetTotals(ctx, query.From, query.To)
	if err != nil {
		return nil, err
	}

	byModel, err := s.GetByModel(ctx, query.From, query.To)
	if err != nil {
		return nil, err
	}

	return &AggregateResult{
		Totals:  *totals,
		ByModel: byModel,
	}, nil
}

// Cleanup removes records older than the specified time.
func (s *SQLiteStorage) Cleanup(ctx context.Context, olderThan time.Time) (int64, error) {
	olderThanStr := olderThan.Format("2006-01-02 15:04:05")
	result, err := s.db.ExecContext(ctx, "DELETE FROM usage_records WHERE timestamp < ?", olderThanStr)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup records: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if deleted > 0 {
		log.WithFields(log.Fields{
			"deleted": deleted,
			"before":  olderThan,
		}).Info("Cleanup completed")
	}

	return deleted, nil
}

// GetRecordCount returns the total number of records in the database.
func (s *SQLiteStorage) GetRecordCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_records").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get record count: %w", err)
	}
	return count, nil
}
