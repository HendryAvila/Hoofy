package changes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- Helper: create a minimal ChangeRecord for testing ---

func testChangeRecord(id, desc string, ct ChangeType, cs ChangeSize) *ChangeRecord {
	flow, _ := StageFlow(ct, cs)
	stages := make([]StageEntry, len(flow))
	for i, s := range flow {
		status := "pending"
		if i == 0 {
			status = "in_progress"
		}
		stages[i] = StageEntry{Name: s, Status: status}
	}
	return &ChangeRecord{
		ID:           id,
		Type:         ct,
		Size:         cs,
		Description:  desc,
		Stages:       stages,
		CurrentStage: flow[0],
		Status:       StatusActive,
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-01T00:00:00Z",
	}
}

// --- Path helpers ---

func TestChangesPath(t *testing.T) {
	got := ChangesPath("/root")
	want := filepath.Join("/root", "sdd", ChangesDir)
	if got != want {
		t.Errorf("ChangesPath = %s, want %s", got, want)
	}
}

func TestHistoryPath(t *testing.T) {
	got := HistoryPath("/root")
	want := filepath.Join("/root", "sdd", HistoryDir)
	if got != want {
		t.Errorf("HistoryPath = %s, want %s", got, want)
	}
}

func TestChangePath(t *testing.T) {
	got := ChangePath("/root", "fix-bug")
	want := filepath.Join("/root", "sdd", ChangesDir, "fix-bug")
	if got != want {
		t.Errorf("ChangePath = %s, want %s", got, want)
	}
}

func TestChangeConfigPath(t *testing.T) {
	got := ChangeConfigPath("/root", "fix-bug")
	want := filepath.Join("/root", "sdd", ChangesDir, "fix-bug", ChangeConfigFile)
	if got != want {
		t.Errorf("ChangeConfigPath = %s, want %s", got, want)
	}
}

// --- NewFileStore ---

func TestNewFileStore(t *testing.T) {
	store := NewFileStore()
	if store == nil {
		t.Fatal("NewFileStore returned nil")
	}
}

// --- Create ---

func TestCreate_WritesChangeJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("fix-bug", "Fix the bug", TypeFix, SizeSmall)

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify change.json exists.
	configPath := ChangeConfigPath(tmpDir, "fix-bug")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("change.json not created at %s", configPath)
	}

	// Verify valid JSON.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var parsed ChangeRecord
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("change.json is not valid JSON: %v", err)
	}
	if parsed.ID != "fix-bug" {
		t.Errorf("ID = %s, want fix-bug", parsed.ID)
	}
	if parsed.Type != TypeFix {
		t.Errorf("Type = %s, want fix", parsed.Type)
	}
}

func TestCreate_CreatesADRsSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("add-auth", "Add auth", TypeFeature, SizeSmall)

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	adrsDir := filepath.Join(ChangePath(tmpDir, "add-auth"), ADRsDir)
	info, err := os.Stat(adrsDir)
	if err != nil {
		t.Fatalf("adrs/ directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("adrs path is not a directory")
	}
}

func TestCreate_SlugCollisionAppendsNumericSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create first change.
	change1 := testChangeRecord("fix-bug", "Fix bug", TypeFix, SizeSmall)
	if err := store.Create(tmpDir, change1); err != nil {
		t.Fatalf("Create first failed: %v", err)
	}
	if change1.ID != "fix-bug" {
		t.Errorf("first change ID = %s, want fix-bug", change1.ID)
	}

	// Create second change with same slug.
	change2 := testChangeRecord("fix-bug", "Fix bug", TypeFix, SizeSmall)
	if err := store.Create(tmpDir, change2); err != nil {
		t.Fatalf("Create second failed: %v", err)
	}
	if change2.ID != "fix-bug-2" {
		t.Errorf("second change ID = %s, want fix-bug-2", change2.ID)
	}

	// Create third change with same slug.
	change3 := testChangeRecord("fix-bug", "Fix bug", TypeFix, SizeSmall)
	if err := store.Create(tmpDir, change3); err != nil {
		t.Fatalf("Create third failed: %v", err)
	}
	if change3.ID != "fix-bug-3" {
		t.Errorf("third change ID = %s, want fix-bug-3", change3.ID)
	}
}

func TestCreate_CreatesChangesDirectoryIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("new-feat", "New feature", TypeFeature, SizeMedium)

	// sdd/changes/ doesn't exist yet.
	changesDir := ChangesPath(tmpDir)
	if _, err := os.Stat(changesDir); !os.IsNotExist(err) {
		t.Fatal("changes/ directory should not exist yet")
	}

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		t.Fatal("changes/ directory should have been created")
	}
}

// --- Load ---

func TestLoad_ReadsCreatedChange(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	original := testChangeRecord("fix-login", "Fix login crash", TypeFix, SizeMedium)

	if err := store.Create(tmpDir, original); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	loaded, err := store.Load(tmpDir, "fix-login")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ID != original.ID {
		t.Errorf("ID = %s, want %s", loaded.ID, original.ID)
	}
	if loaded.Type != original.Type {
		t.Errorf("Type = %s, want %s", loaded.Type, original.Type)
	}
	if loaded.Size != original.Size {
		t.Errorf("Size = %s, want %s", loaded.Size, original.Size)
	}
	if loaded.Description != original.Description {
		t.Errorf("Description = %s, want %s", loaded.Description, original.Description)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status = %s, want %s", loaded.Status, original.Status)
	}
	if loaded.CurrentStage != original.CurrentStage {
		t.Errorf("CurrentStage = %s, want %s", loaded.CurrentStage, original.CurrentStage)
	}
	if len(loaded.Stages) != len(original.Stages) {
		t.Errorf("Stages count = %d, want %d", len(loaded.Stages), len(original.Stages))
	}
}

func TestLoad_NotFoundReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	_, err := store.Load(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("Load should fail for nonexistent change")
	}
	if !containsStr(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %s", err.Error())
	}
}

func TestLoad_CorruptJSONReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create directory structure manually with corrupt JSON.
	changeDir := ChangePath(tmpDir, "corrupt")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	configPath := ChangeConfigPath(tmpDir, "corrupt")
	if err := os.WriteFile(configPath, []byte("not json {{{"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := store.Load(tmpDir, "corrupt")
	if err == nil {
		t.Fatal("Load should fail on corrupt JSON")
	}
	if !containsStr(err.Error(), "parsing change.json") {
		t.Errorf("error should mention 'parsing change.json', got: %s", err.Error())
	}
}

// --- LoadActive ---

func TestLoadActive_ReturnsActiveChange(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create an active change.
	active := testChangeRecord("active-one", "Active change", TypeFeature, SizeLarge)
	active.Status = StatusActive
	if err := store.Create(tmpDir, active); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create a completed change.
	completed := testChangeRecord("done-one", "Done change", TypeFix, SizeSmall)
	completed.Status = StatusCompleted
	if err := store.Create(tmpDir, completed); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive failed: %v", err)
	}
	if found == nil {
		t.Fatal("LoadActive returned nil, expected active change")
	}
	if found.ID != "active-one" {
		t.Errorf("LoadActive returned ID = %s, want active-one", found.ID)
	}
}

func TestLoadActive_ReturnsNilWhenNoActive(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create only a completed change.
	completed := testChangeRecord("done-one", "Done change", TypeFix, SizeSmall)
	completed.Status = StatusCompleted
	if err := store.Create(tmpDir, completed); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive failed: %v", err)
	}
	if found != nil {
		t.Errorf("LoadActive should return nil when no active change, got ID = %s", found.ID)
	}
}

func TestLoadActive_ReturnsNilWhenNoChangesDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// No changes/ directory at all.
	found, err := store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive should not error on missing dir: %v", err)
	}
	if found != nil {
		t.Error("LoadActive should return nil when changes/ doesn't exist")
	}
}

func TestLoadActive_SkipsNonDirectoryEntries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create changes/ with a stray file (not a directory).
	changesDir := ChangesPath(tmpDir)
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changesDir, "stray-file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	found, err := store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive should not error: %v", err)
	}
	if found != nil {
		t.Error("LoadActive should return nil with only non-directory entries")
	}
}

func TestLoadActive_SkipsUnreadableChanges(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create a directory without change.json inside.
	badDir := filepath.Join(ChangesPath(tmpDir), "broken-change")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	// No change.json → Load will error → LoadActive should skip it.

	found, err := store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive should not error on unreadable change: %v", err)
	}
	if found != nil {
		t.Error("LoadActive should return nil when no valid changes exist")
	}
}

// --- Save ---

