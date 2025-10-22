package persistence

import (
	"context"
	"testing"
	"time"
)

func TestNewSQLiteStorage(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	if storage == nil {
		t.Fatal("Expected storage to be non-nil")
	}
}

func TestInsert(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	record := UsageRecord{
		Timestamp: time.Now(),
		Provider:  "openai",
		Model:     "gpt-4",
		APIKey:    "test-key",
		Source:    "test@example.com",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Status:    200,
		Failed:    false,
		LatencyMs: 1500,
	}

	ctx := context.Background()
	err = storage.Insert(ctx, record)
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	// Verify count
	count, err := storage.GetRecordCount(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestInsertBatch(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	records := []UsageRecord{
		{
			Timestamp: time.Now(),
			Provider:  "openai",
			Model:     "gpt-4",
			TotalTokens: 100,
		},
		{
			Timestamp: time.Now(),
			Provider:  "anthropic",
			Model:     "claude-3-opus",
			TotalTokens: 200,
		},
		{
			Timestamp: time.Now(),
			Provider:  "google",
			Model:     "gemini-pro",
			TotalTokens: 150,
		},
	}

	ctx := context.Background()
	err = storage.InsertBatch(ctx, records)
	if err != nil {
		t.Fatalf("Failed to insert batch: %v", err)
	}

	count, err := storage.GetRecordCount(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestGetTotals(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	now := time.Now()
	records := []UsageRecord{
		{
			Timestamp: now,
			Provider:  "openai",
			Model:     "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Status:    200,
			Failed:    false,
		},
		{
			Timestamp: now,
			Provider:  "openai",
			Model:     "gpt-4",
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
			Status:    200,
			Failed:    false,
		},
		{
			Timestamp: now,
			Provider:  "openai",
			Model:     "gpt-4",
			PromptTokens:     50,
			CompletionTokens: 25,
			TotalTokens:      75,
			Status:    500,
			Failed:    true,
		},
	}

	ctx := context.Background()
	storage.InsertBatch(ctx, records)

	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)

	totals, err := storage.GetTotals(ctx, from, to)
	if err != nil {
		t.Fatalf("Failed to get totals: %v", err)
	}

	if totals.Requests != 3 {
		t.Errorf("Expected 3 requests, got %d", totals.Requests)
	}

	if totals.Tokens != 525 {
		t.Errorf("Expected 525 tokens, got %d", totals.Tokens)
	}

	if totals.PromptTokens != 350 {
		t.Errorf("Expected 350 prompt tokens, got %d", totals.PromptTokens)
	}

	if totals.CompletionTokens != 175 {
		t.Errorf("Expected 175 completion tokens, got %d", totals.CompletionTokens)
	}

	if totals.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", totals.FailedRequests)
	}

	if totals.SuccessRequests != 2 {
		t.Errorf("Expected 2 success requests, got %d", totals.SuccessRequests)
	}
}

func TestGetByModel(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	now := time.Now()
	records := []UsageRecord{
		{Timestamp: now, Provider: "openai", Model: "gpt-4", TotalTokens: 100},
		{Timestamp: now, Provider: "openai", Model: "gpt-4", TotalTokens: 200},
		{Timestamp: now, Provider: "anthropic", Model: "claude-3-opus", TotalTokens: 150},
		{Timestamp: now, Provider: "google", Model: "gemini-pro", TotalTokens: 75},
	}

	ctx := context.Background()
	err = storage.InsertBatch(ctx, records)
	if err != nil {
		t.Fatalf("Failed to insert batch: %v", err)
	}

	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)

	byModel, err := storage.GetByModel(ctx, from, to)
	if err != nil {
		t.Fatalf("Failed to get by model: %v", err)
	}

	if len(byModel) != 3 {
		t.Errorf("Expected 3 models, got %d", len(byModel))
	}

	// Find gpt-4
	var gpt4 *ModelMetrics
	for i := range byModel {
		if byModel[i].Model == "gpt-4" {
			gpt4 = &byModel[i]
			break
		}
	}

	if gpt4 == nil {
		t.Fatal("gpt-4 not found in results")
	}

	if gpt4.Requests != 2 {
		t.Errorf("Expected 2 gpt-4 requests, got %d", gpt4.Requests)
	}

	if gpt4.Tokens != 300 {
		t.Errorf("Expected 300 gpt-4 tokens, got %d", gpt4.Tokens)
	}
}

func TestGetTimeseries(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	now := time.Now()
	hour1 := now.Add(-2 * time.Hour)
	hour2 := now.Add(-1 * time.Hour)

	records := []UsageRecord{
		{Timestamp: hour1, Provider: "test", Model: "test-model", TotalTokens: 100},
		{Timestamp: hour1, Provider: "test", Model: "test-model", TotalTokens: 50},
		{Timestamp: hour2, Provider: "test", Model: "test-model", TotalTokens: 200},
	}

	ctx := context.Background()
	err = storage.InsertBatch(ctx, records)
	if err != nil {
		t.Fatalf("Failed to insert batch: %v", err)
	}

	from := now.Add(-3 * time.Hour)
	to := now

	timeseries, err := storage.GetTimeseries(ctx, from, to, "hour")
	if err != nil {
		t.Fatalf("Failed to get timeseries: %v", err)
	}

	if len(timeseries) != 2 {
		t.Errorf("Expected 2 timeseries points, got %d", len(timeseries))
	}
}

func TestCleanup(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	now := time.Now()
	oldDate := now.Add(-100 * 24 * time.Hour)
	recentDate := now.Add(-1 * time.Hour)

	records := []UsageRecord{
		{Timestamp: oldDate, Provider: "test", Model: "old-model", TotalTokens: 100},
		{Timestamp: recentDate, Provider: "test", Model: "recent-model", TotalTokens: 200},
	}

	ctx := context.Background()
	err = storage.InsertBatch(ctx, records)
	if err != nil {
		t.Fatalf("Failed to insert batch: %v", err)
	}

	// Cleanup records older than 90 days
	cutoff := now.Add(-90 * 24 * time.Hour)
	deleted, err := storage.Cleanup(ctx, cutoff)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted record, got %d", deleted)
	}

	// Verify only recent record remains
	count, err := storage.GetRecordCount(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 remaining record, got %d", count)
	}
}

func TestQuery(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	now := time.Now()
	records := []UsageRecord{
		{Timestamp: now, Model: "gpt-4", Provider: "openai", TotalTokens: 100},
		{Timestamp: now, Model: "claude-3-opus", Provider: "anthropic", TotalTokens: 200},
		{Timestamp: now, Model: "gpt-4", Provider: "openai", TotalTokens: 150},
	}

	ctx := context.Background()
	err = storage.InsertBatch(ctx, records)
	if err != nil {
		t.Fatalf("Failed to insert batch: %v", err)
	}

	// Query for gpt-4 only
	filter := QueryFilter{
		Model: "gpt-4",
		Limit: 10,
	}

	results, err := storage.Query(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Model != "gpt-4" {
			t.Errorf("Expected gpt-4, got %s", r.Model)
		}
	}
}
