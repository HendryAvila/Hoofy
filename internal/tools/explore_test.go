package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// newExploreTestStore creates a memory.Store in a temp directory for explore tests.
// It also seeds the "manual-save" session required for FK constraints.
func newExploreTestStore(t *testing.T) *memory.Store {
	t.Helper()
	store, err := memory.New(memory.Config{
		DataDir:              t.TempDir(),
		MaxObservationLength: 2000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	// Seed the default session so FK constraints pass.
	if err := store.CreateSession("manual-save", "", "/tmp/test"); err != nil {
		t.Fatalf("seed manual-save session: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// exploreReq builds a CallToolRequest with the given arguments.
func exploreReq(args map[string]interface{}) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

func TestExploreTool_Definition(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)
	def := tool.Definition()

	if def.Name != "sdd_explore" {
		t.Errorf("tool name = %q, want %q", def.Name, "sdd_explore")
	}

	props := def.InputSchema.Properties
	if len(props) != 10 {
		t.Errorf("parameter count = %d, want 10", len(props))
	}

	// title should be required.
	required := def.InputSchema.Required
	if len(required) != 1 || required[0] != "title" {
		t.Errorf("required = %v, want [title]", required)
	}
}

func TestExploreTool_Handle_BasicSave(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title": "User Auth System",
		"goals": "Implement JWT authentication",
	})

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("unexpected tool error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "User Auth System") {
		t.Error("response should contain the title")
	}
	if !strings.Contains(text, "Created") {
		t.Error("response should indicate 'Created' action")
	}
	if !strings.Contains(text, "## Goals") {
		t.Error("response should contain captured Goals section")
	}
	if !strings.Contains(text, "Implement JWT authentication") {
		t.Error("response should contain goals content")
	}
	if !strings.Contains(text, "explore/") {
		t.Error("response should contain topic key with explore/ prefix")
	}
}

func TestExploreTool_Handle_TitleRequired(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"goals": "something",
	})

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected tool error for missing title")
	}
	if !strings.Contains(getResultText(result), "title") {
		t.Error("error should mention 'title'")
	}
}

func TestExploreTool_Handle_AtLeastOneContentField(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title": "Empty Explore",
	})

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !isErrorResult(result) {
		t.Fatal("expected tool error when no content fields provided")
	}
	if !strings.Contains(getResultText(result), "At least one") {
		t.Error("error should mention 'At least one'")
	}
}

func TestExploreTool_Handle_Upsert(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	// First call — create.
	req1 := exploreReq(map[string]interface{}{
		"title": "My Feature",
		"goals": "Build something cool",
	})
	r1, err := tool.Handle(context.Background(), req1)
	if err != nil {
		t.Fatalf("first Handle: %v", err)
	}
	if isErrorResult(r1) {
		t.Fatalf("first call error: %s", getResultText(r1))
	}
	if !strings.Contains(getResultText(r1), "Created") {
		t.Error("first call should say 'Created'")
	}

	// Second call — upsert (same title).
	req2 := exploreReq(map[string]interface{}{
		"title": "My Feature",
		"goals": "Build something REALLY cool",
	})
	r2, err := tool.Handle(context.Background(), req2)
	if err != nil {
		t.Fatalf("second Handle: %v", err)
	}
	if isErrorResult(r2) {
		t.Fatalf("second call error: %s", getResultText(r2))
	}
	if !strings.Contains(getResultText(r2), "Updated") {
		t.Error("second call should say 'Updated'")
	}
	if !strings.Contains(getResultText(r2), "Build something REALLY cool") {
		t.Error("upsert should contain the new goals content")
	}
}

