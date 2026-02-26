package memtools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// ─── Test helpers ────────────────────────────────────────────────────────────

// newTestStore creates a memory.Store in a temp directory for testing.
func newTestStore(t *testing.T) *memory.Store {
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
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// makeReq builds a mcp.CallToolRequest with the given arguments.
func makeReq(args map[string]interface{}) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

// resultText extracts the text content from a tool result.
func resultText(r *mcp.CallToolResult) string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

// ─── ProgressTool Tests ──────────────────────────────────────────────────────

func TestProgressTool_Definition(t *testing.T) {
	store := newTestStore(t)
	tool := NewProgressTool(store)
	def := tool.Definition()

	if def.Name != "mem_progress" {
		t.Errorf("tool name = %q, want %q", def.Name, "mem_progress")
	}

	props := def.InputSchema.Properties
	if _, ok := props["project"]; !ok {
		t.Error("missing 'project' parameter")
	}
	if _, ok := props["content"]; !ok {
		t.Error("missing 'content' parameter")
	}
	if _, ok := props["session_id"]; !ok {
		t.Error("missing 'session_id' parameter")
	}

	// project should be required
	required := def.InputSchema.Required
	found := false
	for _, r := range required {
		if r == "project" {
			found = true
		}
	}
	if !found {
		t.Error("'project' should be required")
	}
}

func TestProgressTool_ReadEmpty(t *testing.T) {
	store := newTestStore(t)
	tool := NewProgressTool(store)

	result, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "test-project",
	}))
	mustNotError(t, result, err)

	text := resultText(result)
	if !strings.Contains(text, "No progress document found") {
		t.Errorf("expected 'no progress' message, got: %s", text)
	}
	if !strings.Contains(text, "test-project") {
		t.Error("response should include project name")
	}
}

func TestProgressTool_WriteAndRead(t *testing.T) {
	store := newTestStore(t)
	seedManualSession(t, store)
	tool := NewProgressTool(store)

	progressJSON := `{"goal":"Implement F2","completed":["TASK-001"],"next_steps":["TASK-002"],"blockers":[]}`

	// Write
	result, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "hoofy",
		"content": progressJSON,
	}))
	mustNotError(t, result, err)

	text := resultText(result)
	if !strings.Contains(text, "Progress updated") {
		t.Errorf("expected 'Progress updated', got: %s", text)
	}
	if !strings.Contains(text, "hoofy") {
		t.Error("response should include project name")
	}

	// Read it back
	result, err = tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "hoofy",
	}))
	mustNotError(t, result, err)

	text = resultText(result)
	if !strings.Contains(text, "Implement F2") {
		t.Errorf("read should return saved progress content, got: %s", text)
	}
	if !strings.Contains(text, "# Progress: hoofy") {
		t.Error("read should include progress header")
	}
	if !strings.Contains(text, "Revisions:") {
		t.Error("read should include metadata footer")
	}
}

func TestProgressTool_InvalidJSON(t *testing.T) {
	store := newTestStore(t)
	tool := NewProgressTool(store)

	result, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "hoofy",
		"content": "this is not JSON {{{",
	}))
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for invalid JSON")
	}

	text := resultText(result)
	if !strings.Contains(text, "valid JSON") {
		t.Errorf("error should mention JSON validity, got: %s", text)
	}
}

func TestProgressTool_MissingProject(t *testing.T) {
	store := newTestStore(t)
	tool := NewProgressTool(store)

	result, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{}))
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for missing project")
	}

	text := resultText(result)
	if !strings.Contains(text, "project") {
		t.Errorf("error should mention 'project', got: %s", text)
	}
}

func TestProgressTool_Upsert(t *testing.T) {
	store := newTestStore(t)
	seedManualSession(t, store)
	tool := NewProgressTool(store)

	// First write
	result1, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "hoofy",
		"content": `{"goal":"First goal","completed":[],"next_steps":["start"],"blockers":[]}`,
	}))
	mustNotError(t, result1, err)

	// Second write — should upsert (same observation, updated content)
	result2, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "hoofy",
		"content": `{"goal":"Updated goal","completed":["start"],"next_steps":["finish"],"blockers":[]}`,
	}))
	mustNotError(t, result2, err)

	// Read — should return the UPDATED content, not the first
	result, err := tool.Handle(context.Background(), makeReq(map[string]interface{}{
		"project": "hoofy",
	}))
	mustNotError(t, result, err)

	text := resultText(result)
	if !strings.Contains(text, "Updated goal") {
		t.Errorf("upsert should show updated content, got: %s", text)
	}
	if strings.Contains(text, "First goal") {
		t.Error("upsert should have replaced the first progress doc")
	}
}

// mustNotError asserts the Handle call returns no Go error and no tool error.
func mustNotError(t *testing.T, r *mcp.CallToolResult, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if r.IsError {
		t.Fatalf("unexpected tool error: %s", resultText(r))
	}
}

// mustBeToolError asserts the Handle call returns a tool error (not a Go error).
func mustBeToolError(t *testing.T, r *mcp.CallToolResult, err error, wantSubstr string) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !r.IsError {
		t.Fatalf("expected tool error containing %q, got success: %s", wantSubstr, resultText(r))
	}
	if wantSubstr != "" && !strings.Contains(resultText(r), wantSubstr) {
		t.Errorf("error text %q does not contain %q", resultText(r), wantSubstr)
	}
}

