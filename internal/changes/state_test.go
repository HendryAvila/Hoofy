package changes

import (
	"testing"
	"time"
)

func init() {
	// Freeze time for deterministic tests.
	timeNow = func() time.Time {
		return time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	}
}

// --- Helper ---

func testActiveChange(ct ChangeType, cs ChangeSize) *ChangeRecord {
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
		ID:           "test-change",
		Type:         ct,
		Size:         cs,
		Description:  "Test change",
		Stages:       stages,
		CurrentStage: flow[0],
		Status:       StatusActive,
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-01T00:00:00Z",
	}
}

// --- CurrentStageIndex ---

func TestCurrentStageIndex_FirstStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe, tasks, verify
	got := CurrentStageIndex(change)
	if got != 0 {
		t.Errorf("CurrentStageIndex = %d, want 0", got)
	}
}

func TestCurrentStageIndex_MiddleStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeMedium) // describe, context-check, spec, tasks, verify
	change.CurrentStage = StageSpec
	got := CurrentStageIndex(change)
	if got != 2 {
		t.Errorf("CurrentStageIndex = %d, want 2", got)
	}
}

func TestCurrentStageIndex_LastStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe, context-check, tasks, verify
	change.CurrentStage = StageVerify
	got := CurrentStageIndex(change)
	if got != 3 {
		t.Errorf("CurrentStageIndex = %d, want 3", got)
	}
}

func TestCurrentStageIndex_UnknownStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.CurrentStage = ChangeStage("bogus")
	got := CurrentStageIndex(change)
	if got != -1 {
		t.Errorf("CurrentStageIndex for unknown stage = %d, want -1", got)
	}
}

// --- IsLastStage ---

func TestIsLastStage_AtVerify(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe, tasks, verify
	change.CurrentStage = StageVerify
	if !IsLastStage(change) {
		t.Error("IsLastStage should be true at verify")
	}
}

func TestIsLastStage_NotAtVerify(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe, tasks, verify
	if IsLastStage(change) {
		t.Error("IsLastStage should be false at describe")
	}
}

func TestIsLastStage_UnknownStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.CurrentStage = ChangeStage("bogus")
	if IsLastStage(change) {
		t.Error("IsLastStage should be false for unknown stage")
	}
}

// --- CanAdvance ---

func TestCanAdvance_NormalStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe, tasks, verify
	if err := CanAdvance(change); err != nil {
		t.Errorf("CanAdvance at first stage should succeed, got: %v", err)
	}
}

func TestCanAdvance_FinalStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.CurrentStage = StageVerify
	err := CanAdvance(change)
	if err == nil {
		t.Fatal("CanAdvance at final stage should fail")
	}
	if !containsStr(err.Error(), "final stage") {
		t.Errorf("error should mention 'final stage', got: %s", err.Error())
	}
}

func TestCanAdvance_NotActive(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.Status = StatusCompleted
	err := CanAdvance(change)
	if err == nil {
		t.Fatal("CanAdvance on completed change should fail")
	}
	if !containsStr(err.Error(), "not active") {
		t.Errorf("error should mention 'not active', got: %s", err.Error())
	}
}

func TestCanAdvance_ArchivedChange(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.Status = StatusArchived
	err := CanAdvance(change)
	if err == nil {
		t.Fatal("CanAdvance on archived change should fail")
	}
	if !containsStr(err.Error(), "not active") {
		t.Errorf("error should mention 'not active', got: %s", err.Error())
	}
}

func TestCanAdvance_UnknownCurrentStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.CurrentStage = ChangeStage("bogus")
	err := CanAdvance(change)
	if err == nil {
		t.Fatal("CanAdvance with unknown current stage should fail")
	}
	if !containsStr(err.Error(), "unknown current stage") {
		t.Errorf("error should mention 'unknown current stage', got: %s", err.Error())
	}
}

// --- Advance ---

