// Package config handles Hoofy project configuration persistence.
//
// It follows the Single Responsibility Principle: this package ONLY deals with
// the hoofy.json configuration file - reading, writing, and providing type definitions.
// Pipeline logic (transitions, validation) lives in the pipeline package.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DocsDir is the default directory name where Hoofy artifacts live.
	DocsDir = "docs"
	// DocsDirFallback is the subdirectory used when docs/ already exists with non-Hoofy content.
	DocsDirFallback = "specs"
	// ConfigFile is the Hoofy configuration filename.
	ConfigFile = "hoofy.json"
)

// Mode controls how the SDD pipeline interacts with the user.
type Mode string

const (
	// ModeGuided provides step-by-step guidance with more questions,
	// designed for non-technical users and vibe coders.
	ModeGuided Mode = "guided"
	// ModeExpert provides a streamlined experience for technical users
	// who already know what they want.
	ModeExpert Mode = "expert"
)

// Stage represents a discrete phase in the SDD pipeline.
type Stage string

const (
	StageInit          Stage = "init"
	StagePrinciples    Stage = "principles"
	StageCharter       Stage = "charter"
	StageSpecify       Stage = "specify"
	StageBusinessRules Stage = "business-rules"
	StageClarify       Stage = "clarify"
	StageDesign        Stage = "design"
	StageTasks         Stage = "tasks"
	StageValidate      Stage = "validate"
)

// StageOrder defines the sequential pipeline. Used by the pipeline package
// to determine valid transitions.
var StageOrder = []Stage{
	StageInit,
	StagePrinciples,
	StageCharter,
	StageSpecify,
	StageBusinessRules,
	StageClarify,
	StageDesign,
	StageTasks,
	StageValidate,
}

// StageMetadata provides human-readable info about each stage.
type StageMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

// Stages maps each Stage to its metadata.
var Stages = map[Stage]StageMetadata{
	StageInit:          {Name: "Initialize", Description: "Set up project context, constraints, and Hoofy structure", Order: 0},
	StagePrinciples:    {Name: "Principles", Description: "Define golden invariants, coding standards, and domain truths", Order: 1},
	StageCharter:       {Name: "Charter", Description: "Define project scope, vision, stakeholders, and boundaries", Order: 2},
	StageSpecify:       {Name: "Specify", Description: "Extract formal requirements from the charter", Order: 3},
	StageBusinessRules: {Name: "Business Rules", Description: "Extract and document declarative business rules from requirements", Order: 4},
	StageClarify:       {Name: "Clarify", Description: "Detect and resolve ambiguities through the Clarity Gate", Order: 5},
	StageDesign:        {Name: "Design", Description: "Create technical architecture and design decisions", Order: 6},
	StageTasks:         {Name: "Tasks", Description: "Break down design into atomic, actionable tasks", Order: 7},
	StageValidate:      {Name: "Validate", Description: "Verify consistency across all artifacts", Order: 8},
}

// StageStatus tracks progress for a single pipeline stage.
type StageStatus struct {
	Status      string `json:"status"` // pending | in_progress | completed | skipped
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Iterations  int    `json:"iterations"`
}

// ProjectConfig is the root configuration persisted in hoofy.json.
type ProjectConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	Mode         Mode   `json:"mode"`
	CurrentStage Stage  `json:"current_stage"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`

	StageStatus  map[Stage]StageStatus `json:"stage_status"`
	ClarityScore int                   `json:"clarity_score"`
}

// NewProjectConfig creates a config with sensible defaults.
// Init stage is automatically marked as completed.
func NewProjectConfig(name, description string, mode Mode) *ProjectConfig {
	now := time.Now().UTC().Format(time.RFC3339)

	status := make(map[Stage]StageStatus, len(StageOrder))
	for _, s := range StageOrder {
		status[s] = StageStatus{Status: "pending"}
	}
	status[StageInit] = StageStatus{
		Status:      "completed",
		StartedAt:   now,
		CompletedAt: now,
		Iterations:  1,
	}

	return &ProjectConfig{
		Name:         name,
		Description:  description,
		Version:      "0.1.0",
		Mode:         mode,
		CurrentStage: StagePrinciples,
		CreatedAt:    now,
		UpdatedAt:    now,
		StageStatus:  status,
		ClarityScore: 0,
	}
}

