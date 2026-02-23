package memtools

import (
	"context"
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
