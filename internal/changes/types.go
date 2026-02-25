// Package changes handles the adaptive change pipeline for ongoing development.
//
// While the project pipeline (config + pipeline packages) handles full greenfield
// projects with a fixed 7-stage sequence, the change pipeline supports lightweight
// workflows for features, fixes, refactors, and enhancements — each with a
// stage flow determined by change type and size.
//
// This package follows the same design principles as the project pipeline:
// - SRP: types, flows, store, and state machine in separate files
// - DIP: Store is an interface; tools depend on the abstraction
// - OCP: new change types/sizes can be added without modifying existing flows
package changes

import (
	"fmt"
	"strings"
)

// --- Change type enum ---

// ChangeType categorizes what kind of work a change represents.
type ChangeType string

const (
	TypeFeature     ChangeType = "feature"
	TypeFix         ChangeType = "fix"
	TypeRefactor    ChangeType = "refactor"
	TypeEnhancement ChangeType = "enhancement"
)

// validTypes is the set of allowed change types.
var validTypes = map[ChangeType]bool{
	TypeFeature:     true,
	TypeFix:         true,
	TypeRefactor:    true,
	TypeEnhancement: true,
}

// ValidateType returns an error if the type is not recognized.
func ValidateType(t ChangeType) error {
	if !validTypes[t] {
		return fmt.Errorf("invalid change type %q: must be one of: feature, fix, refactor, enhancement", t)
	}
	return nil
}

// --- Change size enum ---

// ChangeSize controls the number of pipeline stages for a change.
type ChangeSize string

const (
	SizeSmall  ChangeSize = "small"
	SizeMedium ChangeSize = "medium"
	SizeLarge  ChangeSize = "large"
)

// validSizes is the set of allowed change sizes.
var validSizes = map[ChangeSize]bool{
	SizeSmall:  true,
	SizeMedium: true,
	SizeLarge:  true,
}

// ValidateSize returns an error if the size is not recognized.
func ValidateSize(s ChangeSize) error {
	if !validSizes[s] {
		return fmt.Errorf("invalid change size %q: must be one of: small, medium, large", s)
	}
	return nil
}

// --- Change stage enum ---

// ChangeStage represents a discrete phase in a change's pipeline.
// Not all changes go through all stages — the flow is determined
// by (ChangeType, ChangeSize).
type ChangeStage string

const (
	StageDescribe     ChangeStage = "describe"      // lightweight: what's the change?
	StageScope        ChangeStage = "scope"         // refactor-specific: what changes, what doesn't
	StagePropose      ChangeStage = "propose"       // full proposal
	StageContextCheck ChangeStage = "context-check" // scan existing specs/rules for conflicts
	StageSpec         ChangeStage = "spec"          // requirements/spec for the change
	StageClarify      ChangeStage = "clarify"       // ambiguity resolution (only for large changes)
	StageDesign       ChangeStage = "design"        // technical design
	StageTasks        ChangeStage = "tasks"         // implementation task breakdown
	StageVerify       ChangeStage = "verify"        // final validation
)

// --- Change status enum ---

// ChangeStatus tracks the overall lifecycle of a change.
type ChangeStatus string

const (
	StatusActive    ChangeStatus = "active"
	StatusCompleted ChangeStatus = "completed"
	StatusArchived  ChangeStatus = "archived"
)

// --- Core data structures ---

// StageEntry tracks progress for a single stage within a change.
type StageEntry struct {
	Name        ChangeStage `json:"name"`
	Status      string      `json:"status"` // pending | in_progress | completed
	StartedAt   string      `json:"started_at,omitempty"`
	CompletedAt string      `json:"completed_at,omitempty"`
}

// ChangeRecord is the root data structure for a change, persisted as change.json.
type ChangeRecord struct {
	ID           string       `json:"id"`
	Type         ChangeType   `json:"type"`
	Size         ChangeSize   `json:"size"`
	Description  string       `json:"description"`
	Stages       []StageEntry `json:"stages"`
	CurrentStage ChangeStage  `json:"current_stage"`
	ADRs         []string     `json:"adrs"`
	Status       ChangeStatus `json:"status"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
}

// ADR represents an Architecture Decision Record captured during a change.
type ADR struct {
	ID                   string `json:"id"`                              // "ADR-001"
	Title                string `json:"title"`                           // "Use PostgreSQL over MongoDB"
	Context              string `json:"context"`                         // problem context
	Decision             string `json:"decision"`                        // what was decided
	Rationale            string `json:"rationale"`                       // why
	AlternativesRejected string `json:"alternatives_rejected,omitempty"` // what else was considered
	Status               string `json:"status"`                          // proposed | accepted | deprecated | superseded
	ChangeID             string `json:"change_id,omitempty"`             // linked change, if any
	CreatedAt            string `json:"created_at"`                      // RFC3339
}

// --- Slug generation ---

const maxSlugLen = 50

// Slugify converts a description string into a URL/filesystem-safe slug.
// Example: "Fix FTS5 empty query crash" → "fix-fts5-empty-query-crash"
//
// Rules:
//   - Lowercase
//   - Spaces and underscores become hyphens
//   - Non-alphanumeric characters (except hyphens) are removed
//   - Consecutive hyphens are collapsed
//   - Leading/trailing hyphens are trimmed
//   - Truncated to 50 characters (at a word boundary if possible)
//   - Empty input returns "unnamed-change"
func Slugify(description string) string {
	if strings.TrimSpace(description) == "" {
		return "unnamed-change"
	}

	s := strings.ToLower(strings.TrimSpace(description))

	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '_' || r == '-':
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")

	if slug == "" {
		return "unnamed-change"
	}

	if len(slug) <= maxSlugLen {
		return slug
	}

	// Truncate at word boundary if possible.
	truncated := slug[:maxSlugLen]
	if lastHyphen := strings.LastIndex(truncated, "-"); lastHyphen > maxSlugLen/2 {
		truncated = truncated[:lastHyphen]
	}

	return strings.TrimRight(truncated, "-")
}