func TestSave_UpdatesExistingRecord(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("my-change", "My change", TypeFix, SizeSmall)

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Modify and save.
	change.CurrentStage = StageTasks
	change.Stages[0].Status = "completed"
	if err := store.Save(tmpDir, change); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify.
	loaded, err := store.Load(tmpDir, "my-change")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.CurrentStage != StageTasks {
		t.Errorf("CurrentStage = %s, want tasks", loaded.CurrentStage)
	}
	if loaded.Stages[0].Status != "completed" {
		t.Errorf("first stage status = %s, want completed", loaded.Stages[0].Status)
	}
}

func TestSave_UpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("ts-test", "Timestamp test", TypeFix, SizeSmall)
	change.UpdatedAt = "2020-01-01T00:00:00Z"

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	oldTimestamp := change.UpdatedAt
	if err := store.Save(tmpDir, change); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// UpdatedAt should have been refreshed.
	if change.UpdatedAt == oldTimestamp {
		t.Error("Save should update the UpdatedAt timestamp")
	}
	if change.UpdatedAt == "" {
		t.Error("UpdatedAt should not be empty after Save")
	}
}

func TestSave_WritesValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("json-test", "JSON test", TypeFeature, SizeMedium)

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	change.Description = "Updated description"
	if err := store.Save(tmpDir, change); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(ChangeConfigPath(tmpDir, "json-test"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Saved file is not valid JSON: %v", err)
	}
	if desc, ok := parsed["description"].(string); !ok || desc != "Updated description" {
		t.Errorf("description = %v, want 'Updated description'", parsed["description"])
	}
}

// --- Archive ---

func TestArchive_MovesToHistory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("completed-change", "Completed change", TypeFix, SizeSmall)
	change.Status = StatusCompleted

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it's in changes/.
	srcPath := ChangePath(tmpDir, "completed-change")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Fatal("change should exist in changes/ before archive")
	}

	if err := store.Archive(tmpDir, "completed-change"); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Source should be gone.
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("change should be removed from changes/ after archive")
	}

	// Destination should exist in history/.
	dstPath := filepath.Join(HistoryPath(tmpDir), "completed-change")
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Fatal("change should exist in history/ after archive")
	}

	// Verify change.json in history has archived status.
	dstConfig := filepath.Join(dstPath, ChangeConfigFile)
	data, err := os.ReadFile(dstConfig)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var archived ChangeRecord
	if err := json.Unmarshal(data, &archived); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if archived.Status != StatusArchived {
		t.Errorf("archived status = %s, want archived", archived.Status)
	}
}

func TestArchive_RefusesActiveChange(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("active-change", "Still active", TypeFeature, SizeLarge)
	change.Status = StatusActive

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err := store.Archive(tmpDir, "active-change")
	if err == nil {
		t.Fatal("Archive should refuse active changes")
	}
	if !containsStr(err.Error(), "cannot archive active change") {
		t.Errorf("error should mention 'cannot archive active change', got: %s", err.Error())
	}
}

func TestArchive_RefusesIfAlreadyInHistory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("done-change", "Done", TypeFix, SizeSmall)
	change.Status = StatusCompleted

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Pre-create a directory in history/ with the same name to simulate collision.
	historyDst := filepath.Join(HistoryPath(tmpDir), "done-change")
	if err := os.MkdirAll(historyDst, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	err := store.Archive(tmpDir, "done-change")
	if err == nil {
		t.Fatal("Archive should refuse if change already exists in history")
	}
	if !containsStr(err.Error(), "already exists in history") {
		t.Errorf("error should mention 'already exists in history', got: %s", err.Error())
	}
}

func TestArchive_NotFoundReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	err := store.Archive(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("Archive should fail for nonexistent change")
	}
}

func TestArchive_CreatesHistoryDirIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()
	change := testChangeRecord("archivable", "Archivable", TypeRefactor, SizeSmall)
	change.Status = StatusCompleted

	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// history/ doesn't exist yet.
	historyDir := HistoryPath(tmpDir)
	if _, err := os.Stat(historyDir); !os.IsNotExist(err) {
		t.Fatal("history/ should not exist yet")
	}

	if err := store.Archive(tmpDir, "archivable"); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		t.Fatal("history/ directory should have been created")
	}
}

// --- List ---

