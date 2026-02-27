package templates

import (
	"strings"
	"testing"
)

// --- NewRenderer ---

func TestNewRenderer_Succeeds(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() failed: %v", err)
	}
	if r == nil {
		t.Fatal("NewRenderer() returned nil")
	}
}

// --- Render: Proposal ---

func TestRender_Proposal(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	data := ProposalData{
		Name:             "Test Project",
		ProblemStatement: "Users struggle with X",
		TargetUsers:      "Developers and designers",
		ProposedSolution: "Build a tool that does Y",
		OutOfScope:       "Mobile support, offline mode",
		SuccessCriteria:  "50% reduction in time spent",
		OpenQuestions:    "What about edge case Z?",
	}

	result, err := r.Render(Proposal, data)
	if err != nil {
		t.Fatalf("Render(Proposal) failed: %v", err)
	}

	// Verify key sections are present.
	checks := []string{
		"# Test Project — Proposal",
		"## Problem Statement",
		"Users struggle with X",
		"## Target Users",
		"Developers and designers",
		"## Proposed Solution",
		"Build a tool that does Y",
		"## Out of Scope",
		"Mobile support, offline mode",
		"## Success Criteria",
		"50% reduction in time spent",
		"## Open Questions",
		"What about edge case Z?",
		"SDD-Hoffy", // Attribution link.
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Proposal output missing: %q", check)
		}
	}
}

// --- Render: Requirements ---

func TestRender_Requirements(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	data := RequirementsData{
		Name:          "Test Project",
		MustHave:      "- User authentication\n- Dashboard",
		ShouldHave:    "- Email notifications",
		CouldHave:     "- Dark mode",
		WontHave:      "- Mobile app",
		NonFunctional: "- Response time < 200ms",
		Constraints:   "- Must use PostgreSQL",
		Assumptions:   "- Users have modern browsers",
		Dependencies:  "- Auth0 for authentication",
	}

	result, err := r.Render(Requirements, data)
	if err != nil {
		t.Fatalf("Render(Requirements) failed: %v", err)
	}

	checks := []string{
		"# Test Project — Requirements",
		"### Must Have",
		"User authentication",
		"### Should Have",
		"Email notifications",
		"### Could Have",
		"Dark mode",
		"### Won't Have",
		"Mobile app",
		"## Non-Functional Requirements",
		"Response time < 200ms",
		"## Constraints",
		"Must use PostgreSQL",
		"## Assumptions",
		"Users have modern browsers",
		"## Dependencies",
		"Auth0 for authentication",
		"SDD-Hoffy", // Attribution link.
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Requirements output missing: %q", check)
		}
	}
}

// --- Render: Clarifications ---

func TestRender_Clarifications(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	data := ClarificationsData{
		Name:         "Test Project",
		ClarityScore: 75,
		Mode:         "guided",
		Threshold:    70,
		Status:       "PASSED",
		Rounds:       "### Round 1\n\nQ: Who are the users?\nA: Developers",
	}

	result, err := r.Render(Clarifications, data)
	if err != nil {
		t.Fatalf("Render(Clarifications) failed: %v", err)
	}

	checks := []string{
		"# Test Project — Clarifications",
		"Clarity Score: 75/100",
		"guided",
		"Threshold:** 70",
		"PASSED",
		"Round 1",
		"Who are the users?",
		"Developers",
		"Clarity Gate",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Clarifications output missing: %q", check)
		}
	}
}

// --- Render: Unknown template ---

func TestRender_UnknownTemplate(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	_, err = r.Render("nonexistent.md.tmpl", nil)
	if err == nil {
		t.Fatal("Render(nonexistent) should fail")
	}
}

// --- Render: Empty data ---

func TestRender_EmptyProposalData(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	// Should render without error even with zero values.
	result, err := r.Render(Proposal, ProposalData{})
	if err != nil {
		t.Fatalf("Render(Proposal, empty) failed: %v", err)
	}

	// Structure should still be present.
	if !strings.Contains(result, "## Problem Statement") {
		t.Error("empty proposal should still contain section headers")
	}
}

// --- Render: Tasks ---

