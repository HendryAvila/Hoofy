package memory_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
)

// newTestStore creates a Store backed by a temp directory for isolation.
func newTestStore(t *testing.T) *memory.Store {
	t.Helper()
	cfg := memory.Config{
		DataDir:              t.TempDir(),
		MaxObservationLength: 2000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	}
	s, err := memory.New(cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ensureSession creates a session that observations depend on.
func ensureSession(t *testing.T, s *memory.Store, id, project string) {
	t.Helper()
	if err := s.CreateSession(id, project, "/tmp/test"); err != nil {
		t.Fatalf("failed to create session %q: %v", id, err)
	}
}

// ─── New / Initialization ───────────────────────────────────────────────────

func TestNew_CreatesDBFile(t *testing.T) {
	dir := t.TempDir()
	cfg := memory.Config{
		DataDir:              dir,
		MaxObservationLength: 2000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	}
	s, err := memory.New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer s.Close()

	dbPath := filepath.Join(dir, "memory.db")
	if _, err := filepath.Abs(dbPath); err != nil {
		t.Fatal(err)
	}
}

func TestNew_IdempotentReopen(t *testing.T) {
	dir := t.TempDir()
	cfg := memory.Config{
		DataDir:              dir,
		MaxObservationLength: 2000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	}

	// Open, insert, close
	s1, err := memory.New(cfg)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := s1.CreateSession("sess-1", "proj", "/tmp"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	s1.Close()

	// Reopen — data should persist
	s2, err := memory.New(cfg)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer s2.Close()

	sess, err := s2.GetSession("sess-1")
	if err != nil {
		t.Fatalf("session not found after reopen: %v", err)
	}
	if sess.Project != "proj" {
		t.Errorf("project = %q, want %q", sess.Project, "proj")
	}
}

// ─── Sessions ───────────────────────────────────────────────────────────────

func TestCreateSession_Basic(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "s1", "my-project")

	sess, err := s.GetSession("s1")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if sess.ID != "s1" {
		t.Errorf("ID = %q, want %q", sess.ID, "s1")
	}
	if sess.Project != "my-project" {
		t.Errorf("Project = %q, want %q", sess.Project, "my-project")
	}
	if sess.EndedAt != nil {
		t.Errorf("EndedAt should be nil, got %v", sess.EndedAt)
	}
}

func TestCreateSession_DuplicateIgnored(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "dup", "proj1")
	// Second create with same ID should be silently ignored (INSERT OR IGNORE)
	if err := s.CreateSession("dup", "proj2", "/other"); err != nil {
		t.Fatalf("duplicate create: %v", err)
	}
	sess, err := s.GetSession("dup")
	if err != nil {
		t.Fatal(err)
	}
	// Original data should be preserved
	if sess.Project != "proj1" {
		t.Errorf("Project = %q, want %q (original)", sess.Project, "proj1")
	}
}

func TestEndSession(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "s-end", "proj")

	if err := s.EndSession("s-end", "All done"); err != nil {
		t.Fatalf("EndSession error: %v", err)
	}

	sess, err := s.GetSession("s-end")
	if err != nil {
		t.Fatal(err)
	}
	if sess.EndedAt == nil {
		t.Error("EndedAt should be set after EndSession")
	}
	if sess.Summary == nil || *sess.Summary != "All done" {
		t.Errorf("Summary = %v, want %q", sess.Summary, "All done")
	}
}

func TestEndSession_EmptySummary(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "s-empty", "proj")

	if err := s.EndSession("s-empty", ""); err != nil {
		t.Fatalf("EndSession error: %v", err)
	}

	sess, err := s.GetSession("s-empty")
	if err != nil {
		t.Fatal(err)
	}
	if sess.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
	// Empty summary → stored as NULL
	if sess.Summary != nil {
		t.Errorf("Summary = %v, want nil for empty string", sess.Summary)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSession("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestRecentSessions_OrderAndLimit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 10; i++ {
		id := "rs-" + string(rune('a'+i))
		ensureSession(t, s, id, "proj")
	}

	results, err := s.RecentSessions("", 3)
	if err != nil {
		t.Fatalf("RecentSessions error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("len = %d, want 3", len(results))
	}
}

func TestRecentSessions_FilterByProject(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "p1-s1", "alpha")
	ensureSession(t, s, "p2-s1", "beta")
	ensureSession(t, s, "p1-s2", "alpha")

	results, err := s.RecentSessions("alpha", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("len = %d, want 2 for project alpha", len(results))
	}
	for _, r := range results {
		if r.Project != "alpha" {
			t.Errorf("unexpected project %q", r.Project)
		}
	}
}

