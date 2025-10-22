package persistence

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

// PersistencePlugin implements coreusage.Plugin to persist usage records to storage.
// It buffers records in memory and flushes them periodically or when buffer is full.
type PersistencePlugin struct {
	storage       Storage
	buffer        []UsageRecord
	bufferSize    int
	flushInterval time.Duration

	mu         sync.Mutex
	stopCh     chan struct{}
	flushTimer *time.Timer
	stopped    bool
}

// NewPersistencePlugin creates a new persistence plugin.
//
// Parameters:
//   - storage: Storage backend to use
//   - bufferSize: Maximum number of records to buffer before flushing
//   - flushInterval: Maximum time to wait before flushing buffered records
//
// Returns:
//   - *PersistencePlugin: Initialized plugin instance
func NewPersistencePlugin(storage Storage, bufferSize int, flushInterval time.Duration) *PersistencePlugin {
	if bufferSize <= 0 {
		bufferSize = 100 // default buffer size
	}
	if flushInterval <= 0 {
		flushInterval = 10 * time.Second // default flush interval
	}

	plugin := &PersistencePlugin{
		storage:       storage,
		buffer:        make([]UsageRecord, 0, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}

	// Start background flush timer
	plugin.startFlushTimer()

	log.WithFields(log.Fields{
		"buffer_size":    bufferSize,
		"flush_interval": flushInterval,
	}).Info("Persistence plugin initialized")

	return plugin
}

// HandleUsage implements coreusage.Plugin interface.
// It buffers the usage record and flushes when buffer is full or timer expires.
func (p *PersistencePlugin) HandleUsage(ctx context.Context, record coreusage.Record) {
	if p == nil || p.storage == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}

	// Convert core usage record to persistence record
	persRecord := p.convertRecord(record)

	// Add to buffer
	p.buffer = append(p.buffer, persRecord)

	// Flush if buffer is full
	if len(p.buffer) >= p.bufferSize {
		p.flushLocked(ctx)
	}
}

// convertRecord converts core usage record to persistence record.
func (p *PersistencePlugin) convertRecord(record coreusage.Record) UsageRecord {
	timestamp := record.RequestedAt
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Determine status code (default to 200 for success, 500 for failure)
	status := 200
	if record.Failed {
		status = 500
	}

	return UsageRecord{
		Timestamp: timestamp,
		APIKey:    record.APIKey,
		Source:    record.Source,
		AuthID:    record.AuthID,
		Provider:  record.Provider,
		Model:     record.Model,

		PromptTokens:     record.Detail.InputTokens,
		CompletionTokens: record.Detail.OutputTokens,
		ReasoningTokens:  record.Detail.ReasoningTokens,
		CachedTokens:     record.Detail.CachedTokens,
		TotalTokens:      record.Detail.TotalTokens,

		Status: status,
		Failed: record.Failed,
	}
}

// startFlushTimer starts the background timer for periodic flushes.
func (p *PersistencePlugin) startFlushTimer() {
	go func() {
		ticker := time.NewTicker(p.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.mu.Lock()
				if !p.stopped && len(p.buffer) > 0 {
					p.flushLocked(context.Background())
				}
				p.mu.Unlock()

			case <-p.stopCh:
				return
			}
		}
	}()
}

// flushLocked flushes buffered records to storage.
// Caller must hold the mutex.
func (p *PersistencePlugin) flushLocked(ctx context.Context) {
	if len(p.buffer) == 0 {
		return
	}

	// Copy buffer for async write
	toFlush := make([]UsageRecord, len(p.buffer))
	copy(toFlush, p.buffer)

	// Clear buffer
	p.buffer = p.buffer[:0]

	// Write to storage in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := p.storage.InsertBatch(ctx, toFlush); err != nil {
			log.WithError(err).WithField("count", len(toFlush)).Error("Failed to flush usage records to storage")
		} else {
			log.WithField("count", len(toFlush)).Debug("Flushed usage records to storage")
		}
	}()
}

// Flush manually flushes all buffered records.
func (p *PersistencePlugin) Flush(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.buffer) == 0 {
		return nil
	}

	toFlush := make([]UsageRecord, len(p.buffer))
	copy(toFlush, p.buffer)
	p.buffer = p.buffer[:0]

	return p.storage.InsertBatch(ctx, toFlush)
}

// Stop stops the persistence plugin and flushes remaining records.
func (p *PersistencePlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return nil
	}
	p.stopped = true
	p.mu.Unlock()

	// Signal stop
	close(p.stopCh)

	// Final flush
	return p.Flush(ctx)
}

// Close closes the plugin and underlying storage.
func (p *PersistencePlugin) Close(ctx context.Context) error {
	if err := p.Stop(ctx); err != nil {
		log.WithError(err).Error("Failed to stop persistence plugin")
	}

	if p.storage != nil {
		return p.storage.Close()
	}

	return nil
}

// GetStorage returns the underlying storage (for metrics queries).
func (p *PersistencePlugin) GetStorage() Storage {
	return p.storage
}