// seedSession creates a session in the store for testing.
func seedSession(t *testing.T, store *memory.Store, id, project string) {
	t.Helper()
	if err := store.CreateSession(id, project, "/tmp/test"); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

// seedManualSession ensures the "manual-save" session exists (needed for FK constraints).
func seedManualSession(t *testing.T, store *memory.Store) {
	t.Helper()
	// CreateSession is idempotent (INSERT OR IGNORE), safe to call multiple times.
	if err := store.CreateSession("manual-save", "", "/tmp/test"); err != nil {
		t.Fatalf("seed manual session: %v", err)
	}
}

// seedObservation creates an observation and returns its ID.
// Requires a session with ID "test-session" to exist (call seedSession first).
func seedObservation(t *testing.T, store *memory.Store, title, content, project string) int64 {
	t.Helper()
	id, err := store.AddObservation(memory.AddObservationParams{
		SessionID: "test-session",
		Type:      "manual",
		Title:     title,
		Content:   content,
		Project:   project,
		Scope:     "project",
	})
	if err != nil {
		t.Fatalf("seed observation: %v", err)
	}
	return id
}

var ctx = context.Background()

// ─── SaveTool ────────────────────────────────────────────────────────────────

func TestSaveTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedManualSession(t, store)
	tool := NewSaveTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"title":   "JWT middleware",
		"content": "**What**: Added JWT\n**Why**: Auth needed",
		"type":    "decision",
		"project": "my-app",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "JWT middleware") {
		t.Errorf("expected title in response, got: %s", text)
	}
	if !strings.Contains(text, "decision") {
		t.Errorf("expected type in response, got: %s", text)
	}
	if !strings.Contains(text, "ID:") {
		t.Errorf("expected ID in response, got: %s", text)
	}
}

func TestSaveTool_SuggestsTopicKey(t *testing.T) {
	store := newTestStore(t)
	seedManualSession(t, store)
	tool := NewSaveTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"title":   "Auth middleware",
		"content": "Added auth middleware",
		"type":    "architecture",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Suggested topic_key:") {
		t.Errorf("expected topic_key suggestion, got: %s", text)
	}
}

func TestSaveTool_NoSuggestionWithTopicKey(t *testing.T) {
	store := newTestStore(t)
	seedManualSession(t, store)
	tool := NewSaveTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"title":     "Auth middleware",
		"content":   "Added auth middleware",
		"topic_key": "architecture/auth",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if strings.Contains(text, "Suggested topic_key:") {
		t.Errorf("should NOT suggest topic_key when one is provided, got: %s", text)
	}
}

func TestSaveTool_MissingTitle(t *testing.T) {
	store := newTestStore(t)
	tool := NewSaveTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content": "some content",
	}))

	mustBeToolError(t, r, err, "title")
}

func TestSaveTool_MissingContent(t *testing.T) {
	store := newTestStore(t)
	tool := NewSaveTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"title": "some title",
	}))

	mustBeToolError(t, r, err, "content")
}

// ─── SavePromptTool ──────────────────────────────────────────────────────────

func TestSavePromptTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedManualSession(t, store)
	tool := NewSavePromptTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content": "How do I set up auth?",
		"project": "my-app",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Prompt saved") {
		t.Errorf("expected success, got: %s", text)
	}
}

func TestSavePromptTool_MissingContent(t *testing.T) {
	store := newTestStore(t)
	tool := NewSavePromptTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "content")
}

// ─── SearchTool ──────────────────────────────────────────────────────────────

func TestSearchTool_FindsResults(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	seedObservation(t, store, "JWT middleware", "Added JWT auth middleware for Express", "my-app")
	seedObservation(t, store, "Database migration", "Ran Prisma migration for users table", "my-app")

	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query": "JWT auth",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "JWT middleware") {
		t.Errorf("expected JWT result, got: %s", text)
	}
}

func TestSearchTool_NoResults(t *testing.T) {
	store := newTestStore(t)
	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query": "nonexistent topic xyz123",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "No memories found") {
		t.Errorf("expected no-results message, got: %s", text)
	}
}

func TestSearchTool_MissingQuery(t *testing.T) {
	store := newTestStore(t)
	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "query")
}