func TestRecentSessions_DefaultLimit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 10; i++ {
		id := "dl-" + string(rune('a'+i))
		ensureSession(t, s, id, "proj")
	}

	results, err := s.RecentSessions("", 0) // 0 → default (5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 5 {
		t.Errorf("len = %d, want 5 (default limit)", len(results))
	}
}

// ─── Observations ────────────────────────────────────────────────────────────

func TestAddObservation_Basic(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "decision",
		Title:     "Use PostgreSQL",
		Content:   "Decided to use PostgreSQL for ACID compliance",
		Project:   "proj",
		Scope:     "project",
	})
	if err != nil {
		t.Fatalf("AddObservation error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	obs, err := s.GetObservation(id)
	if err != nil {
		t.Fatalf("GetObservation error: %v", err)
	}
	if obs.Title != "Use PostgreSQL" {
		t.Errorf("Title = %q, want %q", obs.Title, "Use PostgreSQL")
	}
	if obs.Type != "decision" {
		t.Errorf("Type = %q, want %q", obs.Type, "decision")
	}
	if obs.Scope != "project" {
		t.Errorf("Scope = %q, want %q", obs.Scope, "project")
	}
	if obs.RevisionCount != 1 {
		t.Errorf("RevisionCount = %d, want 1", obs.RevisionCount)
	}
	if obs.DuplicateCount != 1 {
		t.Errorf("DuplicateCount = %d, want 1", obs.DuplicateCount)
	}
}

func TestAddObservation_TopicKeyUpsert(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id1, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "architecture",
		Title:     "Auth strategy v1",
		Content:   "Using session cookies",
		Project:   "proj",
		Scope:     "project",
		TopicKey:  "architecture/auth",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Same topic_key → should upsert (return same ID)
	id2, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "architecture",
		Title:     "Auth strategy v2",
		Content:   "Switched to JWT with refresh tokens",
		Project:   "proj",
		Scope:     "project",
		TopicKey:  "architecture/auth",
	})
	if err != nil {
		t.Fatal(err)
	}

	if id1 != id2 {
		t.Errorf("topic_key upsert: id1=%d, id2=%d — expected same ID", id1, id2)
	}

	obs, err := s.GetObservation(id1)
	if err != nil {
		t.Fatal(err)
	}
	if obs.Content != "Switched to JWT with refresh tokens" {
		t.Errorf("content not updated: %q", obs.Content)
	}
	if obs.RevisionCount != 2 {
		t.Errorf("RevisionCount = %d, want 2 after upsert", obs.RevisionCount)
	}
}

func TestAddObservation_Deduplication(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	params := memory.AddObservationParams{
		SessionID: "sess",
		Type:      "decision",
		Title:     "Same Title",
		Content:   "Exact same content for dedup test",
		Project:   "proj",
		Scope:     "project",
	}

	id1, err := s.AddObservation(params)
	if err != nil {
		t.Fatal(err)
	}

	// Same content within dedup window → should return same ID
	id2, err := s.AddObservation(params)
	if err != nil {
		t.Fatal(err)
	}

	if id1 != id2 {
		t.Errorf("dedup: id1=%d, id2=%d — expected same ID", id1, id2)
	}

	obs, err := s.GetObservation(id1)
	if err != nil {
		t.Fatal(err)
	}
	if obs.DuplicateCount != 2 {
		t.Errorf("DuplicateCount = %d, want 2 after dedup", obs.DuplicateCount)
	}
}