func TestExploreTool_Handle_UpsertMerge(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	// First call with goals only.
	req1 := exploreReq(map[string]interface{}{
		"title": "Merge Test",
		"goals": "Build a REST API",
	})
	r1, _ := tool.Handle(context.Background(), req1)
	if isErrorResult(r1) {
		t.Fatalf("first call: %s", getResultText(r1))
	}

	// Second call with constraints only — goals should be preserved.
	req2 := exploreReq(map[string]interface{}{
		"title":       "Merge Test",
		"constraints": "Must use Go, no CGO",
	})
	r2, _ := tool.Handle(context.Background(), req2)
	if isErrorResult(r2) {
		t.Fatalf("second call: %s", getResultText(r2))
	}

	text := getResultText(r2)
	if !strings.Contains(text, "Build a REST API") {
		t.Error("merge should preserve existing goals from first call")
	}
	if !strings.Contains(text, "Must use Go, no CGO") {
		t.Error("merge should include new constraints from second call")
	}
}

func TestExploreTool_Handle_UpsertOverride(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	// First call.
	req1 := exploreReq(map[string]interface{}{
		"title": "Override Test",
		"goals": "Original goals",
	})
	tool.Handle(context.Background(), req1)

	// Second call overrides goals.
	req2 := exploreReq(map[string]interface{}{
		"title": "Override Test",
		"goals": "New goals",
	})
	r2, _ := tool.Handle(context.Background(), req2)
	if isErrorResult(r2) {
		t.Fatalf("second call: %s", getResultText(r2))
	}

	text := getResultText(r2)
	if strings.Contains(text, "Original goals") {
		t.Error("override should NOT contain original goals")
	}
	if !strings.Contains(text, "New goals") {
		t.Error("override should contain new goals")
	}
}

func TestExploreTool_Handle_AllFields(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title":       "Full Context",
		"goals":       "Build X",
		"constraints": "Budget limited",
		"preferences": "Prefer Go",
		"unknowns":    "Scale unknown",
		"decisions":   "Use PostgreSQL",
		"context":     "Migrating from legacy",
	})

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("tool error: %s", getResultText(result))
	}

	text := getResultText(result)
	for _, section := range []string{"## Goals", "## Constraints", "## Preferences", "## Unknowns", "## Decisions", "## Context"} {
		if !strings.Contains(text, section) {
			t.Errorf("response missing section %q", section)
		}
	}
	for _, content := range []string{"Build X", "Budget limited", "Prefer Go", "Scale unknown", "Use PostgreSQL", "Migrating from legacy"} {
		if !strings.Contains(text, content) {
			t.Errorf("response missing content %q", content)
		}
	}
}

func TestExploreTool_Handle_TypeSuggestion_Fix(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title": "Login Bug",
		"goals": "Fix the login crash when password is empty",
	})

	result, _ := tool.Handle(context.Background(), req)
	if isErrorResult(result) {
		t.Fatalf("tool error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "**Suggested type:** fix") {
		t.Error("should suggest type=fix for fix/crash keywords")
	}
}

func TestExploreTool_Handle_TypeSuggestion_Feature(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title": "Dark Mode",
		"goals": "Add dark mode toggle to settings",
	})

	result, _ := tool.Handle(context.Background(), req)
	if isErrorResult(result) {
		t.Fatalf("tool error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "**Suggested type:** feature") {
		t.Errorf("should suggest type=feature for 'add' keyword, got: %s", text)
	}
}

func TestExploreTool_Handle_TypeSuggestion_NoSignal(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title":   "Thinking About Things",
		"context": "Just thinking about the system in general",
	})

	result, _ := tool.Handle(context.Background(), req)
	if isErrorResult(result) {
		t.Fatalf("tool error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "no strong signal") {
		t.Error("should indicate 'no strong signal' when no keywords match")
	}
}

func TestExploreTool_Handle_ProjectParam(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title":   "My Feature",
		"goals":   "Build X",
		"project": "my-project",
	})

	result, _ := tool.Handle(context.Background(), req)
	if isErrorResult(result) {
		t.Fatalf("tool error: %s", getResultText(result))
	}

	// Verify observation was saved with the project.
	results, err := store.Search("Build X", memory.SearchOptions{
		Project: "my-project",
		Type:    "explore",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected observation saved with project=my-project")
	}
	if results[0].Project == nil || *results[0].Project != "my-project" {
		t.Error("observation project should be 'my-project'")
	}
}