func TestSearchTool_WithFilters(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj-a")

	if _, err := store.AddObservation(memory.AddObservationParams{
		SessionID: "test-session",
		Type:      "bugfix",
		Title:     "Fixed crash",
		Content:   "Fixed null pointer crash in handler",
		Project:   "proj-a",
		Scope:     "project",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if _, err := store.AddObservation(memory.AddObservationParams{
		SessionID: "test-session",
		Type:      "decision",
		Title:     "Chose PostgreSQL",
		Content:   "Decided to use PostgreSQL for crash data",
		Project:   "proj-b",
		Scope:     "project",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query":   "crash",
		"project": "proj-a",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Fixed crash") {
		t.Errorf("expected proj-a result, got: %s", text)
	}
	// proj-b result should be filtered out
	if strings.Contains(text, "Chose PostgreSQL") {
		t.Errorf("should not contain proj-b result, got: %s", text)
	}
}

// ─── ContextTool ─────────────────────────────────────────────────────────────

func TestContextTool_EmptyStore(t *testing.T) {
	store := newTestStore(t)
	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "No memory context") {
		t.Errorf("expected empty message, got: %s", text)
	}
}

func TestContextTool_WithData(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	seedObservation(t, store, "Added auth", "Auth module added", "my-app")

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project": "my-app",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// FormatContext should return something non-empty with our data
	if text == "" || strings.Contains(text, "No memory context") {
		t.Errorf("expected formatted context, got: %s", text)
	}
}

func TestContextTool_SummaryLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	seedObservation(t, store, "Added auth", "Auth module with JWT tokens and bcrypt hashing", "my-app")

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project":      "my-app",
		"detail_level": "summary",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Summary should have title but NOT content snippets
	if !strings.Contains(text, "Added auth") {
		t.Errorf("expected observation title, got: %s", text)
	}
	if strings.Contains(text, "JWT tokens") {
		t.Errorf("summary should NOT contain content snippets, got: %s", text)
	}
	// Should have footer hint
	if !strings.Contains(text, "detail_level") {
		t.Errorf("expected footer hint about detail_level, got: %s", text)
	}
}

func TestContextTool_FullLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	longContent := strings.Repeat("This is detailed content. ", 50) // 1300+ chars
	seedObservation(t, store, "Big observation", longContent, "my-app")

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project":      "my-app",
		"detail_level": "full",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Full mode should NOT truncate — all repeated blocks should be present
	// Standard mode truncates to 300 chars (~11 repetitions), full should have all 50
	lastChunk := "This is detailed content. This is detailed content. This is detailed content. "
	if !strings.Contains(text, lastChunk) {
		t.Errorf("full mode should contain untruncated content")
	}
	// Verify it's truly untruncated — check there are no truncation markers
	if strings.Contains(text, "...") {
		t.Errorf("full mode should NOT contain truncation markers, got: %s", text[:500])
	}
	// Should NOT have footer hint
	if strings.Contains(text, "💡") {
		t.Errorf("full mode should NOT have footer hint")
	}
}

func TestContextTool_StandardLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	longContent := strings.Repeat("Standard content block. ", 50)
	seedObservation(t, store, "Standard obs", longContent, "my-app")

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project":      "my-app",
		"detail_level": "standard",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Standard should truncate content
	if strings.Contains(text, longContent) {
		t.Errorf("standard mode should truncate long content")
	}
	if !strings.Contains(text, "Standard content block") {
		t.Errorf("standard mode should contain beginning of content, got: %s", text)
	}
}

func TestContextTool_LimitParam(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	seedObservation(t, store, "First obs", "First content", "my-app")
	seedObservation(t, store, "Second obs", "Second content", "my-app")
	seedObservation(t, store, "Third obs", "Third content", "my-app")

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project": "my-app",
		"limit":   float64(1),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// With limit=1, we should see at most 1 observation
	// Count observation entries (each starts with "- [manual]")
	count := strings.Count(text, "[manual]")
	if count > 1 {
		t.Errorf("expected at most 1 observation with limit=1, got %d occurrences", count)
	}
}

// ─── SearchTool detail_level ─────────────────────────────────────────────────

func TestSearchTool_SummaryLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	seedObservation(t, store, "JWT middleware", "Added JWT auth middleware with bcrypt and refresh tokens", "my-app")

	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query":        "JWT auth",
		"detail_level": "summary",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Summary should have title but NOT content snippets
	if !strings.Contains(text, "JWT middleware") {
		t.Errorf("summary should contain title, got: %s", text)
	}
	if strings.Contains(text, "bcrypt") {
		t.Errorf("summary should NOT contain content details, got: %s", text)
	}
	// Should have footer hint
	if !strings.Contains(text, "detail_level") {
		t.Errorf("summary should have footer hint, got: %s", text)
	}
}

func TestSearchTool_FullLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	longContent := strings.Repeat("Detailed search content. ", 50) // 1250+ chars
	seedObservation(t, store, "Big search result", longContent, "my-app")

	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query":        "search content",
		"detail_level": "full",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Full mode should contain much more content than the 300-char standard truncation.
	// The content is 1250+ chars — check that at least 800 chars appear (well beyond
	// the 300-char standard cutoff), proving full mode doesn't truncate.
	contentInResult := strings.Count(text, "Detailed search content.")
	if contentInResult < 30 {
		t.Errorf("full mode should contain most/all repetitions (got %d, want >= 30 of 50)", contentInResult)
	}
	// Should NOT have footer hint
	if strings.Contains(text, "💡") {
		t.Errorf("full mode should NOT have footer hint")
	}
}

func TestSearchTool_StandardLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	longContent := strings.Repeat("Standard search block. ", 50)
	seedObservation(t, store, "Standard search obs", longContent, "my-app")

	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query":        "search block",
		"detail_level": "standard",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Standard should truncate content (300 chars max snippet)
	if strings.Contains(text, longContent) {
		t.Errorf("standard mode should truncate long content")
	}
	if !strings.Contains(text, "Standard search block") {
		t.Errorf("standard mode should contain beginning of content, got: %s", text)
	}
}

// ─── TimelineTool detail_level ──────────────────────────────────────────────

func TestTimelineTool_SummaryLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")

	seedObservation(t, store, "Before obs", "Before observation detailed content", "my-app")
	id2 := seedObservation(t, store, "Focus obs", "Focus observation detailed content", "my-app")
	seedObservation(t, store, "After obs", "After observation detailed content", "my-app")

	tool := NewTimelineTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(id2),
		"before":         float64(5),
		"after":          float64(5),
		"detail_level":   "summary",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Summary should have titles but NOT content
	if !strings.Contains(text, "Focus obs") {
		t.Errorf("summary should contain focus title, got: %s", text)
	}
	if strings.Contains(text, "detailed content") {
		t.Errorf("summary should NOT contain observation content, got: %s", text)
	}
	// Should have footer hint
	if !strings.Contains(text, "detail_level") {
		t.Errorf("summary should have footer hint, got: %s", text)
	}
}