func TestRender_Tasks_WithWaveAssignments(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	data := TasksData{
		Name:               "Test Project",
		TotalTasks:         "5",
		EstimatedEffort:    "3-4 days",
		Tasks:              "### TASK-001: Scaffolding\n**Component**: Setup",
		DependencyGraph:    "TASK-001 → TASK-002",
		WaveAssignments:    "**Wave 1**:\n- TASK-001: Scaffolding\n\n**Wave 2**:\n- TASK-002: API endpoints",
		AcceptanceCriteria: "- All tests pass",
	}

	result, err := r.Render(Tasks, data)
	if err != nil {
		t.Fatalf("Render(Tasks) failed: %v", err)
	}

	checks := []string{
		"# Test Project — Implementation Tasks",
		"**Total Tasks:** 5",
		"**Estimated Effort:** 3-4 days",
		"TASK-001",
		"TASK-001 → TASK-002",
		"## Execution Waves",
		"in parallel",
		"**Wave 1**",
		"**Wave 2**",
		"## Acceptance Criteria",
		"All tests pass",
		"SDD-Hoffy",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Tasks output missing: %q", check)
		}
	}
}

func TestRender_Tasks_WithoutWaveAssignments(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	data := TasksData{
		Name:               "Test Project",
		TotalTasks:         "3",
		EstimatedEffort:    "2 days",
		Tasks:              "### TASK-001: Scaffolding",
		DependencyGraph:    "TASK-001 → TASK-002",
		WaveAssignments:    "", // empty — should NOT render wave section
		AcceptanceCriteria: "- All tests pass",
	}

	result, err := r.Render(Tasks, data)
	if err != nil {
		t.Fatalf("Render(Tasks) failed: %v", err)
	}

	// Wave section must NOT be present.
	if strings.Contains(result, "## Execution Waves") {
		t.Error("Execution Waves section should NOT render when WaveAssignments is empty")
	}
	if strings.Contains(result, "in parallel") {
		t.Error("wave blockquote should NOT render when WaveAssignments is empty")
	}

	// Other sections must still be present (backwards compatibility).
	checks := []string{
		"# Test Project — Implementation Tasks",
		"TASK-001",
		"## Dependency Graph",
		"## Acceptance Criteria",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Tasks output missing: %q", check)
		}
	}
}

// --- Render: Design ---

func TestRender_Design_WithQualityAnalysis(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	data := DesignData{
		Name:                 "Test Project",
		ArchitectureOverview: "A modular monolith using Clean Architecture",
		TechStack:            "- **Runtime**: Go 1.25",
		Components:           "### AuthModule\n- Handles user auth",
		APIContracts:         "POST /auth/login",
		DataModel:            "### User\n| id | UUID |",
		Infrastructure:       "Docker + Railway",
		Security:             "JWT with refresh tokens",
		DesignDecisions:      "### ADR-001: Go over Node.js",
		QualityAnalysis:      "### SOLID Compliance\n- SRP: AuthModule has single responsibility\n\n### Potential Code Smells\n- No Shotgun Surgery detected\n\n### Coupling & Cohesion\n- Low coupling between modules\n\n### Mitigations\n- DIP via interface injection",
	}

	result, err := r.Render(Design, data)
	if err != nil {
		t.Fatalf("Render(Design) failed: %v", err)
	}

	checks := []string{
		"# Test Project — Technical Design",
		"## Architecture Overview",
		"Clean Architecture",
		"## Tech Stack",
		"Go 1.25",
		"## Components",
		"AuthModule",
		"## API Contracts",
		"POST /auth/login",
		"## Data Model",
		"User",
		"## Infrastructure & Deployment",
		"Docker + Railway",
		"## Security Considerations",
		"JWT with refresh tokens",
		"## Design Decisions",
		"ADR-001",
		"## Structural Quality Analysis",
		"SOLID Compliance",
		"Shotgun Surgery",
		"Coupling & Cohesion",
		"Mitigations",
		"SDD-Hoffy", // Attribution link.
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Design output missing: %q", check)
		}
	}
}

func TestRender_Design_WithoutQualityAnalysis(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	// QualityAnalysis is empty — section header should still render.
	data := DesignData{
		Name:                 "Test Project",
		ArchitectureOverview: "Microservices",
		TechStack:            "Node.js",
		Components:           "API Gateway",
		DataModel:            "Users table",
	}

	result, err := r.Render(Design, data)
	if err != nil {
		t.Fatalf("Render(Design) failed: %v", err)
	}

	// Section header should still be present even with empty content.
	if !strings.Contains(result, "## Structural Quality Analysis") {
		t.Error("Design output should contain Structural Quality Analysis header even when empty")
	}
}

// --- Renderer interface compliance ---

func TestEmbedRenderer_ImplementsRenderer(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	// Compile-time interface check.
	var _ Renderer = r
}