func TestAddObservation_PrivateTagsStripped(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "decision",
		Title:     "Has <private>secret</private> title",
		Content:   "Content with <private>API_KEY=abc123</private> inside",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	obs, err := s.GetObservation(id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(obs.Title, "secret") {
		t.Errorf("private tag not stripped from title: %q", obs.Title)
	}
	if strings.Contains(obs.Content, "API_KEY") {
		t.Errorf("private tag not stripped from content: %q", obs.Content)
	}
	if !strings.Contains(obs.Title, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in title, got: %q", obs.Title)
	}
	if !strings.Contains(obs.Content, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in content, got: %q", obs.Content)
	}
}

func TestAddObservation_Truncation(t *testing.T) {
	dir := t.TempDir()
	cfg := memory.Config{
		DataDir:              dir,
		MaxObservationLength: 50,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	}
	s, err := memory.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.CreateSession("sess", "proj", "/tmp"); err != nil {
		t.Fatal(err)
	}

	longContent := strings.Repeat("x", 200)
	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "manual",
		Title:     "Long",
		Content:   longContent,
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	obs, err := s.GetObservation(id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(obs.Content, "... [truncated]") {
		t.Errorf("expected truncated content, got len=%d suffix=%q",
			len(obs.Content), obs.Content[len(obs.Content)-20:])
	}
}

func TestAddObservation_ScopeNormalization(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	tests := []struct {
		input string
		want  string
	}{
		{"", "project"},
		{"project", "project"},
		{"personal", "personal"},
		{"PERSONAL", "personal"},
		{"  Personal  ", "personal"},
		{"invalid", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, err := s.AddObservation(memory.AddObservationParams{
				SessionID: "sess",
				Type:      "manual",
				Title:     "Scope test " + tt.input,
				Content:   "Content for scope " + tt.input + " " + tt.want,
				Project:   "proj",
				Scope:     tt.input,
			})
			if err != nil {
				t.Fatal(err)
			}
			obs, err := s.GetObservation(id)
			if err != nil {
				t.Fatal(err)
			}
			if obs.Scope != tt.want {
				t.Errorf("scope %q → %q, want %q", tt.input, obs.Scope, tt.want)
			}
		})
	}
}

func TestGetObservation_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetObservation(99999)
	if err == nil {
		t.Error("expected error for nonexistent observation")
	}
}

func TestGetObservation_SoftDeletedHidden(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "manual",
		Title:     "Will be deleted",
		Content:   "Ephemeral content",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteObservation(id, false); err != nil {
		t.Fatal(err)
	}

	_, err = s.GetObservation(id)
	if err == nil {
		t.Error("soft-deleted observation should not be found by GetObservation")
	}
}