func TestAdvance_MovesToNextStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe → context-check → tasks → verify
	if err := Advance(change); err != nil {
		t.Fatalf("Advance failed: %v", err)
	}
	if change.CurrentStage != StageContextCheck {
		t.Errorf("CurrentStage = %s, want context-check", change.CurrentStage)
	}
}

func TestAdvance_MarksPreviousCompleted(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	_ = Advance(change)

	if change.Stages[0].Status != "completed" {
		t.Errorf("first stage status = %s, want completed", change.Stages[0].Status)
	}
	if change.Stages[0].CompletedAt == "" {
		t.Error("first stage CompletedAt should be set")
	}
}

func TestAdvance_MarksNextInProgress(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	_ = Advance(change)

	if change.Stages[1].Status != "in_progress" {
		t.Errorf("second stage status = %s, want in_progress", change.Stages[1].Status)
	}
	if change.Stages[1].StartedAt == "" {
		t.Error("second stage StartedAt should be set")
	}
}

func TestAdvance_UpdatesTimestamp(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	_ = Advance(change)

	if change.UpdatedAt == "2026-01-01T00:00:00Z" {
		t.Error("UpdatedAt should have been updated by Advance")
	}
}

func TestAdvance_DoesNotAutoCompleteAtVerify(t *testing.T) {
	// fix/small has 4 stages: describe → context-check → tasks → verify
	change := testActiveChange(TypeFix, SizeSmall)

	// Advance through describe → context-check.
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to context-check failed: %v", err)
	}
	// Advance through context-check → tasks.
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to tasks failed: %v", err)
	}
	// Advance through tasks → verify.
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to verify failed: %v", err)
	}

	// Should be at verify, still active.
	if change.CurrentStage != StageVerify {
		t.Errorf("CurrentStage = %s, want verify", change.CurrentStage)
	}
	if change.Status != StatusActive {
		t.Errorf("Status = %s, want active (should not auto-complete at verify)", change.Status)
	}
}

func TestAdvance_CannotAdvancePastFinal(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe → context-check → tasks → verify
	// Advance to verify (3 advances for 4-stage flow).
	_ = Advance(change) // → context-check
	_ = Advance(change) // → tasks
	_ = Advance(change) // → verify

	err := Advance(change)
	if err == nil {
		t.Fatal("Advance past verify should fail")
	}
	if !containsStr(err.Error(), "final stage") {
		t.Errorf("error should mention 'final stage', got: %s", err.Error())
	}
}

func TestAdvance_FullFlow_FixSmall(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe → context-check → tasks → verify
	expected := []ChangeStage{StageContextCheck, StageTasks, StageVerify}

	for _, want := range expected {
		if err := Advance(change); err != nil {
			t.Fatalf("Advance to %s failed: %v", want, err)
		}
		if change.CurrentStage != want {
			t.Fatalf("CurrentStage = %s, want %s", change.CurrentStage, want)
		}
	}

	// Still active at verify — need CompleteChange.
	if change.Status != StatusActive {
		t.Errorf("Status = %s, want active at verify", change.Status)
	}
}

func TestAdvance_FullFlow_FeatureLarge(t *testing.T) {
	change := testActiveChange(TypeFeature, SizeLarge)
	// propose → context-check → spec → clarify → design → tasks → verify
	expected := []ChangeStage{StageContextCheck, StageSpec, StageClarify, StageDesign, StageTasks, StageVerify}

	for _, want := range expected {
		if err := Advance(change); err != nil {
			t.Fatalf("Advance to %s failed: %v", want, err)
		}
		if change.CurrentStage != want {
			t.Fatalf("CurrentStage = %s, want %s", change.CurrentStage, want)
		}
	}

	// All stages before verify are completed.
	for i := 0; i < len(change.Stages)-1; i++ {
		if change.Stages[i].Status != "completed" {
			t.Errorf("stage %d (%s) status = %s, want completed", i, change.Stages[i].Name, change.Stages[i].Status)
		}
	}
}