func TestTimelineTool_FullLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")

	longContent := strings.Repeat("Before content block. ", 30) // 660+ chars
	seedObservation(t, store, "Before full", longContent, "my-app")
	id2 := seedObservation(t, store, "Focus full", "Focus full content", "my-app")
	seedObservation(t, store, "After full", "After full content", "my-app")

	tool := NewTimelineTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(id2),
		"before":         float64(5),
		"after":          float64(5),
		"detail_level":   "full",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Full mode should contain much more content than the 200-char standard truncation.
	// The before content is 660+ chars — count repetitions to verify it's untruncated.
	contentInResult := strings.Count(text, "Before content block.")
	if contentInResult < 20 {
		t.Errorf("full mode should contain most/all before content repetitions (got %d, want >= 20 of 30)", contentInResult)
	}
	if !strings.Contains(text, "Focus full content") {
		t.Errorf("full mode should contain focus content")
	}
	// Should NOT have footer hint
	if strings.Contains(text, "💡") {
		t.Errorf("full mode should NOT have footer hint")
	}
}

func TestTimelineTool_StandardLevel(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")

	longContent := strings.Repeat("Standard timeline block. ", 30) // 720+ chars
	seedObservation(t, store, "Before std", longContent, "my-app")
	id2 := seedObservation(t, store, "Focus std", "Focus standard content here", "my-app")
	seedObservation(t, store, "After std", "After standard content", "my-app")

	tool := NewTimelineTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(id2),
		"before":         float64(5),
		"after":          float64(5),
		"detail_level":   "standard",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Standard should truncate before/after but show full focus
	if strings.Contains(text, longContent) {
		t.Errorf("standard mode should truncate long before content")
	}
	// Focus content should be shown in full (standard behavior)
	if !strings.Contains(text, "Focus standard content here") {
		t.Errorf("standard mode should show full focus content, got: %s", text)
	}
}

// ─── SessionStartTool ────────────────────────────────────────────────────────

func TestSessionStartTool_Success(t *testing.T) {
	store := newTestStore(t)
	tool := NewSessionStartTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id":        "sess-001",
		"project":   "my-app",
		"directory": "/home/user/my-app",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "sess-001") {
		t.Errorf("expected session id in response, got: %s", text)
	}
	if !strings.Contains(text, "my-app") {
		t.Errorf("expected project in response, got: %s", text)
	}
}

func TestSessionStartTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewSessionStartTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project": "my-app",
	}))

	mustBeToolError(t, r, err, "id")
}

func TestSessionStartTool_MissingProject(t *testing.T) {
	store := newTestStore(t)
	tool := NewSessionStartTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": "sess-001",
	}))

	mustBeToolError(t, r, err, "project")
}

// ─── SessionEndTool ──────────────────────────────────────────────────────────

func TestSessionEndTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "sess-001", "my-app")

	tool := NewSessionEndTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id":      "sess-001",
		"summary": "Completed auth module",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "sess-001") {
		t.Errorf("expected session id in response, got: %s", text)
	}
	if !strings.Contains(text, "completed") {
		t.Errorf("expected 'completed' in response, got: %s", text)
	}
}

func TestSessionEndTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewSessionEndTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "id")
}

// ─── SessionSummaryTool ──────────────────────────────────────────────────────

func TestSessionSummaryTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "sess-001", "my-app")

	tool := NewSessionSummaryTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content":    "## Goal\nBuild auth\n## Accomplished\n- Added JWT",
		"project":    "my-app",
		"session_id": "sess-001",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "summary saved") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "my-app") {
		t.Errorf("expected project in response, got: %s", text)
	}
}

func TestSessionSummaryTool_MissingContent(t *testing.T) {
	store := newTestStore(t)
	tool := NewSessionSummaryTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project": "my-app",
	}))

	mustBeToolError(t, r, err, "content")
}

func TestSessionSummaryTool_MissingProject(t *testing.T) {
	store := newTestStore(t)
	tool := NewSessionSummaryTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content": "some summary",
	}))

	mustBeToolError(t, r, err, "project")
}

// ─── TimelineTool ────────────────────────────────────────────────────────────

func TestTimelineTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")

	id1 := seedObservation(t, store, "First", "First observation", "my-app")
	id2 := seedObservation(t, store, "Second", "Second observation", "my-app")
	_ = seedObservation(t, store, "Third", "Third observation", "my-app")

	_ = id1 // suppress unused

	tool := NewTimelineTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(id2),
		"before":         float64(5),
		"after":          float64(5),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Second") {
		t.Errorf("expected focus observation, got: %s", text)
	}
}

func TestTimelineTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewTimelineTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "observation_id")
}

// ─── GetObservationTool ──────────────────────────────────────────────────────

func TestGetObservationTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	obsID := seedObservation(t, store, "JWT auth", "Full JWT implementation details", "my-app")

	tool := NewGetObservationTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(obsID),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "JWT auth") {
		t.Errorf("expected title in response, got: %s", text)
	}
	if !strings.Contains(text, "Full JWT implementation details") {
		t.Errorf("expected full content, got: %s", text)
	}
}

func TestGetObservationTool_NotFound(t *testing.T) {
	store := newTestStore(t)
	tool := NewGetObservationTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(99999),
	}))

	mustBeToolError(t, r, err, "not found")
}

func TestGetObservationTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewGetObservationTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "id")
}

// ─── StatsTool ───────────────────────────────────────────────────────────────

func TestStatsTool_EmptyStore(t *testing.T) {
	store := newTestStore(t)
	tool := NewStatsTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Sessions") {
		t.Errorf("expected Sessions header, got: %s", text)
	}
	if !strings.Contains(text, "Observations") {
		t.Errorf("expected Observations header, got: %s", text)
	}
	if !strings.Contains(text, "none") {
		t.Errorf("expected 'none' for empty projects, got: %s", text)
	}
}

func TestStatsTool_WithData(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj-a")
	seedObservation(t, store, "Something", "Details", "proj-a")

	tool := NewStatsTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "proj-a") {
		t.Errorf("expected project name, got: %s", text)
	}
	if strings.Contains(text, "none") {
		t.Errorf("should not say 'none' when projects exist, got: %s", text)
	}
}

// ─── DeleteTool ──────────────────────────────────────────────────────────────

func TestDeleteTool_SoftDelete(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	obsID := seedObservation(t, store, "To delete", "This will be deleted", "my-app")

	tool := NewDeleteTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(obsID),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "soft-deleted") {
		t.Errorf("expected soft-deleted message, got: %s", text)
	}

	// Verify it's soft-deleted (GetObservation should fail)
	_, getErr := store.GetObservation(obsID)
	if getErr == nil {
		t.Errorf("expected error when getting soft-deleted observation")
	}
}

func TestDeleteTool_HardDelete(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	obsID := seedObservation(t, store, "To remove", "This will be permanently removed", "my-app")

	tool := NewDeleteTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id":          float64(obsID),
		"hard_delete": true,
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "permanently deleted") {
		t.Errorf("expected permanently deleted message, got: %s", text)
	}
}

func TestDeleteTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewDeleteTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "id")
}

// ─── UpdateTool ──────────────────────────────────────────────────────────────

func TestUpdateTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	obsID := seedObservation(t, store, "Original title", "Original content", "my-app")

	tool := NewUpdateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id":    float64(obsID),
		"title": "Updated title",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Updated title") {
		t.Errorf("expected updated title, got: %s", text)
	}

	// Verify in store
	obs, err := store.GetObservation(obsID)
	if err != nil {
		t.Fatalf("failed to get updated observation: %v", err)
	}
	if obs.Title != "Updated title" {
		t.Errorf("expected title 'Updated title', got: %s", obs.Title)
	}
}

func TestUpdateTool_MultipleFields(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	obsID := seedObservation(t, store, "Original", "Original content", "my-app")

	tool := NewUpdateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id":      float64(obsID),
		"title":   "New Title",
		"content": "New Content",
		"type":    "bugfix",
	}))

	mustNotError(t, r, err)

	obs, err := store.GetObservation(obsID)
	if err != nil {
		t.Fatalf("get observation: %v", err)
	}
	if obs.Title != "New Title" {
		t.Errorf("title = %q, want 'New Title'", obs.Title)
	}
	if obs.Content != "New Content" {
		t.Errorf("content = %q, want 'New Content'", obs.Content)
	}
	if obs.Type != "bugfix" {
		t.Errorf("type = %q, want 'bugfix'", obs.Type)
	}
}

func TestUpdateTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewUpdateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"title": "test",
	}))

	mustBeToolError(t, r, err, "id")
}

func TestUpdateTool_NoFields(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	obsID := seedObservation(t, store, "Test", "Content", "my-app")

	tool := NewUpdateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(obsID),
	}))

	mustBeToolError(t, r, err, "at least one field")
}

// ─── SuggestTopicKeyTool ─────────────────────────────────────────────────────

func TestSuggestTopicKeyTool_FromTitle(t *testing.T) {
	tool := NewSuggestTopicKeyTool()

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"title": "Auth middleware setup",
		"type":  "architecture",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Suggested topic_key:") {
		t.Errorf("expected topic_key suggestion, got: %s", text)
	}
	// Should contain a slash (family/segment pattern)
	if !strings.Contains(text, "/") {
		t.Errorf("expected family/segment format, got: %s", text)
	}
}

func TestSuggestTopicKeyTool_FromContent(t *testing.T) {
	tool := NewSuggestTopicKeyTool()

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content": "Decided to use PostgreSQL for the database",
		"type":    "decision",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Suggested topic_key:") {
		t.Errorf("expected topic_key suggestion, got: %s", text)
	}
}

func TestSuggestTopicKeyTool_MissingBoth(t *testing.T) {
	tool := NewSuggestTopicKeyTool()

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "title")
}

// ─── PassiveCaptureTool ──────────────────────────────────────────────────────

func TestPassiveCaptureTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "sess-1", "my-app")

	tool := NewPassiveCaptureTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content":    "TIL: You need to handle the case where the database is locked. Also learned that WAL mode prevents this issue.",
		"session_id": "sess-1",
		"project":    "my-app",
		"source":     "conversation",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Passive capture complete") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "extracted") {
		t.Errorf("expected 'extracted' count, got: %s", text)
	}
}

func TestPassiveCaptureTool_NoLearnings(t *testing.T) {
	store := newTestStore(t)
	tool := NewPassiveCaptureTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"content": "Hello, how are you?",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "0 extracted") {
		t.Errorf("expected 0 extracted for trivial content, got: %s", text)
	}
}

func TestPassiveCaptureTool_MissingContent(t *testing.T) {
	store := newTestStore(t)
	tool := NewPassiveCaptureTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "content")
}

