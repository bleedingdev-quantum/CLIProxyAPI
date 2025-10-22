package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

var (
	globalPersistencePlugin *PersistencePlugin
	globalStorage           Storage
)

// Initialize initializes persistence based on configuration.
// It creates the storage backend and registers the persistence plugin.
//
// Parameters:
//   - cfg: Application configuration
//
// Returns:
//   - error: Any initialization error
func Initialize(cfg *config.Config) error {
	if !cfg.Persistence.Enabled {
		log.Info("Persistence is disabled")
		return nil
	}

	// Parse flush interval
	flushInterval, err := time.ParseDuration(cfg.Persistence.FlushInterval)
	if err != nil {
		log.WithError(err).Warn("Invalid flush interval, using default 10s")
		flushInterval = 10 * time.Second
	}

	// Create storage backend
	var storage Storage
	switch cfg.Persistence.Type {
	case "sqlite", "":
		storage, err = initSQLite(cfg.Persistence.Path)
		if err != nil {
			return fmt.Errorf("failed to initialize SQLite storage: %w", err)
		}
	case "json":
		return fmt.Errorf("JSON storage not implemented yet")
	default:
		return fmt.Errorf("unknown persistence type: %s", cfg.Persistence.Type)
	}

	globalStorage = storage

	// Create persistence plugin
	bufferSize := cfg.Persistence.BufferSize
	if bufferSize <= 0 {
		bufferSize = 100 // default
	}

	plugin := NewPersistencePlugin(storage, bufferSize, flushInterval)
	globalPersistencePlugin = plugin

	// Register plugin with usage manager
	coreusage.RegisterPlugin(plugin)

	log.WithFields(log.Fields{
		"type":           cfg.Persistence.Type,
		"path":           cfg.Persistence.Path,
		"buffer_size":    bufferSize,
		"flush_interval": flushInterval,
		"retention_days": cfg.Persistence.RetentionDays,
	}).Info("Persistence initialized successfully")

	// Start cleanup job if retention is configured
	if cfg.Persistence.RetentionDays > 0 {
		go startCleanupJob(storage, cfg.Persistence.RetentionDays)
	}

	return nil
}

// initSQLite initializes SQLite storage.
func initSQLite(path string) (Storage, error) {
	// Resolve path
	if path == "" {
		path = "./data/usage.db"
	}

	// Expand home directory
	if path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create SQLite storage
	storage, err := NewSQLiteStorage(path)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

// startCleanupJob runs periodic cleanup of old records.
func startCleanupJob(storage Storage, retentionDays int) {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		deleted, err := storage.Cleanup(ctx, cutoff)
		if err != nil {
			log.WithError(err).Error("Failed to cleanup old records")
		} else if deleted > 0 {
			log.WithFields(log.Fields{
				"deleted":      deleted,
				"cutoff":       cutoff,
				"retention_days": retentionDays,
			}).Info("Cleanup job completed")
		}

		cancel()
	}
}

// Shutdown gracefully shuts down persistence.
func Shutdown(ctx context.Context) error {
	if globalPersistencePlugin != nil {
		if err := globalPersistencePlugin.Close(ctx); err != nil {
			return fmt.Errorf("failed to close persistence plugin: %w", err)
		}
		log.Info("Persistence shutdown completed")
	}
	return nil
}

// GetStorage returns the global storage instance (for metrics queries).
func GetStorage() Storage {
	return globalStorage
}

// GetPlugin returns the global persistence plugin instance.
func GetPlugin() *PersistencePlugin {
	return globalPersistencePlugin
}