func TestUpdateObservation(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "decision",
		Title:     "Original Title",
		Content:   "Original content",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Updated Title"
	newContent := "Updated content with more detail"
	newScope := "personal"
	updated, err := s.UpdateObservation(id, memory.UpdateObservationParams{
		Title:   &newTitle,
		Content: &newContent,
		Scope:   &newScope,
	})
	if err != nil {
		t.Fatalf("UpdateObservation error: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Content != newContent {
		t.Errorf("Content = %q, want %q", updated.Content, newContent)
	}
	if updated.Scope != "personal" {
		t.Errorf("Scope = %q, want personal", updated.Scope)
	}
	if updated.RevisionCount != 2 {
		t.Errorf("RevisionCount = %d, want 2", updated.RevisionCount)
	}
}

func TestUpdateObservation_NotFound(t *testing.T) {
	s := newTestStore(t)
	title := "x"
	_, err := s.UpdateObservation(99999, memory.UpdateObservationParams{Title: &title})
	if err == nil {
		t.Error("expected error for nonexistent observation")
	}
}

func TestUpdateObservation_PrivateTagsStripped(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "manual",
		Title:     "Safe",
		Content:   "Clean content",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	privateContent := "Has <private>SECRET</private> data"
	updated, err := s.UpdateObservation(id, memory.UpdateObservationParams{
		Content: &privateContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(updated.Content, "SECRET") {
		t.Errorf("private tag not stripped on update: %q", updated.Content)
	}
}

func TestDeleteObservation_SoftDelete(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "manual",
		Title:     "To soft delete",
		Content:   "Content",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteObservation(id, false); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// Should not appear in GetObservation
	_, err = s.GetObservation(id)
	if err == nil {
		t.Error("soft-deleted should not be returned")
	}

	// But should still exist in search/recent (filtered by deleted_at IS NULL)
	obs, err := s.RecentObservations("proj", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range obs {
		if o.ID == id {
			t.Error("soft-deleted should not appear in RecentObservations")
		}
	}
}

func TestDeleteObservation_HardDelete(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "manual",
		Title:     "To hard delete",
		Content:   "Gone forever",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteObservation(id, true); err != nil {
		t.Fatalf("hard delete: %v", err)
	}

	_, err = s.GetObservation(id)
	if err == nil {
		t.Error("hard-deleted should not be found")
	}
}

func TestRecentObservations(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	for i := 0; i < 5; i++ {
		_, err := s.AddObservation(memory.AddObservationParams{
			SessionID: "sess",
			Type:      "manual",
			Title:     "Obs " + string(rune('A'+i)),
			Content:   "Content " + string(rune('A'+i)) + " unique content for observation",
			Project:   "proj",
			Scope:     "project",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	results, err := s.RecentObservations("proj", "project", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("len = %d, want 3", len(results))
	}
}

func TestRecentObservations_EmptyFilters(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	_, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess",
		Type:      "manual",
		Title:     "Any",
		Content:   "Universal content here",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	// No filters → return all
	results, err := s.RecentObservations("", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result with no filters")
	}
}

// ─── User Prompts ────────────────────────────────────────────────────────────

func TestAddPrompt_Basic(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddPrompt(memory.AddPromptParams{
		SessionID: "sess",
		Content:   "How do I implement JWT auth?",
		Project:   "proj",
	})
	if err != nil {
		t.Fatalf("AddPrompt error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestAddPrompt_PrivateTagsStripped(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, err := s.AddPrompt(memory.AddPromptParams{
		SessionID: "sess",
		Content:   "Use this key: <private>sk-abc123</private>",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}

	prompts, err := s.RecentPrompts("proj", 10)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, p := range prompts {
		if p.ID == id {
			found = true
			if strings.Contains(p.Content, "sk-abc123") {
				t.Errorf("private content not stripped: %q", p.Content)
			}
			if !strings.Contains(p.Content, "[REDACTED]") {
				t.Errorf("expected [REDACTED] in prompt, got: %q", p.Content)
			}
		}
	}
	if !found {
		t.Error("prompt not found in RecentPrompts")
	}
}

func TestRecentPrompts_FilterByProject(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess1", "alpha")
	ensureSession(t, s, "sess2", "beta")

	s.AddPrompt(memory.AddPromptParams{SessionID: "sess1", Content: "Alpha question 1", Project: "alpha"})
	s.AddPrompt(memory.AddPromptParams{SessionID: "sess2", Content: "Beta question 1", Project: "beta"})
	s.AddPrompt(memory.AddPromptParams{SessionID: "sess1", Content: "Alpha question 2", Project: "alpha"})

	results, err := s.RecentPrompts("alpha", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("len = %d, want 2", len(results))
	}
}

func TestSearchPrompts(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	s.AddPrompt(memory.AddPromptParams{SessionID: "sess", Content: "How to implement JWT authentication", Project: "proj"})
	s.AddPrompt(memory.AddPromptParams{SessionID: "sess", Content: "Fix the database migration error", Project: "proj"})
	s.AddPrompt(memory.AddPromptParams{SessionID: "sess", Content: "Add unit tests for user service", Project: "proj"})

	results, err := s.SearchPrompts("JWT authentication", "", 10)
	if err != nil {
		t.Fatalf("SearchPrompts error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 search result for 'JWT authentication'")
	}
}

// ─── Search (FTS5) ──────────────────────────────────────────────────────────

func TestSearch_Basic(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "decision", Title: "JWT middleware",
		Content: "Implemented JWT authentication with refresh tokens", Project: "proj",
	})
	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "bugfix", Title: "Memory leak fix",
		Content: "Fixed goroutine leak in WebSocket handler", Project: "proj",
	})
	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "pattern", Title: "Repository pattern",
		Content: "Using repository pattern for data access layer", Project: "proj",
	})

	results, err := s.Search("JWT authentication", memory.SearchOptions{})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result for 'JWT authentication'")
	}
	// First result should be the JWT one (best match)
	if !strings.Contains(results[0].Title, "JWT") {
		t.Errorf("first result title = %q, expected JWT match", results[0].Title)
	}
}

func TestSearch_FilterByType(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "decision", Title: "Decision about auth",
		Content: "Auth flow decision details", Project: "proj",
	})
	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "bugfix", Title: "Auth bug fix",
		Content: "Fixed auth bug in login flow", Project: "proj",
	})

	results, err := s.Search("auth", memory.SearchOptions{Type: "bugfix"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Type != "bugfix" {
			t.Errorf("expected type bugfix, got %q", r.Type)
		}
	}
}

func TestSearch_FilterByProject(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "s1", "alpha")
	ensureSession(t, s, "s2", "beta")

	s.AddObservation(memory.AddObservationParams{
		SessionID: "s1", Type: "manual", Title: "Alpha auth",
		Content: "Authentication for alpha project", Project: "alpha",
	})
	s.AddObservation(memory.AddObservationParams{
		SessionID: "s2", Type: "manual", Title: "Beta auth",
		Content: "Authentication for beta project", Project: "beta",
	})

	results, err := s.Search("authentication", memory.SearchOptions{Project: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		proj := ""
		if r.Project != nil {
			proj = *r.Project
		}
		if proj != "alpha" {
			t.Errorf("expected project alpha, got %q", proj)
		}
	}
}