// ─── RelateTool ──────────────────────────────────────────────────────────────

func TestRelateTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id1 := seedObservation(t, store, "Decision A", "First decision content", "my-app")
	id2 := seedObservation(t, store, "Decision B", "Second decision content", "my-app")

	tool := NewRelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"from_id":       float64(id1),
		"to_id":         float64(id2),
		"relation_type": "depends_on",
		"note":          "B depends on A",
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Relation created") {
		t.Errorf("expected 'Relation created', got: %s", text)
	}
	if !strings.Contains(text, "depends_on") {
		t.Errorf("expected relation type in response, got: %s", text)
	}
	if !strings.Contains(text, "Relation ID:") {
		t.Errorf("expected relation ID in response, got: %s", text)
	}
}

func TestRelateTool_Bidirectional(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id1 := seedObservation(t, store, "Bidir Left", "Left side content", "my-app")
	id2 := seedObservation(t, store, "Bidir Right", "Right side content", "my-app")

	tool := NewRelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"from_id":       float64(id1),
		"to_id":         float64(id2),
		"relation_type": "relates_to",
		"bidirectional": true,
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Bidirectional") {
		t.Errorf("expected 'Bidirectional' in response, got: %s", text)
	}
	if !strings.Contains(text, "↔") {
		t.Errorf("expected bidirectional arrow ↔, got: %s", text)
	}
}

func TestRelateTool_MissingFromID(t *testing.T) {
	store := newTestStore(t)
	tool := NewRelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"to_id":         float64(1),
		"relation_type": "relates_to",
	}))

	mustBeToolError(t, r, err, "from_id")
}

func TestRelateTool_MissingToID(t *testing.T) {
	store := newTestStore(t)
	tool := NewRelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"from_id":       float64(1),
		"relation_type": "relates_to",
	}))

	mustBeToolError(t, r, err, "to_id")
}

func TestRelateTool_MissingRelationType(t *testing.T) {
	store := newTestStore(t)
	tool := NewRelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"from_id": float64(1),
		"to_id":   float64(2),
	}))

	mustBeToolError(t, r, err, "relation_type")
}

func TestRelateTool_SelfRelationError(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id := seedObservation(t, store, "Self ref", "Self reference content", "my-app")

	tool := NewRelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"from_id":       float64(id),
		"to_id":         float64(id),
		"relation_type": "relates_to",
	}))

	mustBeToolError(t, r, err, "self-relation")
}

// ─── UnrelateTool ────────────────────────────────────────────────────────────

func TestUnrelateTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id1 := seedObservation(t, store, "Unrel A", "Unrelate content A", "my-app")
	id2 := seedObservation(t, store, "Unrel B", "Unrelate content B", "my-app")

	relIDs, err := store.AddRelation(memory.AddRelationParams{
		FromID: id1, ToID: id2, Type: "relates_to",
	})
	if err != nil {
		t.Fatalf("seed relation: %v", err)
	}

	tool := NewUnrelateTool(store)

	r, toolErr := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(relIDs[0]),
	}))

	mustNotError(t, r, toolErr)
	text := resultText(r)

	if !strings.Contains(text, "removed") {
		t.Errorf("expected 'removed' in response, got: %s", text)
	}
}

func TestUnrelateTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewUnrelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "id")
}

func TestUnrelateTool_NotFound(t *testing.T) {
	store := newTestStore(t)
	tool := NewUnrelateTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(99999),
	}))

	mustBeToolError(t, r, err, "not found")
}

// ─── BuildContextTool ────────────────────────────────────────────────────────

func TestBuildContextTool_Success(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id1 := seedObservation(t, store, "Root node", "Root context content", "my-app")
	id2 := seedObservation(t, store, "Child node", "Child context content", "my-app")

	if _, err := store.AddRelation(memory.AddRelationParams{
		FromID: id1, ToID: id2, Type: "implements",
	}); err != nil {
		t.Fatalf("AddRelation: %v", err)
	}

	tool := NewBuildContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(id1),
		"depth":          float64(2),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Context Graph") {
		t.Errorf("expected 'Context Graph' header, got: %s", text)
	}
	if !strings.Contains(text, "Root node") {
		t.Errorf("expected root title, got: %s", text)
	}
	if !strings.Contains(text, "Child node") {
		t.Errorf("expected connected child title, got: %s", text)
	}
	if !strings.Contains(text, "implements") {
		t.Errorf("expected relation type, got: %s", text)
	}
}

func TestBuildContextTool_NoRelations(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id := seedObservation(t, store, "Isolated node", "No connections here", "my-app")

	tool := NewBuildContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(id),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "No relations found") {
		t.Errorf("expected 'No relations found', got: %s", text)
	}
}

func TestBuildContextTool_MissingID(t *testing.T) {
	store := newTestStore(t)
	tool := NewBuildContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "observation_id")
}

func TestBuildContextTool_NotFound(t *testing.T) {
	store := newTestStore(t)
	tool := NewBuildContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(99999),
	}))

	mustBeToolError(t, r, err, "not found")
}

// ─── GetObservationTool with Relations ───────────────────────────────────────

