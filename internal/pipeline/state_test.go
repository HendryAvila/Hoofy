package pipeline

import (
	"testing"
	"time"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
)

func init() {
	// Freeze time for deterministic tests.
	timeNow = func() time.Time {
		return time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	}
}

// --- ClarityThreshold ---

func TestClarityThreshold_GuidedMode(t *testing.T) {
	got := ClarityThreshold(config.ModeGuided)
	if got != ClarityThresholdGuided {
		t.Errorf("ClarityThreshold(guided) = %d, want %d", got, ClarityThresholdGuided)
	}
}

func TestClarityThreshold_ExpertMode(t *testing.T) {
	got := ClarityThreshold(config.ModeExpert)
	if got != ClarityThresholdExpert {
		t.Errorf("ClarityThreshold(expert) = %d, want %d", got, ClarityThresholdExpert)
	}
}

func TestClarityThreshold_UnknownMode_DefaultsToGuided(t *testing.T) {
	got := ClarityThreshold(config.Mode("banana"))
	if got != ClarityThresholdGuided {
		t.Errorf("ClarityThreshold(unknown) = %d, want %d (guided default)", got, ClarityThresholdGuided)
	}
}

// --- StageIndex ---

func TestStageIndex_AllStages(t *testing.T) {
	tests := []struct {
		stage config.Stage
		want  int
	}{
		{config.StageInit, 0},
		{config.StagePropose, 1},
		{config.StageSpecify, 2},
		{config.StageClarify, 3},
		{config.StageDesign, 4},
		{config.StageTasks, 5},
		{config.StageValidate, 6},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			got := StageIndex(tt.stage)
			if got != tt.want {
				t.Errorf("StageIndex(%s) = %d, want %d", tt.stage, got, tt.want)
			}
		})
	}
}

func TestStageIndex_UnknownStage(t *testing.T) {
	got := StageIndex(config.Stage("nonexistent"))
	if got != -1 {
		t.Errorf("StageIndex(nonexistent) = %d, want -1", got)
	}
}

// --- CanAdvance ---

func newTestConfig(stage config.Stage, mode config.Mode, clarityScore int) *config.ProjectConfig {
	status := make(map[config.Stage]config.StageStatus, len(config.StageOrder))
	for _, s := range config.StageOrder {
		status[s] = config.StageStatus{Status: "pending"}
	}
	return &config.ProjectConfig{
		Name:         "test-project",
		Mode:         mode,
		CurrentStage: stage,
		ClarityScore: clarityScore,
		StageStatus:  status,
	}
}

func TestCanAdvance_NormalStage(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	if err := CanAdvance(cfg); err != nil {
		t.Errorf("CanAdvance(propose) should succeed, got: %v", err)
	}
}