func TestSearch_LimitCapped(t *testing.T) {
	dir := t.TempDir()
	cfg := memory.Config{
		DataDir:              dir,
		MaxObservationLength: 2000,
		MaxContextResults:    20,
		MaxSearchResults:     5,
		DedupeWindow:         15 * time.Minute,
	}
	s, err := memory.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.CreateSession("sess", "proj", "/tmp"); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 20; i++ {
		s.AddObservation(memory.AddObservationParams{
			SessionID: "sess", Type: "manual", Title: "Auth entry",
			Content: "Authentication related content number " + string(rune('a'+i)),
			Project: "proj",
		})
	}

	results, err := s.Search("auth", memory.SearchOptions{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > 5 {
		t.Errorf("len = %d, should be capped at MaxSearchResults=5", len(results))
	}
}

func TestSearch_SoftDeletedExcluded(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "manual", Title: "Deletable search item",
		Content: "Unique searchable deletable content xyzzy", Project: "proj",
	})
	s.DeleteObservation(id, false)

	results, err := s.Search("xyzzy", memory.SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.ID == id {
			t.Error("soft-deleted observation should not appear in search results")
		}
	}
}

// ─── Timeline ────────────────────────────────────────────────────────────────

func TestTimeline_Basic(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	var ids []int64
	for i := 0; i < 7; i++ {
		id, err := s.AddObservation(memory.AddObservationParams{
			SessionID: "sess",
			Type:      "manual",
			Title:     "Timeline obs " + string(rune('A'+i)),
			Content:   "Content " + string(rune('A'+i)) + " for timeline testing unique",
			Project:   "proj",
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}

	// Focus on the 4th observation (index 3)
	focusID := ids[3]
	result, err := s.Timeline(focusID, 2, 2)
	if err != nil {
		t.Fatalf("Timeline error: %v", err)
	}

	if result.Focus.ID != focusID {
		t.Errorf("Focus.ID = %d, want %d", result.Focus.ID, focusID)
	}
	if len(result.Before) > 2 {
		t.Errorf("Before: len = %d, want <= 2", len(result.Before))
	}
	if len(result.After) > 2 {
		t.Errorf("After: len = %d, want <= 2", len(result.After))
	}
	if result.SessionInfo == nil {
		t.Error("SessionInfo should not be nil")
	}
	if result.TotalInRange != 7 {
		t.Errorf("TotalInRange = %d, want 7", result.TotalInRange)
	}
}

func TestTimeline_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Timeline(99999, 5, 5)
	if err == nil {
		t.Error("expected error for nonexistent observation")
	}
}

func TestTimeline_DefaultLimits(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "manual", Title: "Solo",
		Content: "Only observation for default timeline limits", Project: "proj",
	})

	// Passing 0,0 → defaults to 5,5
	result, err := s.Timeline(id, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Focus.ID != id {
		t.Errorf("Focus.ID = %d, want %d", result.Focus.ID, id)
	}
}

// ─── Stats ──────────────────────────────────────────────────────────────────

func TestStats_Empty(t *testing.T) {
	s := newTestStore(t)

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", stats.TotalSessions)
	}
	if stats.TotalObservations != 0 {
		t.Errorf("TotalObservations = %d, want 0", stats.TotalObservations)
	}
	if stats.TotalPrompts != 0 {
		t.Errorf("TotalPrompts = %d, want 0", stats.TotalPrompts)
	}
}

func TestStats_WithData(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "s1", "proj-a")
	ensureSession(t, s, "s2", "proj-b")

	s.AddObservation(memory.AddObservationParams{
		SessionID: "s1", Type: "manual", Title: "O1",
		Content: "Observation one content", Project: "proj-a",
	})
	s.AddObservation(memory.AddObservationParams{
		SessionID: "s2", Type: "manual", Title: "O2",
		Content: "Observation two content", Project: "proj-b",
	})
	s.AddPrompt(memory.AddPromptParams{
		SessionID: "s1", Content: "A question", Project: "proj-a",
	})

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", stats.TotalSessions)
	}
	if stats.TotalObservations != 2 {
		t.Errorf("TotalObservations = %d, want 2", stats.TotalObservations)
	}
	if stats.TotalPrompts != 1 {
		t.Errorf("TotalPrompts = %d, want 1", stats.TotalPrompts)
	}
	if len(stats.Projects) != 2 {
		t.Errorf("Projects count = %d, want 2", len(stats.Projects))
	}
}