func TestGetObservationTool_ShowsRelations(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id1 := seedObservation(t, store, "Obs with rels", "Content with relations", "my-app")
	id2 := seedObservation(t, store, "Related obs", "Related content", "my-app")

	if _, err := store.AddRelation(memory.AddRelationParams{
		FromID: id1, ToID: id2, Type: "depends_on", Note: "critical dependency",
	}); err != nil {
		t.Fatalf("AddRelation: %v", err)
	}

	tool := NewGetObservationTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(id1),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "## Relations") {
		t.Errorf("expected Relations section, got: %s", text)
	}
	if !strings.Contains(text, "Outgoing") {
		t.Errorf("expected Outgoing label, got: %s", text)
	}
	if !strings.Contains(text, "depends_on") {
		t.Errorf("expected relation type, got: %s", text)
	}
	if !strings.Contains(text, "critical dependency") {
		t.Errorf("expected note in output, got: %s", text)
	}
}

func TestGetObservationTool_NoRelationsSection(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id := seedObservation(t, store, "No rels obs", "Content without relations", "my-app")

	tool := NewGetObservationTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(id),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	// Should NOT have a Relations section when there are no relations
	if strings.Contains(text, "## Relations") {
		t.Errorf("should not show Relations section when none exist, got: %s", text)
	}
}

func TestGetObservationTool_IncomingRelations(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "my-app")
	id1 := seedObservation(t, store, "Target obs", "Target content", "my-app")
	id2 := seedObservation(t, store, "Source obs", "Source content", "my-app")

	// id2 → id1 (incoming from id1's perspective)
	if _, err := store.AddRelation(memory.AddRelationParams{
		FromID: id2, ToID: id1, Type: "caused_by",
	}); err != nil {
		t.Fatalf("AddRelation: %v", err)
	}

	tool := NewGetObservationTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"id": float64(id1),
	}))

	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Incoming") {
		t.Errorf("expected Incoming label, got: %s", text)
	}
	if !strings.Contains(text, "caused_by") {
		t.Errorf("expected relation type, got: %s", text)
	}
}

// ─── Definition tests ────────────────────────────────────────────────────────

func TestAllTools_HaveDefinitions(t *testing.T) {
	store := newTestStore(t)

	tools := []struct {
		name string
		def  mcp.Tool
	}{
		{"mem_save", NewSaveTool(store).Definition()},
		{"mem_save_prompt", NewSavePromptTool(store).Definition()},
		{"mem_search", NewSearchTool(store).Definition()},
		{"mem_context", NewContextTool(store).Definition()},
		{"mem_session_start", NewSessionStartTool(store).Definition()},
		{"mem_session_end", NewSessionEndTool(store).Definition()},
		{"mem_session_summary", NewSessionSummaryTool(store).Definition()},
		{"mem_timeline", NewTimelineTool(store).Definition()},
		{"mem_get_observation", NewGetObservationTool(store).Definition()},
		{"mem_stats", NewStatsTool(store).Definition()},
		{"mem_delete", NewDeleteTool(store).Definition()},
		{"mem_update", NewUpdateTool(store).Definition()},
		{"mem_suggest_topic_key", NewSuggestTopicKeyTool().Definition()},
		{"mem_capture_passive", NewPassiveCaptureTool(store).Definition()},
		{"mem_relate", NewRelateTool(store).Definition()},
		{"mem_unrelate", NewUnrelateTool(store).Definition()},
		{"mem_build_context", NewBuildContextTool(store).Definition()},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			if tt.def.Name != tt.name {
				t.Errorf("definition name = %q, want %q", tt.def.Name, tt.name)
			}
			if tt.def.Description == "" {
				t.Errorf("definition description is empty for %s", tt.name)
			}
		})
	}
}

// ─── helpers_test ────────────────────────────────────────────────────────────

func TestIntArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		def      int
		expected int
	}{
		{"present", map[string]interface{}{"limit": float64(20)}, "limit", 10, 20},
		{"missing", map[string]interface{}{}, "limit", 10, 10},
		{"wrong type", map[string]interface{}{"limit": "not a number"}, "limit", 10, 10},
		{"zero", map[string]interface{}{"limit": float64(0)}, "limit", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeReq(tt.args)
			got := intArg(req, tt.key, tt.def)
			if got != tt.expected {
				t.Errorf("intArg() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// ─── Navigation Hints Tests ──────────────────────────────────────────────────

func TestSearchTool_NavigationHint_WhenCapped(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")

	// Create 5 observations that all match "auth".
	for i := 0; i < 5; i++ {
		seedObservation(t, store, "auth fix "+string(rune('A'+i)), "Fixed auth bug in module "+string(rune('A'+i)), "proj")
	}

	tool := NewSearchTool(store)

	// Search with limit=2 — should cap results and show navigation hint.
	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query": "auth",
		"limit": float64(2),
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "📊 Showing 2 of 5") {
		t.Errorf("expected navigation hint 'Showing 2 of 5', got: %s", text)
	}
}

func TestSearchTool_NavigationHint_NotShownWhenAllReturned(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")
	seedObservation(t, store, "unique topic xyz", "Content about unique topic xyz", "proj")

	tool := NewSearchTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"query": "unique topic xyz",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if strings.Contains(text, "📊 Showing") {
		t.Errorf("navigation hint should NOT appear when all results returned, got: %s", text)
	}
}

func TestContextTool_NavigationHint_WhenCapped(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")

	// Create 25 observations — exceeds default limit of 20.
	for i := 0; i < 25; i++ {
		seedObservation(t, store, "obs "+string(rune('A'+i)), "Content "+string(rune('A'+i)), "proj")
	}

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project": "proj",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "📊 Showing 20 of 25") {
		t.Errorf("expected navigation hint 'Showing 20 of 25', got: %s", text)
	}
}

func TestContextTool_NavigationHint_NotShownWhenAllReturned(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")

	// Create 3 observations — below the default limit.
	for i := 0; i < 3; i++ {
		seedObservation(t, store, "obs "+string(rune('A'+i)), "Content "+string(rune('A'+i)), "proj")
	}

	tool := NewContextTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"project": "proj",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if strings.Contains(text, "📊 Showing") {
		t.Errorf("navigation hint should NOT appear when all results returned, got: %s", text)
	}
}

func TestTimelineTool_NavigationHint_WhenWindowSmaller(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")

	// Create 15 observations to ensure TotalInRange > window.
	var focusID int64
	for i := 0; i < 15; i++ {
		id := seedObservation(t, store, "timeline obs "+string(rune('A'+i)), "Content for timeline "+string(rune('A'+i)), "proj")
		if i == 7 {
			focusID = id
		}
	}

	tool := NewTimelineTool(store)

	// Request with before=2, after=2 — window of 5, but TotalInRange is 15.
	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"observation_id": float64(focusID),
		"before":         float64(2),
		"after":          float64(2),
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "📊 Showing") {
		t.Errorf("expected navigation hint when window < total, got: %s", text)
	}
	if !strings.Contains(text, "Increase before/after") {
		t.Errorf("expected hint about increasing before/after, got: %s", text)
	}
}

// ─── CompactTool ─────────────────────────────────────────────────────────────

func TestCompactTool_IdentifyMode_NoStaleFound(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")
	seedObservation(t, store, "Fresh obs", "content", "proj")

	tool := NewCompactTool(store)
	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"project":         "proj",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "No stale observations found") {
		t.Errorf("expected 'no stale' message, got: %s", text)
	}
}