// --- Path helpers ---

// ResolveDocsDir determines the docs directory relative to projectRoot.
// Resolution algorithm:
//  1. If docs/hoofy.json exists → "docs"
//  2. If docs/specs/hoofy.json exists → "docs/specs"
//  3. Neither exists → default to "docs" (for new projects)
func ResolveDocsDir(projectRoot string) string {
	primary := filepath.Join(projectRoot, DocsDir, ConfigFile)
	if _, err := os.Stat(primary); err == nil {
		return DocsDir
	}

	fallback := filepath.Join(projectRoot, DocsDir, DocsDirFallback, ConfigFile)
	if _, err := os.Stat(fallback); err == nil {
		return filepath.Join(DocsDir, DocsDirFallback)
	}

	return DocsDir
}

// DocsPath returns the absolute path to the resolved docs directory.
func DocsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ResolveDocsDir(projectRoot))
}

// ConfigPath returns the absolute path to hoofy.json.
func ConfigPath(projectRoot string) string {
	return filepath.Join(DocsPath(projectRoot), ConfigFile)
}

// StagePath returns the absolute path to a stage's markdown artifact.
func StagePath(projectRoot string, stage Stage) string {
	filename := stageFilenames[stage]
	if filename == "" {
		return ""
	}
	return filepath.Join(DocsPath(projectRoot), filename)
}

// ADRsPath returns the absolute path to the central ADRs directory.
func ADRsPath(projectRoot string) string {
	return filepath.Join(DocsPath(projectRoot), "adrs")
}

// StageFilename returns the output filename for a stage, or empty if none.
func StageFilename(stage Stage) string {
	return stageFilenames[stage]
}

// stageFilenames maps stages to their output filenames.
var stageFilenames = map[Stage]string{
	StagePrinciples:    "principles.md",
	StageCharter:       "charter.md",
	StageSpecify:       "requirements.md",
	StageBusinessRules: "business-rules.md",
	StageClarify:       "clarifications.md",
	StageDesign:        "design.md",
	StageTasks:         "tasks.md",
	StageValidate:      "validation.md",
}

// --- Persistence (Open/Closed: extend via interfaces, not modification) ---

// Loader reads project configuration. Abstracted for testability.
type Loader interface {
	Load(projectRoot string) (*ProjectConfig, error)
}

// Saver writes project configuration. Abstracted for testability.
type Saver interface {
	Save(projectRoot string, cfg *ProjectConfig) error
}

// Store combines loading and saving. This is the primary interface
// that tools depend on (Interface Segregation: tools that only read
// depend on Loader, not the full Store).
type Store interface {
	Loader
	Saver
}

// FileStore implements Store using the local filesystem.
type FileStore struct{}

// NewFileStore creates a filesystem-backed config store.
func NewFileStore() *FileStore {
	return &FileStore{}
}

// Load reads and parses hoofy.json from disk.
func (fs *FileStore) Load(projectRoot string) (*ProjectConfig, error) {
	path := ConfigPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("hoofy project not initialized — run sdd_init_project first")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing hoofy.json: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to hoofy.json, creating directories as needed.
func (fs *FileStore) Save(projectRoot string, cfg *ProjectConfig) error {
	cfg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	dir := filepath.Dir(ConfigPath(projectRoot))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating docs directory: %w", err)
	}

	return os.WriteFile(ConfigPath(projectRoot), data, 0o644)
}

// Exists checks whether a Hoofy project is initialized at the given root.
func Exists(projectRoot string) bool {
	_, err := os.Stat(ConfigPath(projectRoot))
	return err == nil
}