func TestStats_SoftDeletedExcluded(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "manual", Title: "Will delete",
		Content: "To be deleted from stats", Project: "proj",
	})
	s.DeleteObservation(id, false)

	stats, _ := s.Stats()
	if stats.TotalObservations != 0 {
		t.Errorf("TotalObservations = %d, want 0 (soft-deleted excluded)", stats.TotalObservations)
	}
}

// ─── FormatContext ──────────────────────────────────────────────────────────

func TestFormatContext_Empty(t *testing.T) {
	s := newTestStore(t)

	ctx, err := s.FormatContext("", "")
	if err != nil {
		t.Fatal(err)
	}
	if ctx != "" {
		t.Errorf("expected empty string for empty DB, got %q", ctx)
	}
}

func TestFormatContext_WithData(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "decision", Title: "Use PostgreSQL",
		Content: "ACID compliance needed", Project: "proj",
	})
	s.AddPrompt(memory.AddPromptParams{
		SessionID: "sess", Content: "How to set up DB?", Project: "proj",
	})

	ctx, err := s.FormatContext("proj", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx, "Memory from Previous Sessions") {
		t.Error("expected header in context")
	}
	if !strings.Contains(ctx, "PostgreSQL") {
		t.Error("expected observation content in context")
	}
	if !strings.Contains(ctx, "How to set up DB?") {
		t.Error("expected prompt content in context")
	}
}

// ─── Export / Import ─────────────────────────────────────────────────────────

func TestExport_Empty(t *testing.T) {
	s := newTestStore(t)

	data, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	if data.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", data.Version, "0.1.0")
	}
	if len(data.Sessions) != 0 {
		t.Errorf("Sessions: len = %d, want 0", len(data.Sessions))
	}
	if len(data.Observations) != 0 {
		t.Errorf("Observations: len = %d, want 0", len(data.Observations))
	}
}

func TestExportImport_RoundTrip(t *testing.T) {
	s1 := newTestStore(t)
	ensureSession(t, s1, "sess", "proj")

	s1.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "decision", Title: "Round trip test",
		Content: "Content survives export/import", Project: "proj", Scope: "project",
	})
	s1.AddPrompt(memory.AddPromptParams{
		SessionID: "sess", Content: "Prompt survives too", Project: "proj",
	})

	exported, err := s1.Export()
	if err != nil {
		t.Fatal(err)
	}

	if len(exported.Sessions) != 1 {
		t.Fatalf("exported sessions: %d, want 1", len(exported.Sessions))
	}
	if len(exported.Observations) != 1 {
		t.Fatalf("exported observations: %d, want 1", len(exported.Observations))
	}
	if len(exported.Prompts) != 1 {
		t.Fatalf("exported prompts: %d, want 1", len(exported.Prompts))
	}

	// Import into a fresh store
	s2 := newTestStore(t)
	result, err := s2.Import(exported)
	if err != nil {
		t.Fatalf("Import error: %v", err)
	}
	if result.SessionsImported != 1 {
		t.Errorf("SessionsImported = %d, want 1", result.SessionsImported)
	}
	if result.ObservationsImported != 1 {
		t.Errorf("ObservationsImported = %d, want 1", result.ObservationsImported)
	}
	if result.PromptsImported != 1 {
		t.Errorf("PromptsImported = %d, want 1", result.PromptsImported)
	}

	// Verify data is searchable in the new store
	stats, _ := s2.Stats()
	if stats.TotalSessions != 1 {
		t.Errorf("imported TotalSessions = %d, want 1", stats.TotalSessions)
	}
	if stats.TotalObservations != 1 {
		t.Errorf("imported TotalObservations = %d, want 1", stats.TotalObservations)
	}
}