func TestCompactTool_IdentifyMode_MissingDays(t *testing.T) {
	store := newTestStore(t)
	tool := NewCompactTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{}))
	mustBeToolError(t, r, err, "older_than_days")
}

func TestCompactTool_IdentifyMode_ZeroDays(t *testing.T) {
	store := newTestStore(t)
	tool := NewCompactTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(0),
	}))
	mustBeToolError(t, r, err, "must be > 0")
}

func TestCompactTool_ExecuteMode_InvalidJSON(t *testing.T) {
	store := newTestStore(t)
	tool := NewCompactTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"compact_ids":     "not valid json",
	}))
	mustBeToolError(t, r, err, "valid JSON array")
}

func TestCompactTool_ExecuteMode_EmptyArray(t *testing.T) {
	store := newTestStore(t)
	tool := NewCompactTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"compact_ids":     "[]",
	}))
	mustBeToolError(t, r, err, "empty")
}

func TestCompactTool_ExecuteMode_SummaryContentWithoutTitle(t *testing.T) {
	store := newTestStore(t)
	tool := NewCompactTool(store)

	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"compact_ids":     "[1]",
		"summary_content": "some content",
	}))
	mustBeToolError(t, r, err, "summary_title")
}

func TestCompactTool_ExecuteMode_BasicCompaction(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")
	id1 := seedObservation(t, store, "Compact me", "old content", "proj")

	tool := NewCompactTool(store)
	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"compact_ids":     fmt.Sprintf("[%d]", id1),
		"project":         "proj",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Compaction Complete") {
		t.Errorf("expected compaction complete message, got: %s", text)
	}
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected deleted count in response, got: %s", text)
	}
}

func TestCompactTool_ExecuteMode_WithSummary(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")
	seedManualSession(t, store)
	id1 := seedObservation(t, store, "Compact me", "old content", "proj")

	tool := NewCompactTool(store)
	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"compact_ids":     fmt.Sprintf("[%d]", id1),
		"project":         "proj",
		"summary_title":   "Consolidated old notes",
		"summary_content": "These were old session notes.",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "Compaction Complete") {
		t.Errorf("expected compaction complete, got: %s", text)
	}
	if !strings.Contains(text, "compaction_summary") {
		t.Errorf("expected summary type in response, got: %s", text)
	}
}

func TestCompactTool_ExecuteMode_SkippedWarning(t *testing.T) {
	store := newTestStore(t)
	seedSession(t, store, "test-session", "proj")
	id1 := seedObservation(t, store, "Will delete", "content", "proj")

	// Soft-delete first
	if err := store.DeleteObservation(id1, false); err != nil {
		t.Fatalf("pre-delete: %v", err)
	}

	tool := NewCompactTool(store)
	r, err := tool.Handle(ctx, makeReq(map[string]interface{}{
		"older_than_days": float64(30),
		"compact_ids":     fmt.Sprintf("[%d]", id1),
		"project":         "proj",
	}))
	mustNotError(t, r, err)
	text := resultText(r)

	if !strings.Contains(text, "skipped") {
		t.Errorf("expected skipped warning for already-deleted obs, got: %s", text)
	}
}

func TestBoolArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		def      bool
		expected bool
	}{
		{"true", map[string]interface{}{"flag": true}, "flag", false, true},
		{"false", map[string]interface{}{"flag": false}, "flag", true, false},
		{"missing", map[string]interface{}{}, "flag", true, true},
		{"wrong type", map[string]interface{}{"flag": "yes"}, "flag", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeReq(tt.args)
			got := boolArg(req, tt.key, tt.def)
			if got != tt.expected {
				t.Errorf("boolArg() = %v, want %v", got, tt.expected)
			}
		})
	}
}