func TestCanAdvance_ClarifyGate_GuidedMode_BelowThreshold(t *testing.T) {
	cfg := newTestConfig(config.StageClarify, config.ModeGuided, 69)
	err := CanAdvance(cfg)
	if err == nil {
		t.Fatal("CanAdvance(clarify, score=69, guided) should fail")
	}
	if got := err.Error(); !contains(got, "clarity gate not passed") {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestCanAdvance_ClarifyGate_GuidedMode_AtThreshold(t *testing.T) {
	cfg := newTestConfig(config.StageClarify, config.ModeGuided, 70)
	if err := CanAdvance(cfg); err != nil {
		t.Errorf("CanAdvance(clarify, score=70, guided) should pass, got: %v", err)
	}
}

func TestCanAdvance_ClarifyGate_ExpertMode_BelowThreshold(t *testing.T) {
	cfg := newTestConfig(config.StageClarify, config.ModeExpert, 49)
	err := CanAdvance(cfg)
	if err == nil {
		t.Fatal("CanAdvance(clarify, score=49, expert) should fail")
	}
}

func TestCanAdvance_ClarifyGate_ExpertMode_AtThreshold(t *testing.T) {
	cfg := newTestConfig(config.StageClarify, config.ModeExpert, 50)
	if err := CanAdvance(cfg); err != nil {
		t.Errorf("CanAdvance(clarify, score=50, expert) should pass, got: %v", err)
	}
}

func TestCanAdvance_FinalStage(t *testing.T) {
	cfg := newTestConfig(config.StageValidate, config.ModeGuided, 100)
	err := CanAdvance(cfg)
	if err == nil {
		t.Fatal("CanAdvance(validate) should fail — already at final stage")
	}
	if got := err.Error(); !contains(got, "final stage") {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestCanAdvance_UnknownStage(t *testing.T) {
	cfg := newTestConfig(config.Stage("bogus"), config.ModeGuided, 0)
	err := CanAdvance(cfg)
	if err == nil {
		t.Fatal("CanAdvance(bogus) should fail")
	}
	if got := err.Error(); !contains(got, "unknown stage") {
		t.Errorf("unexpected error: %s", got)
	}
}

// --- Advance ---

func TestAdvance_MovesToNextStage(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	if err := Advance(cfg); err != nil {
		t.Fatalf("Advance(propose) failed: %v", err)
	}
	if cfg.CurrentStage != config.StageSpecify {
		t.Errorf("CurrentStage = %s, want %s", cfg.CurrentStage, config.StageSpecify)
	}
}

func TestAdvance_MarksPreviousCompleted(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	_ = Advance(cfg)

	proposeStatus := cfg.StageStatus[config.StagePropose]
	if proposeStatus.Status != "completed" {
		t.Errorf("propose status = %s, want completed", proposeStatus.Status)
	}
	if proposeStatus.CompletedAt == "" {
		t.Error("propose CompletedAt should be set")
	}
}

func TestAdvance_MarksNextInProgress(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	_ = Advance(cfg)

	specifyStatus := cfg.StageStatus[config.StageSpecify]
	if specifyStatus.Status != "in_progress" {
		t.Errorf("specify status = %s, want in_progress", specifyStatus.Status)
	}
	if specifyStatus.Iterations != 1 {
		t.Errorf("specify iterations = %d, want 1", specifyStatus.Iterations)
	}
}

func TestAdvance_BlockedByClarityGate(t *testing.T) {
	cfg := newTestConfig(config.StageClarify, config.ModeGuided, 30)
	err := Advance(cfg)
	if err == nil {
		t.Fatal("Advance should fail when clarity gate not passed")
	}
	// Stage should NOT have changed.
	if cfg.CurrentStage != config.StageClarify {
		t.Errorf("CurrentStage changed to %s, should still be clarify", cfg.CurrentStage)
	}
}

func TestAdvance_ThroughClarityGate(t *testing.T) {
	cfg := newTestConfig(config.StageClarify, config.ModeGuided, 85)
	if err := Advance(cfg); err != nil {
		t.Fatalf("Advance(clarify, score=85) failed: %v", err)
	}
	if cfg.CurrentStage != config.StageDesign {
		t.Errorf("CurrentStage = %s, want %s", cfg.CurrentStage, config.StageDesign)
	}
}

func TestAdvance_FullPipeline(t *testing.T) {
	cfg := newTestConfig(config.StageInit, config.ModeExpert, 0)

	expected := []config.Stage{
		config.StagePropose,
		config.StageSpecify,
		config.StageClarify,
		config.StageDesign,
		config.StageTasks,
		config.StageValidate,
	}

	for _, want := range expected {
		// Set high clarity score so the gate passes.
		cfg.ClarityScore = 100
		if err := Advance(cfg); err != nil {
			t.Fatalf("Advance to %s failed: %v", want, err)
		}
		if cfg.CurrentStage != want {
			t.Fatalf("CurrentStage = %s, want %s", cfg.CurrentStage, want)
		}
	}

	// Final stage — should not advance further.
	err := Advance(cfg)
	if err == nil {
		t.Fatal("Advance past validate should fail")
	}
}

// --- MarkInProgress ---

func TestMarkInProgress_IncrementsIterations(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)

	MarkInProgress(cfg)
	MarkInProgress(cfg)
	MarkInProgress(cfg)

	st := cfg.StageStatus[config.StagePropose]
	if st.Iterations != 3 {
		t.Errorf("Iterations = %d, want 3", st.Iterations)
	}
	if st.Status != "in_progress" {
		t.Errorf("Status = %s, want in_progress", st.Status)
	}
}

func TestMarkInProgress_PreservesStartedAt(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)

	MarkInProgress(cfg)
	firstStartedAt := cfg.StageStatus[config.StagePropose].StartedAt

	MarkInProgress(cfg)
	secondStartedAt := cfg.StageStatus[config.StagePropose].StartedAt

	if firstStartedAt != secondStartedAt {
		t.Errorf("StartedAt changed across iterations: %s → %s", firstStartedAt, secondStartedAt)
	}
}

// --- IsCompleted ---

func TestIsCompleted_False_WhenPending(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	if IsCompleted(cfg, config.StagePropose) {
		t.Error("propose should not be completed")
	}
}

func TestIsCompleted_True_AfterAdvance(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	_ = Advance(cfg)
	if !IsCompleted(cfg, config.StagePropose) {
		t.Error("propose should be completed after advancing")
	}
}

// --- RequireStage ---

func TestRequireStage_Matches(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	if err := RequireStage(cfg, config.StagePropose); err != nil {
		t.Errorf("RequireStage(propose, propose) should pass, got: %v", err)
	}
}

func TestRequireStage_Mismatch(t *testing.T) {
	cfg := newTestConfig(config.StagePropose, config.ModeGuided, 0)
	err := RequireStage(cfg, config.StageClarify)
	if err == nil {
		t.Fatal("RequireStage(propose, clarify) should fail")
	}
	got := err.Error()
	if !contains(got, "wrong pipeline stage") {
		t.Errorf("unexpected error: %s", got)
	}
	if !contains(got, "propose") || !contains(got, "clarify") {
		t.Errorf("error should mention both stages: %s", got)
	}
}

// --- helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
