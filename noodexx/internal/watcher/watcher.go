package watcher

import (
	"context"
	"fmt"
	"log"
	"noodexx/internal/logging"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors folders for file changes
type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	ingester    Ingester
	store       Store
	privacyMode bool
	allowedExts []string
	maxSize     int64
	logger      *logging.Logger
}

// Ingester interface for processing files
type Ingester interface {
	IngestText(ctx context.Context, source, text string, tags []string) error
}

// Store interface for folder management
type Store interface {
	AddWatchedFolder(ctx context.Context, path string) error
	GetWatchedFolders(ctx context.Context) ([]WatchedFolder, error)
	DeleteSource(ctx context.Context, source string) error
}

// WatchedFolder represents a monitored directory
type WatchedFolder struct {
	ID       int64
	Path     string
	Active   bool
	LastScan time.Time
}

// NewWatcher creates a folder watcher with fsnotify initialization
func NewWatcher(ingester Ingester, store Store, privacyMode bool, logger *logging.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to create fsnotify watcher")
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &Watcher{
		fsWatcher:   fsw,
		ingester:    ingester,
		store:       store,
		privacyMode: privacyMode,
		allowedExts: []string{".txt", ".md", ".pdf"},
		maxSize:     10 * 1024 * 1024, // 10MB
		logger:      logger,
	}, nil
}

// Start begins watching configured folders and starts event loop
func (w *Watcher) Start(ctx context.Context) error {
	w.logger.Debug("starting file watcher")

	// Load watched folders from database
	folders, err := w.store.GetWatchedFolders(ctx)
	if err != nil {
		w.logger.WithContext("error", err.Error()).Error("failed to load watched folders")
		return fmt.Errorf("failed to load watched folders: %w", err)
	}

	// Add each folder to fsnotify
	for _, folder := range folders {
		if !folder.Active {
			continue
		}

		if err := w.validatePath(folder.Path); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"folder_path": folder.Path,
				"error":       err.Error(),
			}).Warn("skipping invalid folder")
			continue
		}

		if err := w.fsWatcher.Add(folder.Path); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"folder_path": folder.Path,
				"error":       err.Error(),
			}).Warn("failed to watch folder")
			continue
		}

		w.logger.WithContext("folder_path", folder.Path).Debug("watching folder")
	}

	// Start event loop in goroutine
	go w.eventLoop(ctx)

	w.logger.WithContext("folder_count", len(folders)).Debug("file watcher started")
	return nil
}

// eventLoop processes filesystem events
func (w *Watcher) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.fsWatcher.Close()
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			w.handleEvent(ctx, event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.logger.WithContext("error", err.Error()).Error("watcher error")
		}
	}
}

// handleEvent processes create/modify/delete events
func (w *Watcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	logger := w.logger.WithFields(map[string]interface{}{
		"file_path":  event.Name,
		"event_type": event.Op.String(),
	})

	// Check if it's a file we care about
	if !w.shouldProcess(event.Name) {
		return
	}

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		logger.Debug("file created")
		w.ingestFile(ctx, event.Name)

	case event.Op&fsnotify.Write == fsnotify.Write:
		logger.Debug("file modified")
		w.ingestFile(ctx, event.Name)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		logger.Debug("file deleted")
		w.deleteFile(ctx, event.Name)
	}
}

// shouldProcess checks extension and size validation
func (w *Watcher) shouldProcess(path string) bool {
	// Check extension
	ext := strings.ToLower(filepath.Ext(path))
	allowed := false
	for _, allowedExt := range w.allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}

	if !allowed {
		return false
	}

	// Check file size (only for existing files, not for delete events)
	info, err := os.Stat(path)
	if err != nil {
		// File might not exist (delete event) or other error
		// For delete events, we still want to process
		if os.IsNotExist(err) {
			return true
		}
		return false
	}

	if info.Size() > w.maxSize {
		log.Printf("File %s exceeds size limit (%d > %d)", path, info.Size(), w.maxSize)
		return false
	}

	return true
}

// validatePath blocks system directories
func (w *Watcher) validatePath(path string) error {
	// Block system directories
	systemDirs := []string{"/etc", "/System", "/Windows", "/sys", "/proc", "C:\\Windows", "C:\\System"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(path, sysDir) {
			return fmt.Errorf("cannot watch system directory: %s", path)
		}
	}

	// Ensure path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}

	// Ensure it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// AddFolder adds a new folder to watch
func (w *Watcher) AddFolder(ctx context.Context, path string) error {
	logger := w.logger.WithContext("folder_path", path)
	logger.Debug("adding watched folder")

	if err := w.validatePath(path); err != nil {
		logger.WithContext("error", err.Error()).Error("invalid folder path")
		return err
	}

	if err := w.fsWatcher.Add(path); err != nil {
		logger.WithContext("error", err.Error()).Error("failed to add folder to watcher")
		return fmt.Errorf("failed to add folder to watcher: %w", err)
	}

	if err := w.store.AddWatchedFolder(ctx, path); err != nil {
		// Remove from fsnotify if database save fails
		w.fsWatcher.Remove(path)
		logger.WithContext("error", err.Error()).Error("failed to save watched folder")
		return fmt.Errorf("failed to save watched folder: %w", err)
	}

	logger.Debug("watched folder added successfully")
	return nil
}

// ingestFile processes a file by reading it and calling ingester
func (w *Watcher) ingestFile(ctx context.Context, path string) {
	logger := w.logger.WithContext("file_path", path)

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to read file")
		return
	}

	// Use file path as source
	tags := []string{"auto-ingested"}

	// Ingest the text
	if err := w.ingester.IngestText(ctx, path, string(content), tags); err != nil {
		logger.WithContext("error", err.Error()).Error("failed to ingest file")
	} else {
		logger.Debug("file ingested successfully")
	}
}

// deleteFile removes chunks for a deleted file
func (w *Watcher) deleteFile(ctx context.Context, path string) {
	logger := w.logger.WithContext("file_path", path)

	if err := w.store.DeleteSource(ctx, path); err != nil {
		logger.WithContext("error", err.Error()).Error("failed to delete chunks")
	} else {
		logger.Debug("chunks deleted successfully")
	}
}