func TestAdvance_FailsOnCompletedChange(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.Status = StatusCompleted

	err := Advance(change)
	if err == nil {
		t.Fatal("Advance on completed change should fail")
	}
}

// --- CompleteChange ---

func TestCompleteChange_AtVerifyStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe → context-check → tasks → verify
	_ = Advance(change)                            // → context-check
	_ = Advance(change)                            // → tasks
	_ = Advance(change)                            // → verify

	if err := CompleteChange(change); err != nil {
		t.Fatalf("CompleteChange failed: %v", err)
	}

	if change.Status != StatusCompleted {
		t.Errorf("Status = %s, want completed", change.Status)
	}

	// Final stage should be marked completed.
	lastIdx := len(change.Stages) - 1
	if change.Stages[lastIdx].Status != "completed" {
		t.Errorf("final stage status = %s, want completed", change.Stages[lastIdx].Status)
	}
	if change.Stages[lastIdx].CompletedAt == "" {
		t.Error("final stage CompletedAt should be set")
	}
}

func TestCompleteChange_NotAtFinalStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe → tasks → verify
	// Still at describe.

	err := CompleteChange(change)
	if err == nil {
		t.Fatal("CompleteChange should fail when not at final stage")
	}
	if !containsStr(err.Error(), "not at the final stage") {
		t.Errorf("error should mention 'not at the final stage', got: %s", err.Error())
	}
}

func TestCompleteChange_NotActive(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.Status = StatusCompleted
	change.CurrentStage = StageVerify

	err := CompleteChange(change)
	if err == nil {
		t.Fatal("CompleteChange should fail on non-active change")
	}
	if !containsStr(err.Error(), "not active") {
		t.Errorf("error should mention 'not active', got: %s", err.Error())
	}
}

func TestCompleteChange_UnknownCurrentStage(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall)
	change.CurrentStage = ChangeStage("bogus")

	err := CompleteChange(change)
	if err == nil {
		t.Fatal("CompleteChange should fail on unknown stage")
	}
	if !containsStr(err.Error(), "unknown current stage") {
		t.Errorf("error should mention 'unknown current stage', got: %s", err.Error())
	}
}

func TestCompleteChange_UpdatesTimestamp(t *testing.T) {
	change := testActiveChange(TypeFix, SizeSmall) // describe → context-check → tasks → verify
	_ = Advance(change)                            // → context-check
	_ = Advance(change)                            // → tasks
	_ = Advance(change)                            // → verify
	change.UpdatedAt = "2020-01-01T00:00:00Z"

	if err := CompleteChange(change); err != nil {
		t.Fatalf("CompleteChange failed: %v", err)
	}
	if change.UpdatedAt == "2020-01-01T00:00:00Z" {
		t.Error("UpdatedAt should have been updated")
	}
}

// --- Full lifecycle: Advance + CompleteChange ---

func TestFullLifecycle_AdvanceThenComplete(t *testing.T) {
	change := testActiveChange(TypeFeature, SizeMedium)
	// propose → context-check → spec → tasks → verify

	// Advance through all stages.
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to context-check failed: %v", err)
	}
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to spec failed: %v", err)
	}
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to tasks failed: %v", err)
	}
	if err := Advance(change); err != nil {
		t.Fatalf("Advance to verify failed: %v", err)
	}

	// At verify, still active.
	if change.Status != StatusActive {
		t.Fatalf("Status = %s, want active at verify", change.Status)
	}

	// Complete.
	if err := CompleteChange(change); err != nil {
		t.Fatalf("CompleteChange failed: %v", err)
	}
	if change.Status != StatusCompleted {
		t.Errorf("Status = %s, want completed", change.Status)
	}

	// All stages completed.
	for i, stage := range change.Stages {
		if stage.Status != "completed" {
			t.Errorf("stage %d (%s) status = %s, want completed", i, stage.Name, stage.Status)
		}
	}

	// Cannot advance further.
	if err := Advance(change); err == nil {
		t.Fatal("Advance on completed change should fail")
	}
}