func TestImport_DuplicateSessionsIgnored(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "existing-sess", "proj")

	data := &memory.ExportData{
		Version:    "0.1.0",
		ExportedAt: memory.Now(),
		Sessions: []memory.Session{
			{ID: "existing-sess", Project: "proj", Directory: "/tmp", StartedAt: memory.Now()},
			{ID: "new-sess", Project: "proj2", Directory: "/tmp2", StartedAt: memory.Now()},
		},
	}

	result, err := s.Import(data)
	if err != nil {
		t.Fatal(err)
	}
	// existing-sess is INSERT OR IGNORE → 0 rows affected, new-sess → 1
	if result.SessionsImported != 1 {
		t.Errorf("SessionsImported = %d, want 1 (existing skipped)", result.SessionsImported)
	}
}

// ─── Passive Capture ─────────────────────────────────────────────────────────

func TestExtractLearnings_English(t *testing.T) {
	text := `
## Summary
Did some work.

## Key Learnings:
1. Always validate input before processing to avoid injection attacks
2. Use WAL mode for better SQLite concurrency performance
3. Short text
`
	learnings := memory.ExtractLearnings(text)
	if len(learnings) != 2 {
		t.Errorf("len = %d, want 2 (short items filtered out)", len(learnings))
	}
	if len(learnings) > 0 && !strings.Contains(learnings[0], "validate input") {
		t.Errorf("first learning = %q", learnings[0])
	}
}

func TestExtractLearnings_Spanish(t *testing.T) {
	text := `
## Aprendizajes Clave:
- Siempre sanitizar las queries de FTS5 para evitar errores de sintaxis
- Usar topic_key para upserts de observaciones que evolucionan con el tiempo
`
	learnings := memory.ExtractLearnings(text)
	if len(learnings) != 2 {
		t.Errorf("len = %d, want 2", len(learnings))
	}
}

func TestExtractLearnings_BulletFallback(t *testing.T) {
	text := `
## Learnings:
- First learning that is long enough to pass the minimum length threshold
- Second learning that also exceeds the minimum character count easily
`
	learnings := memory.ExtractLearnings(text)
	if len(learnings) != 2 {
		t.Errorf("len = %d, want 2", len(learnings))
	}
}

func TestExtractLearnings_NoSection(t *testing.T) {
	text := "Just some random text without any learning section."
	learnings := memory.ExtractLearnings(text)
	if len(learnings) != 0 {
		t.Errorf("len = %d, want 0 (no learning section)", len(learnings))
	}
}

