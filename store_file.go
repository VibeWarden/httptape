package httptape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrInvalidID is returned when a tape ID contains path separators or
// directory traversal components that could escape the base directory.
var ErrInvalidID = errors.New("httptape: invalid tape ID")

// FileStore is a filesystem-backed Store implementation. Each tape is persisted
// as a single JSON file.
//
// FileStore is safe for concurrent use by multiple goroutines within a single
// process. It is not safe for multi-process concurrent access to the same
// directory.
type FileStore struct {
	dir string // base directory for fixtures
	mu  sync.RWMutex
}

// FileStoreOption configures a FileStore.
type FileStoreOption func(*FileStore)

// WithDirectory sets the base directory for fixture storage.
// If not set, defaults to "fixtures" in the current working directory.
func WithDirectory(dir string) FileStoreOption {
	return func(fs *FileStore) {
		fs.dir = dir
	}
}

// NewFileStore creates a new FileStore. The base directory is created if it
// does not exist (with mode 0o755).
func NewFileStore(opts ...FileStoreOption) (*FileStore, error) {
	fs := &FileStore{
		dir: "fixtures",
	}
	for _, opt := range opts {
		opt(fs)
	}

	if err := os.MkdirAll(fs.dir, 0o755); err != nil {
		return nil, fmt.Errorf("httptape: filestore create directory %s: %w", fs.dir, err)
	}
	return fs, nil
}

// Save persists a tape as a JSON file. If a tape with the same ID already exists,
// it is overwritten. Writes are atomic via a temporary file and rename.
func (fs *FileStore) Save(ctx context.Context, tape Tape) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("httptape: filestore save %s: %w", tape.ID, err)
	}
	if err := validateID(tape.ID); err != nil {
		return fmt.Errorf("httptape: filestore save: %w", err)
	}

	data, err := json.MarshalIndent(tape, "", "  ")
	if err != nil {
		return fmt.Errorf("httptape: filestore save %s: %w", tape.ID, err)
	}
	data = append(data, '\n') // POSIX trailing newline

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Write to a temporary file first for atomicity.
	tmpFile, err := os.CreateTemp(fs.dir, "tape-*.tmp")
	if err != nil {
		return fmt.Errorf("httptape: filestore save %s: %w", tape.ID, err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("httptape: filestore save %s: %w", tape.ID, err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("httptape: filestore save %s: %w", tape.ID, err)
	}

	target := fs.tapePath(tape.ID)
	if err := os.Rename(tmpPath, target); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("httptape: filestore save %s: %w", tape.ID, err)
	}
	return nil
}

// Load retrieves a single tape by ID from the filesystem.
// Returns an error wrapping ErrNotFound if the file does not exist.
func (fs *FileStore) Load(ctx context.Context, id string) (Tape, error) {
	if err := ctx.Err(); err != nil {
		return Tape{}, fmt.Errorf("httptape: filestore load %s: %w", id, err)
	}
	if err := validateID(id); err != nil {
		return Tape{}, fmt.Errorf("httptape: filestore load: %w", err)
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	data, err := os.ReadFile(fs.tapePath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return Tape{}, fmt.Errorf("httptape: filestore load %s: %w", id, ErrNotFound)
		}
		return Tape{}, fmt.Errorf("httptape: filestore load %s: %w", id, err)
	}

	var tape Tape
	if err := json.Unmarshal(data, &tape); err != nil {
		return Tape{}, fmt.Errorf("httptape: filestore load %s: %w", id, err)
	}
	return tape, nil
}

// List returns all tapes matching the given filter by scanning all JSON files
// in the base directory. Returns an empty slice (not nil) if no tapes match.
func (fs *FileStore) List(ctx context.Context, filter Filter) ([]Tape, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("httptape: filestore list: %w", err)
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("httptape: filestore list: %w", err)
	}

	result := make([]Tape, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("httptape: filestore list: %w", err)
		}

		data, err := os.ReadFile(filepath.Join(fs.dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("httptape: filestore list: %w", err)
		}

		var tape Tape
		if err := json.Unmarshal(data, &tape); err != nil {
			return nil, fmt.Errorf("httptape: filestore list: %w", err)
		}

		if filter.Route != "" && tape.Route != filter.Route {
			continue
		}
		if filter.Method != "" && tape.Request.Method != filter.Method {
			continue
		}
		result = append(result, tape)
	}
	return result, nil
}

// Delete removes a tape by ID from the filesystem.
// Returns an error wrapping ErrNotFound if the file does not exist.
func (fs *FileStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("httptape: filestore delete %s: %w", id, err)
	}
	if err := validateID(id); err != nil {
		return fmt.Errorf("httptape: filestore delete: %w", err)
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.tapePath(id)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("httptape: filestore delete %s: %w", id, ErrNotFound)
		}
		return fmt.Errorf("httptape: filestore delete %s: %w", id, err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("httptape: filestore delete %s: %w", id, err)
	}
	return nil
}

// validateID checks that id is safe to use as a filename component.
// It rejects IDs containing path separators or ".." traversal components.
func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: empty ID", ErrInvalidID)
	}
	if strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("%w: contains path separator", ErrInvalidID)
	}
	if id == ".." || strings.HasPrefix(id, ".."+string(filepath.Separator)) ||
		strings.HasSuffix(id, string(filepath.Separator)+"..") ||
		strings.Contains(id, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: contains path traversal", ErrInvalidID)
	}
	// Also reject the bare ".." even without separators (already handled above)
	// and any cleaned path that would escape.
	if id == "." {
		return fmt.Errorf("%w: invalid ID", ErrInvalidID)
	}
	return nil
}

// tapePath returns the filesystem path for a tape with the given ID.
func (fs *FileStore) tapePath(id string) string {
	return filepath.Join(fs.dir, id+".json")
}
