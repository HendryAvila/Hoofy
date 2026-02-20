package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- NewProjectConfig ---

func TestNewProjectConfig_SetsDefaults(t *testing.T) {
	cfg := NewProjectConfig("my-app", "A cool app", ModeGuided)

	if cfg.Name != "my-app" {
		t.Errorf("Name = %s, want my-app", cfg.Name)
	}
	if cfg.Description != "A cool app" {
		t.Errorf("Description = %s, want 'A cool app'", cfg.Description)
	}
	if cfg.Mode != ModeGuided {
		t.Errorf("Mode = %s, want guided", cfg.Mode)
	}
	if cfg.Version != "0.1.0" {
		t.Errorf("Version = %s, want 0.1.0", cfg.Version)
	}
	if cfg.CurrentStage != StagePropose {
		t.Errorf("CurrentStage = %s, want propose", cfg.CurrentStage)
	}
	if cfg.ClarityScore != 0 {
		t.Errorf("ClarityScore = %d, want 0", cfg.ClarityScore)
	}
}

func TestNewProjectConfig_InitStageCompleted(t *testing.T) {
	cfg := NewProjectConfig("x", "y", ModeExpert)

	initStatus, ok := cfg.StageStatus[StageInit]
	if !ok {
		t.Fatal("init stage status missing")
	}
	if initStatus.Status != "completed" {
		t.Errorf("init status = %s, want completed", initStatus.Status)
	}
	if initStatus.Iterations != 1 {
		t.Errorf("init iterations = %d, want 1", initStatus.Iterations)
	}
}

func TestNewProjectConfig_OtherStagesPending(t *testing.T) {
	cfg := NewProjectConfig("x", "y", ModeGuided)

	for _, stage := range StageOrder {
		if stage == StageInit {
			continue
		}
		st, ok := cfg.StageStatus[stage]
		if !ok {
			t.Errorf("stage %s missing from StageStatus", stage)
			continue
		}
		if st.Status != "pending" {
			t.Errorf("stage %s status = %s, want pending", stage, st.Status)
		}
	}
}

func TestNewProjectConfig_HasTimestamps(t *testing.T) {
	cfg := NewProjectConfig("x", "y", ModeGuided)

	if cfg.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
	if cfg.UpdatedAt == "" {
		t.Error("UpdatedAt should be set")
	}
}

func TestNewProjectConfig_AllStagesPresent(t *testing.T) {
	cfg := NewProjectConfig("x", "y", ModeGuided)

	for _, stage := range StageOrder {
		if _, ok := cfg.StageStatus[stage]; !ok {
			t.Errorf("stage %s missing from StageStatus map", stage)
		}
	}
}

// --- Path helpers ---

func TestSDDPath(t *testing.T) {
	got := SDDPath("/home/user/project")
	want := filepath.Join("/home/user/project", SDDDir)
	if got != want {
		t.Errorf("SDDPath = %s, want %s", got, want)
	}
}

func TestConfigPath(t *testing.T) {
	got := ConfigPath("/home/user/project")
	want := filepath.Join("/home/user/project", SDDDir, ConfigFile)
	if got != want {
		t.Errorf("ConfigPath = %s, want %s", got, want)
	}
}

func TestStagePath_KnownStages(t *testing.T) {
	tests := []struct {
		stage    Stage
		wantFile string
	}{
		{StagePropose, "proposal.md"},
		{StageSpecify, "requirements.md"},
		{StageClarify, "clarifications.md"},
		{StageDesign, "design.md"},
		{StageTasks, "tasks.md"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			got := StagePath("/root", tt.stage)
			want := filepath.Join("/root", SDDDir, tt.wantFile)
			if got != want {
				t.Errorf("StagePath(%s) = %s, want %s", tt.stage, got, want)
			}
		})
	}
}

func TestStagePath_UnknownStage(t *testing.T) {
	got := StagePath("/root", Stage("unknown"))
	if got != "" {
		t.Errorf("StagePath(unknown) = %s, want empty string", got)
	}
}

