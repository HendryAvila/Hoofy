package changes

import "fmt"

// --- State machine for the adaptive change pipeline ---
//
// Unlike the project pipeline (fixed 7-stage order from config.StageOrder),
// the change pipeline reads stage order from ChangeRecord.Stages,
// which is set at creation time based on (ChangeType, ChangeSize).

// CurrentStageIndex returns the ordinal position of the current stage
// within the change's stage list, or -1 if not found.
func CurrentStageIndex(change *ChangeRecord) int {
	for i, entry := range change.Stages {
		if entry.Name == change.CurrentStage {
			return i
		}
	}
	return -1
}

// IsLastStage returns true if the current stage is the final stage (verify).
func IsLastStage(change *ChangeRecord) bool {
	idx := CurrentStageIndex(change)
	return idx >= 0 && idx == len(change.Stages)-1
}

// CanAdvance checks whether the change can move past the current stage.
// Returns an error if advancement is not possible.
func CanAdvance(change *ChangeRecord) error {
	if change.Status != StatusActive {
		return fmt.Errorf("change %q is not active (status: %s)", change.ID, change.Status)
	}

	idx := CurrentStageIndex(change)
	if idx < 0 {
		return fmt.Errorf("unknown current stage %q in change %q", change.CurrentStage, change.ID)
	}

	if idx >= len(change.Stages)-1 {
		return fmt.Errorf("already at the final stage %q in change %q", change.CurrentStage, change.ID)
	}

	return nil
}

// Advance moves the change to the next stage. It validates the transition
// first, marks the current stage completed, and moves to the next.
// When advancing past the final stage, the change status is set to completed.
func Advance(change *ChangeRecord) error {
	if err := CanAdvance(change); err != nil {
		return err
	}

	idx := CurrentStageIndex(change)
	now := timeNow().UTC().Format("2006-01-02T15:04:05Z07:00")

	// Mark current stage completed.
	change.Stages[idx].Status = "completed"
	change.Stages[idx].CompletedAt = now

	nextIdx := idx + 1
	// Mark next stage in_progress.
	change.Stages[nextIdx].Status = "in_progress"
	change.Stages[nextIdx].StartedAt = now

	// Advance current stage pointer.
	change.CurrentStage = change.Stages[nextIdx].Name
	change.UpdatedAt = now

	// If we just moved to the final stage (verify), we don't auto-complete.
	// The verify stage still needs content. Completion happens when
	// verify is advanced via AdvanceComplete.

	return nil
}

// CompleteChange marks the change as completed. Called after the final
// stage (verify) content has been saved.
func CompleteChange(change *ChangeRecord) error {
	if change.Status != StatusActive {
		return fmt.Errorf("change %q is not active (status: %s)", change.ID, change.Status)
	}

	idx := CurrentStageIndex(change)
	if idx < 0 {
		return fmt.Errorf("unknown current stage %q in change %q", change.CurrentStage, change.ID)
	}

	if !IsLastStage(change) {
		return fmt.Errorf("cannot complete change %q: not at the final stage (current: %s)", change.ID, change.CurrentStage)
	}

	now := timeNow().UTC().Format("2006-01-02T15:04:05Z07:00")

	// Mark final stage completed.
	change.Stages[idx].Status = "completed"
	change.Stages[idx].CompletedAt = now

	// Mark change as completed.
	change.Status = StatusCompleted
	change.UpdatedAt = now

	return nil
}
