-- QuantumSpring Usage Statistics Schema
-- SQLite database schema for persistent usage tracking

-- Main usage records table
CREATE TABLE IF NOT EXISTS usage_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Temporal data (stored as TEXT for strftime compatibility)
    timestamp TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,

    -- Request identification
    request_id TEXT,

    -- Authentication & source
    api_key TEXT,
    source TEXT,
    auth_id TEXT,

    -- Provider information
    provider TEXT NOT NULL,
    model TEXT NOT NULL,

    -- Token metrics
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,

    -- Request status
    status INTEGER NOT NULL DEFAULT 200,
    failed BOOLEAN NOT NULL DEFAULT 0,
    latency_ms INTEGER
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_usage_timestamp ON usage_records(timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_model ON usage_records(model);
CREATE INDEX IF NOT EXISTS idx_usage_api_key ON usage_records(api_key);
CREATE INDEX IF NOT EXISTS idx_usage_provider ON usage_records(provider);
CREATE INDEX IF NOT EXISTS idx_usage_created_at ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_failed ON usage_records(failed);

-- Composite index for common queries
CREATE INDEX IF NOT EXISTS idx_usage_timestamp_model ON usage_records(timestamp, model);
CREATE INDEX IF NOT EXISTS idx_usage_timestamp_provider ON usage_records(timestamp, provider);

-- Aggregation materialized view for fast dashboard queries
-- Note: SQLite doesn't support materialized views, but we can create a regular view
-- and refresh it periodically if needed
CREATE VIEW IF NOT EXISTS usage_aggregates AS
SELECT
    DATE(timestamp) as date,
    strftime('%H', timestamp) as hour,
    provider,
    model,
    api_key,
    COUNT(*) as request_count,
    SUM(total_tokens) as total_tokens,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    SUM(reasoning_tokens) as reasoning_tokens,
    SUM(cached_tokens) as cached_tokens,
    AVG(latency_ms) as avg_latency_ms,
    SUM(CASE WHEN failed = 1 THEN 1 ELSE 0 END) as failed_count,
    SUM(CASE WHEN failed = 0 THEN 1 ELSE 0 END) as success_count
FROM usage_records
GROUP BY date, hour, provider, model, api_key;

-- Daily aggregates view
CREATE VIEW IF NOT EXISTS usage_daily AS
SELECT
    DATE(timestamp) as date,
    provider,
    model,
    COUNT(*) as request_count,
    SUM(total_tokens) as total_tokens,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    AVG(latency_ms) as avg_latency_ms,
    SUM(CASE WHEN failed = 1 THEN 1 ELSE 0 END) as failed_count
FROM usage_records
GROUP BY date, provider, model;

-- Model totals view
CREATE VIEW IF NOT EXISTS usage_by_model AS
SELECT
    model,
    COUNT(*) as request_count,
    SUM(total_tokens) as total_tokens,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    AVG(latency_ms) as avg_latency_ms,
    SUM(CASE WHEN failed = 1 THEN 1 ELSE 0 END) as failed_count
FROM usage_records
GROUP BY model;

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial version
INSERT OR IGNORE INTO schema_version (version) VALUES (1);