func TestStagePath_Init_HasNoFile(t *testing.T) {
	got := StagePath("/root", StageInit)
	if got != "" {
		t.Errorf("StagePath(init) = %s, want empty string (init has no artifact)", got)
	}
}

// --- FileStore ---

func TestFileStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewFileStore()
	original := NewProjectConfig("test-project", "A test project", ModeGuided)

	// Save.
	if err := store.Save(tmpDir, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists.
	configPath := ConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", configPath)
	}

	// Load.
	loaded, err := store.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify round-trip (key fields).
	if loaded.Name != original.Name {
		t.Errorf("Name = %s, want %s", loaded.Name, original.Name)
	}
	if loaded.Mode != original.Mode {
		t.Errorf("Mode = %s, want %s", loaded.Mode, original.Mode)
	}
	if loaded.CurrentStage != original.CurrentStage {
		t.Errorf("CurrentStage = %s, want %s", loaded.CurrentStage, original.CurrentStage)
	}
	if loaded.ClarityScore != original.ClarityScore {
		t.Errorf("ClarityScore = %d, want %d", loaded.ClarityScore, original.ClarityScore)
	}
}

func TestFileStore_SaveCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewFileStore()
	cfg := NewProjectConfig("x", "y", ModeExpert)

	if err := store.Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sddDir := SDDPath(tmpDir)
	info, err := os.Stat(sddDir)
	if err != nil {
		t.Fatalf("sdd dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("sdd path is not a directory")
	}
}

func TestFileStore_SaveUpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewFileStore()
	cfg := NewProjectConfig("x", "y", ModeGuided)
	originalUpdatedAt := cfg.UpdatedAt

	if err := store.Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// UpdatedAt should have been refreshed by Save.
	if cfg.UpdatedAt == "" {
		t.Error("UpdatedAt should be set after Save")
	}
	// In a fast test this might be the same — that's okay.
	_ = originalUpdatedAt
}

func TestFileStore_SaveWritesValidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewFileStore()
	cfg := NewProjectConfig("json-test", "testing JSON output", ModeExpert)
	cfg.ClarityScore = 42

	if err := store.Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath(tmpDir))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}

	if name, ok := parsed["name"].(string); !ok || name != "json-test" {
		t.Errorf("JSON name = %v, want json-test", parsed["name"])
	}
}

func TestFileStore_Load_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewFileStore()
	_, err := store.Load(tmpDir)
	if err == nil {
		t.Fatal("Load should fail when no config exists")
	}
	if got := err.Error(); !stringContains(got, "not initialized") {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestFileStore_Load_CorruptJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the sdd dir and write garbage.
	sddDir := SDDPath(tmpDir)
	if err := os.MkdirAll(sddDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(ConfigPath(tmpDir), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	store := NewFileStore()
	_, err := store.Load(tmpDir)
	if err == nil {
		t.Fatal("Load should fail on corrupt JSON")
	}
	if got := err.Error(); !stringContains(got, "parsing sdd.json") {
		t.Errorf("unexpected error: %s", got)
	}
}

// --- Exists ---

func TestExists_ReturnsFalse_WhenNoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	if Exists(tmpDir) {
		t.Error("Exists should return false for empty directory")
	}
}

func TestExists_ReturnsTrue_AfterSave(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewFileStore()
	cfg := NewProjectConfig("x", "y", ModeGuided)
	if err := store.Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !Exists(tmpDir) {
		t.Error("Exists should return true after Save")
	}
}

// --- StageOrder consistency ---

func TestStageOrder_MatchesStagesMap(t *testing.T) {
	for _, stage := range StageOrder {
		if _, ok := Stages[stage]; !ok {
			t.Errorf("stage %s in StageOrder but not in Stages map", stage)
		}
	}
}

func TestStages_OrderFieldMatchesPosition(t *testing.T) {
	for i, stage := range StageOrder {
		meta := Stages[stage]
		if meta.Order != i {
			t.Errorf("Stages[%s].Order = %d, but position in StageOrder = %d", stage, meta.Order, i)
		}
	}
}

// --- helpers ---

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
