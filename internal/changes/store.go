package changes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// ChangesDir is the subdirectory under sdd/ where changes live.
	ChangesDir = "changes"
	// HistoryDir is the subdirectory under sdd/ where archived changes live.
	HistoryDir = "history"
	// ChangeConfigFile is the filename for change records.
	ChangeConfigFile = "change.json"
	// ADRsDir is the subdirectory within a change for ADRs.
	ADRsDir = "adrs"
)

// Store defines the persistence interface for change records.
// Abstracted for testability (DIP).
type Store interface {
	Create(projectRoot string, change *ChangeRecord) error
	Load(projectRoot, changeID string) (*ChangeRecord, error)
	LoadActive(projectRoot string) (*ChangeRecord, error)
	Save(projectRoot string, change *ChangeRecord) error
	Archive(projectRoot, changeID string) error
	List(projectRoot string) ([]ChangeRecord, error)
}

// FileStore implements Store using the local filesystem.
type FileStore struct{}

// NewFileStore creates a filesystem-backed change store.
func NewFileStore() *FileStore {
	return &FileStore{}
}

// ChangesPath returns the absolute path to the sdd/changes/ directory.
func ChangesPath(projectRoot string) string {
	return filepath.Join(projectRoot, "sdd", ChangesDir)
}

// HistoryPath returns the absolute path to the sdd/history/ directory.
func HistoryPath(projectRoot string) string {
	return filepath.Join(projectRoot, "sdd", HistoryDir)
}

// ChangePath returns the absolute path to a specific change's directory.
func ChangePath(projectRoot, changeID string) string {
	return filepath.Join(ChangesPath(projectRoot), changeID)
}

// ChangeConfigPath returns the absolute path to a change's change.json.
func ChangeConfigPath(projectRoot, changeID string) string {
	return filepath.Join(ChangePath(projectRoot, changeID), ChangeConfigFile)
}

// Create persists a new change record, creating the directory structure.
// If the slug already exists, appends a numeric suffix (-2, -3, etc.).
func (fs *FileStore) Create(projectRoot string, change *ChangeRecord) error {
	changesDir := ChangesPath(projectRoot)
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		return fmt.Errorf("creating changes directory: %w", err)
	}

	// Handle slug collisions.
	originalID := change.ID
	changeDir := ChangePath(projectRoot, change.ID)
	suffix := 2
	for {
		if _, err := os.Stat(changeDir); os.IsNotExist(err) {
			break
		}
		change.ID = fmt.Sprintf("%s-%d", originalID, suffix)
		changeDir = ChangePath(projectRoot, change.ID)
		suffix++
	}

	// Create change directory and adrs subdirectory.
	adrsDir := filepath.Join(changeDir, ADRsDir)
	if err := os.MkdirAll(adrsDir, 0o755); err != nil {
		return fmt.Errorf("creating change directory: %w", err)
	}

	return fs.writeConfig(projectRoot, change)
}

// Load reads a specific change record by ID.
func (fs *FileStore) Load(projectRoot, changeID string) (*ChangeRecord, error) {
	path := ChangeConfigPath(projectRoot, changeID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("change %q not found", changeID)
		}
		return nil, fmt.Errorf("reading change config: %w", err)
	}

	var change ChangeRecord
	if err := json.Unmarshal(data, &change); err != nil {
		return nil, fmt.Errorf("parsing change.json for %q: %w", changeID, err)
	}
	return &change, nil
}

// LoadActive scans all changes and returns the one with status "active".
// Returns nil (not an error) if no active change exists.
func (fs *FileStore) LoadActive(projectRoot string) (*ChangeRecord, error) {
	changesDir := ChangesPath(projectRoot)
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading changes directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		change, err := fs.Load(projectRoot, entry.Name())
		if err != nil {
			continue // skip unreadable changes
		}
		if change.Status == StatusActive {
			return change, nil
		}
	}

	return nil, nil
}

// Save updates an existing change record.
func (fs *FileStore) Save(projectRoot string, change *ChangeRecord) error {
	change.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return fs.writeConfig(projectRoot, change)
}

// Archive moves a completed change from changes/ to history/.
func (fs *FileStore) Archive(projectRoot, changeID string) error {
	change, err := fs.Load(projectRoot, changeID)
	if err != nil {
		return err
	}

	if change.Status == StatusActive {
		return fmt.Errorf("cannot archive active change %q — complete it first", changeID)
	}

	srcDir := ChangePath(projectRoot, changeID)
	historyDir := HistoryPath(projectRoot)
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	dstDir := filepath.Join(historyDir, changeID)
	if _, err := os.Stat(dstDir); err == nil {
		return fmt.Errorf("change %q already exists in history", changeID)
	}

	// Update status before moving.
	change.Status = StatusArchived
	change.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := fs.writeConfig(projectRoot, change); err != nil {
		return fmt.Errorf("updating change status: %w", err)
	}

	if err := os.Rename(srcDir, dstDir); err != nil {
		return fmt.Errorf("moving change to history: %w", err)
	}

	return nil
}

// List returns all changes from both changes/ and history/ directories.
func (fs *FileStore) List(projectRoot string) ([]ChangeRecord, error) {
	var result []ChangeRecord

	// Scan changes/ directory.
	changesDir := ChangesPath(projectRoot)
	if entries, err := os.ReadDir(changesDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			change, err := fs.Load(projectRoot, entry.Name())
			if err != nil {
				continue
			}
			result = append(result, *change)
		}
	}

	// Scan history/ directory.
	historyDir := HistoryPath(projectRoot)
	if entries, err := os.ReadDir(historyDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			configPath := filepath.Join(historyDir, entry.Name(), ChangeConfigFile)
			data, err := os.ReadFile(configPath)
			if err != nil {
				continue
			}
			var change ChangeRecord
			if err := json.Unmarshal(data, &change); err != nil {
				continue
			}
			result = append(result, change)
		}
	}

	return result, nil
}

// writeConfig marshals and writes a change record to its change.json.
func (fs *FileStore) writeConfig(projectRoot string, change *ChangeRecord) error {
	data, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling change config: %w", err)
	}

	path := ChangeConfigPath(projectRoot, change.ID)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating change directory: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}