func TestExploreTool_Handle_ScopeDefault(t *testing.T) {
	store := newExploreTestStore(t)
	tool := NewExploreTool(store)

	req := exploreReq(map[string]interface{}{
		"title": "Scope Test",
		"goals": "Test default scope",
	})

	result, _ := tool.Handle(context.Background(), req)
	if isErrorResult(result) {
		t.Fatalf("tool error: %s", getResultText(result))
	}

	// Verify observation has scope=project (default).
	results, err := store.Search("Test default scope", memory.SearchOptions{
		Scope: "project",
		Type:  "explore",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected observation saved with scope=project")
	}
	if results[0].Scope != "project" {
		t.Errorf("scope = %q, want %q", results[0].Scope, "project")
	}
}

// ─── Unit tests for private helpers ─────────────────────────────────────────

func TestParseExploreContent(t *testing.T) {
	md := "## Goals\n\nBuild something\n\n## Constraints\n\nNo budget\n\n## Decisions\n\nUse Go"
	sections := parseExploreContent(md)

	if sections["Goals"] != "Build something" {
		t.Errorf("Goals = %q, want %q", sections["Goals"], "Build something")
	}
	if sections["Constraints"] != "No budget" {
		t.Errorf("Constraints = %q, want %q", sections["Constraints"], "No budget")
	}
	if sections["Decisions"] != "Use Go" {
		t.Errorf("Decisions = %q, want %q", sections["Decisions"], "Use Go")
	}
}

func TestParseExploreContent_Empty(t *testing.T) {
	sections := parseExploreContent("")
	if len(sections) != 0 {
		t.Errorf("expected empty map for empty input, got %d entries", len(sections))
	}
}

func TestMergeExploreSections(t *testing.T) {
	existing := map[string]string{
		"Goals":       "Original goals",
		"Constraints": "Original constraints",
	}
	incoming := map[string]string{
		"Goals":    "New goals",
		"Unknowns": "Some unknowns",
	}

	merged := mergeExploreSections(existing, incoming)

	if merged["Goals"] != "New goals" {
		t.Error("new non-empty value should override existing")
	}
	if merged["Constraints"] != "Original constraints" {
		t.Error("empty new value should preserve existing")
	}
	if merged["Unknowns"] != "Some unknowns" {
		t.Error("new values should be included")
	}
}

func TestSuggestChangeType_Keywords(t *testing.T) {
	tests := []struct {
		text     string
		wantType string
		wantSize string
	}{
		{"fix the login bug", "fix", "medium"},
		{"crash on startup", "fix", "medium"},
		{"refactor the auth module", "refactor", "medium"},
		{"improve performance of queries", "enhancement", "medium"},
		{"add a new feature", "feature", "medium"},
		{"create user management", "feature", "medium"},
		{"just thinking about stuff", "feature", "medium"}, // default
		{"quick fix for typo", "fix", "small"},
		{"major rewrite of the entire system", "feature", "large"},
		{"simple cleanup task", "refactor", "small"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			gotType, gotSize, _ := suggestChangeType(tt.text)
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
			if gotSize != tt.wantSize {
				t.Errorf("size = %q, want %q", gotSize, tt.wantSize)
			}
		})
	}
}

func TestFormatExploreContent(t *testing.T) {
	sections := map[string]string{
		"Goals":    "Build it",
		"Unknowns": "Not sure about scale",
	}

	out := formatExploreContent(sections)

	if !strings.Contains(out, "## Goals\n\nBuild it") {
		t.Error("should format Goals section")
	}
	if !strings.Contains(out, "## Unknowns\n\nNot sure about scale") {
		t.Error("should format Unknowns section")
	}
	if strings.Contains(out, "## Constraints") {
		t.Error("should not include empty Constraints section")
	}
}
