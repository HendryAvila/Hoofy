package tools

import (
	"fmt"
	"log"
	"strings"

	"github.com/HendryAvila/Hoofy/internal/changes"
	"github.com/HendryAvila/Hoofy/internal/config"
	"github.com/HendryAvila/Hoofy/internal/memory"
)

// StageObserver is notified when an SDD pipeline stage completes.
// It's an optional dependency — tools work fine with a nil observer.
type StageObserver interface {
	// OnStageComplete is called after a stage artifact has been written
	// to disk and the pipeline has advanced. projectName identifies the
	// SDD project, stage identifies which pipeline stage completed, and
	// content is the rendered artifact (markdown) that was saved.
	OnStageComplete(projectName string, stage config.Stage, content string)
}

// MemoryBridge saves compact SDD stage summaries to the memory store
// using topic_key upserts, so each stage has one evolving observation
// per project. This enables cross-session awareness of SDD pipeline state.
type MemoryBridge struct {
	store *memory.Store
}

// NewMemoryBridge creates a bridge that auto-saves SDD stage completions
// to the memory store. Returns nil if store is nil — callers should
// check before using (or just assign to a StageObserver variable).
func NewMemoryBridge(store *memory.Store) *MemoryBridge {
	if store == nil {
		return nil
	}
	return &MemoryBridge{store: store}
}

// OnStageComplete saves a compact summary of the completed stage to memory.
// Uses topic_key "sdd/{project}/{stage}" for upserts — each stage overwrites
// its previous observation rather than creating duplicates.
//
// This method is best-effort: memory save failures are logged but don't
// propagate errors, because SDD pipeline completion is the primary concern.
func (b *MemoryBridge) OnStageComplete(projectName string, stage config.Stage, content string) {
	topicKey := fmt.Sprintf("sdd/%s/%s", normalizeProject(projectName), stage)
	title := fmt.Sprintf("SDD %s: %s", stage, projectName)

	// Create a compact summary — we don't store the full artifact,
	// just enough for cross-session context retrieval.
	summary := compactSummary(stage, content)

	// Ensure the manual-save session exists for bridge observations.
	_ = b.store.CreateSession("manual-save", projectName, "")

	_, err := b.store.AddObservation(memory.AddObservationParams{
		SessionID: "manual-save",
		Type:      "decision",
		Title:     title,
		Content:   summary,
		Project:   projectName,
		Scope:     "project",
		TopicKey:  topicKey,
	})
	if err != nil {
		log.Printf("WARNING: memory bridge: save %s stage for %q: %v", stage, projectName, err)
	}
}

// notifyObserver is a nil-safe helper called from SDD tool Handle methods.
// If observer is nil, this is a no-op.
func notifyObserver(obs StageObserver, projectName string, stage config.Stage, content string) {
	if obs == nil {
		return
	}
	obs.OnStageComplete(projectName, stage, content)
}

// ChangeObserver is notified when a change pipeline stage completes.
// It's an optional dependency — tools work fine with a nil observer.
type ChangeObserver interface {
	// OnChangeStageComplete is called after a change stage artifact has been
	// written to disk and the pipeline has advanced. changeID identifies the
	// change, stage identifies which stage completed, and content is the
	// rendered artifact (markdown) that was saved.
	OnChangeStageComplete(changeID string, stage changes.ChangeStage, content string)
}

// OnChangeStageComplete saves a compact summary of the completed change stage
// to memory. Uses topic_key "change/{project}/{change-id}/{stage}" for upserts.
//
// Best-effort: memory save failures are logged but don't propagate.
func (b *MemoryBridge) OnChangeStageComplete(changeID string, stage changes.ChangeStage, content string) {
	projectSlug := "unknown"
	// Use changeID as the project context for the topic_key.
	topicKey := fmt.Sprintf("change/%s/%s/%s", projectSlug, changeID, stage)
	title := fmt.Sprintf("Change %s: %s", stage, changeID)

	summary := fmt.Sprintf("**Stage**: %s completed for change `%s`\n\n", stage, changeID)
	const maxLen = 500
	remaining := maxLen - len(summary)
	if remaining > 0 {
		if len(content) <= remaining {
			summary += content
		} else {
			truncated := content[:remaining]
			if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > remaining/2 {
				truncated = truncated[:lastNewline]
			}
			summary += truncated + "\n\n[...truncated]"
		}
	}

	_ = b.store.CreateSession("manual-save", "", "")

	_, err := b.store.AddObservation(memory.AddObservationParams{
		SessionID: "manual-save",
		Type:      "decision",
		Title:     title,
		Content:   summary,
		Scope:     "project",
		TopicKey:  topicKey,
	})
	if err != nil {
		log.Printf("WARNING: change bridge: save %s stage for %q: %v", stage, changeID, err)
	}
}

// notifyChangeObserver is a nil-safe helper called from change tool Handle methods.
// If observer is nil, this is a no-op.
func notifyChangeObserver(obs ChangeObserver, changeID string, stage changes.ChangeStage, content string) {
	if obs == nil {
		return
	}
	obs.OnChangeStageComplete(changeID, stage, content)
}

// normalizeProject converts a project name to a lowercase slug suitable
// for use in topic_key paths (e.g. "My Project" → "my-project").
func normalizeProject(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	// Remove characters that aren't alphanumeric, hyphens, or underscores.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// compactSummary extracts the first ~500 chars of a stage artifact
// as a compact representation for memory storage. For structured stages
// (like requirements), it tries to preserve the key sections.
func compactSummary(stage config.Stage, content string) string {
	const maxLen = 500

	// For all stages, take a meaningful prefix.
	// The topic_key upsert means we'll always have the latest version.
	summary := fmt.Sprintf("**Stage**: %s completed\n\n", stage)

	remaining := maxLen - len(summary)
	if remaining <= 0 {
		return summary
	}

	if len(content) <= remaining {
		return summary + content
	}

	// Truncate at a line boundary if possible.
	truncated := content[:remaining]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > remaining/2 {
		truncated = truncated[:lastNewline]
	}

	return summary + truncated + "\n\n[...truncated]"
}