func TestList_ReturnsActiveAndCompletedChanges(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create an active change.
	active := testChangeRecord("active-feat", "Active feature", TypeFeature, SizeMedium)
	active.Status = StatusActive
	if err := store.Create(tmpDir, active); err != nil {
		t.Fatalf("Create active failed: %v", err)
	}

	// Create a completed change.
	completed := testChangeRecord("done-fix", "Done fix", TypeFix, SizeSmall)
	completed.Status = StatusCompleted
	if err := store.Create(tmpDir, completed); err != nil {
		t.Fatalf("Create completed failed: %v", err)
	}

	list, err := store.List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("List returned %d changes, want 2", len(list))
	}
}

func TestList_IncludesArchivedChanges(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create and archive a change.
	change := testChangeRecord("archived-one", "Archived", TypeFix, SizeSmall)
	change.Status = StatusCompleted
	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Archive(tmpDir, "archived-one"); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Create an active change.
	active := testChangeRecord("active-two", "Active", TypeFeature, SizeLarge)
	active.Status = StatusActive
	if err := store.Create(tmpDir, active); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	list, err := store.List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("List returned %d changes, want 2 (1 active + 1 archived)", len(list))
	}

	// Verify both are present.
	ids := map[string]bool{}
	for _, c := range list {
		ids[c.ID] = true
	}
	if !ids["active-two"] {
		t.Error("List should include active change")
	}
	if !ids["archived-one"] {
		t.Error("List should include archived change")
	}
}

func TestList_EmptyWhenNoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	list, err := store.List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List returned %d changes, want 0", len(list))
	}
}

func TestList_SkipsNonDirectoriesAndCorruptEntries(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// Create a valid change.
	valid := testChangeRecord("valid-one", "Valid", TypeFix, SizeSmall)
	if err := store.Create(tmpDir, valid); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Add a stray file in changes/.
	changesDir := ChangesPath(tmpDir)
	if err := os.WriteFile(filepath.Join(changesDir, "stray.txt"), []byte("junk"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Add a directory without change.json in changes/.
	if err := os.MkdirAll(filepath.Join(changesDir, "broken"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Add a stray file in history/.
	historyDir := HistoryPath(tmpDir)
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(historyDir, "garbage.txt"), []byte("junk"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Add a directory with corrupt JSON in history/.
	corruptDir := filepath.Join(historyDir, "corrupt-hist")
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, ChangeConfigFile), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	list, err := store.List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should only return the one valid change.
	if len(list) != 1 {
		t.Errorf("List returned %d changes, want 1 (only valid-one)", len(list))
	}
	if len(list) > 0 && list[0].ID != "valid-one" {
		t.Errorf("List[0].ID = %s, want valid-one", list[0].ID)
	}
}

// --- Round-trip: Create → Save → Load → Archive → List ---

func TestFullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore()

	// 1. Create.
	change := testChangeRecord("lifecycle-test", "Full lifecycle", TypeFeature, SizeMedium)
	if err := store.Create(tmpDir, change); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 2. LoadActive finds it.
	active, err := store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive failed: %v", err)
	}
	if active == nil || active.ID != "lifecycle-test" {
		t.Fatalf("LoadActive didn't find active change")
	}

	// 3. Save with updates.
	change.CurrentStage = StageSpec
	change.Stages[0].Status = "completed"
	change.Stages[1].Status = "in_progress"
	if err := store.Save(tmpDir, change); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 4. Load verifies the update.
	reloaded, err := store.Load(tmpDir, "lifecycle-test")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if reloaded.CurrentStage != StageSpec {
		t.Errorf("CurrentStage after save = %s, want spec", reloaded.CurrentStage)
	}

	// 5. Complete and archive.
	change.Status = StatusCompleted
	if err := store.Save(tmpDir, change); err != nil {
		t.Fatalf("Save (complete) failed: %v", err)
	}
	if err := store.Archive(tmpDir, "lifecycle-test"); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// 6. LoadActive returns nil (no active).
	active, err = store.LoadActive(tmpDir)
	if err != nil {
		t.Fatalf("LoadActive after archive failed: %v", err)
	}
	if active != nil {
		t.Error("LoadActive should return nil after archive")
	}

	// 7. List includes the archived change.
	list, err := store.List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List returned %d, want 1", len(list))
	}
	if list[0].Status != StatusArchived {
		t.Errorf("archived change status = %s, want archived", list[0].Status)
	}
}

// --- Store interface compliance ---

func TestFileStore_ImplementsStoreInterface(t *testing.T) {
	// Compile-time check — if this compiles, FileStore satisfies Store.
	var _ Store = (*FileStore)(nil)
}

// --- helpers ---

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