func TestPassiveCapture_SavesLearnings(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	result, err := s.PassiveCapture(memory.PassiveCaptureParams{
		SessionID: "sess",
		Content: `## Key Learnings:
1. Always wrap FTS5 queries in quotes to prevent syntax errors from special characters
2. Use WAL mode for SQLite to improve concurrent read performance significantly
`,
		Project: "proj",
		Source:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Extracted != 2 {
		t.Errorf("Extracted = %d, want 2", result.Extracted)
	}
	if result.Saved != 2 {
		t.Errorf("Saved = %d, want 2", result.Saved)
	}

	// Verify they're searchable
	obs, _ := s.RecentObservations("proj", "", 10)
	if len(obs) < 2 {
		t.Errorf("expected at least 2 observations after passive capture, got %d", len(obs))
	}
}

func TestPassiveCapture_DeduplicatesAcrossCalls(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	content := `## Key Learnings:
1. Always wrap FTS5 queries in quotes to prevent syntax errors from special characters
`
	r1, _ := s.PassiveCapture(memory.PassiveCaptureParams{
		SessionID: "sess", Content: content, Project: "proj",
	})
	r2, _ := s.PassiveCapture(memory.PassiveCaptureParams{
		SessionID: "sess", Content: content, Project: "proj",
	})

	if r1.Saved != 1 {
		t.Errorf("first call: Saved = %d, want 1", r1.Saved)
	}
	if r2.Duplicates != 1 {
		t.Errorf("second call: Duplicates = %d, want 1", r2.Duplicates)
	}
	if r2.Saved != 0 {
		t.Errorf("second call: Saved = %d, want 0", r2.Saved)
	}
}

func TestPassiveCapture_NoLearnings(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	result, err := s.PassiveCapture(memory.PassiveCaptureParams{
		SessionID: "sess",
		Content:   "Just regular text with no learning section at all.",
		Project:   "proj",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Extracted != 0 {
		t.Errorf("Extracted = %d, want 0", result.Extracted)
	}
	if result.Saved != 0 {
		t.Errorf("Saved = %d, want 0", result.Saved)
	}
}

// ─── Helper Functions ────────────────────────────────────────────────────────

func TestSuggestTopicKey(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		title   string
		content string
		want    string
	}{
		{
			name: "architecture type",
			typ:  "architecture", title: "Auth strategy", content: "",
			want: "architecture/auth-strategy",
		},
		{
			name: "bugfix type",
			typ:  "bugfix", title: "Memory leak fix", content: "",
			want: "bug/memory-leak-fix",
		},
		{
			name: "decision type",
			typ:  "decision", title: "Use PostgreSQL", content: "",
			want: "decision/use-postgresql",
		},
		{
			name: "pattern type",
			typ:  "pattern", title: "Repository pattern", content: "",
			want: "pattern/repository-pattern",
		},
		{
			name: "config type",
			typ:  "config", title: "Docker setup", content: "",
			want: "config/docker-setup",
		},
		{
			name: "empty title uses content",
			typ:  "manual", title: "", content: "Some descriptive content about auth",
			want: "topic/some-descriptive-content-about-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := memory.SuggestTopicKey(tt.typ, tt.title, tt.content)
			if got != tt.want {
				t.Errorf("SuggestTopicKey(%q, %q, %q) = %q, want %q",
					tt.typ, tt.title, tt.content, got, tt.want)
			}
		})
	}
}

func TestClassifyTool(t *testing.T) {
	tests := []struct {
		tool string
		want string
	}{
		{"write", "file_change"},
		{"edit", "file_change"},
		{"patch", "file_change"},
		{"bash", "command"},
		{"read", "file_read"},
		{"view", "file_read"},
		{"grep", "search"},
		{"glob", "search"},
		{"ls", "search"},
		{"unknown", "tool_use"},
		{"", "tool_use"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := memory.ClassifyTool(tt.tool)
			if got != tt.want {
				t.Errorf("ClassifyTool(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := memory.Truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestNow_ReturnsUTCFormat(t *testing.T) {
	now := memory.Now()
	_, err := time.Parse("2006-01-02 15:04:05", now)
	if err != nil {
		t.Errorf("Now() = %q, not in expected format: %v", now, err)
	}
}

// ─── Edge Cases ─────────────────────────────────────────────────────────────

func TestTopicKeyUpsert_DifferentProjectNoConflict(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj-a")
	ensureSession(t, s, "sess2", "proj-b")

	id1, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "architecture", Title: "Auth v1",
		Content: "Project A auth", Project: "proj-a", TopicKey: "architecture/auth",
	})
	id2, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess2", Type: "architecture", Title: "Auth v1",
		Content: "Project B auth", Project: "proj-b", TopicKey: "architecture/auth",
	})

	if id1 == id2 {
		t.Error("same topic_key in different projects should create separate observations")
	}
}

func TestTopicKeyUpsert_DifferentScopeNoConflict(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	id1, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "pattern", Title: "Naming conv",
		Content: "Project-level naming convention details", Project: "proj",
		Scope: "project", TopicKey: "pattern/naming",
	})
	id2, _ := s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "pattern", Title: "Naming conv personal",
		Content: "Personal naming convention preferences", Project: "proj",
		Scope: "personal", TopicKey: "pattern/naming",
	})

	if id1 == id2 {
		t.Error("same topic_key in different scopes should create separate observations")
	}
}

func TestSearch_SpecialCharactersSanitized(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sess", "proj")

	s.AddObservation(memory.AddObservationParams{
		SessionID: "sess", Type: "manual", Title: "Normal obs",
		Content: "Some normal searchable content", Project: "proj",
	})

	// These would crash FTS5 without sanitization
	dangerousQueries := []string{
		"fix auth bug",
		"hello world",
		"test*query",
		"(broken)",
		"OR AND NOT",
	}

	for _, q := range dangerousQueries {
		t.Run(q, func(t *testing.T) {
			_, err := s.Search(q, memory.SearchOptions{})
			if err != nil {
				t.Errorf("Search(%q) should not error: %v", q, err)
			}
		})
	}
}

func TestExtractLearnings_MarkdownStripped(t *testing.T) {
	text := `## Key Learnings:
1. Always use **bold** and *italic* and ` + "`code`" + ` formatting that should be stripped cleanly from learnings
`
	learnings := memory.ExtractLearnings(text)
	if len(learnings) != 1 {
		t.Fatalf("len = %d, want 1", len(learnings))
	}
	if strings.Contains(learnings[0], "**") {
		t.Errorf("markdown not stripped: %q", learnings[0])
	}
	if strings.Contains(learnings[0], "`") {
		t.Errorf("inline code not stripped: %q", learnings[0])
	}
}
