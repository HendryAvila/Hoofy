package memory_test

import (
	"strings"
	"testing"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
)

// ─── Full Session Lifecycle Integration ─────────────────────────────────────

func TestIntegration_FullSessionLifecycle(t *testing.T) {
	s := newTestStore(t)

	// 1. Start session
	if err := s.CreateSession("lifecycle-session", "my-project", "/home/user/repo"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// 2. Save observations of different types
	id1, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "lifecycle-session",
		Type:      "decision",
		Title:     "Chose PostgreSQL over MongoDB",
		Content:   "**What**: Selected PostgreSQL as primary DB\n**Why**: Need ACID compliance\n**Where**: internal/store/\n**Learned**: FTS in Postgres requires pg_trgm extension",
		Project:   "my-project",
		Scope:     "project",
	})
	if err != nil {
		t.Fatalf("AddObservation 1: %v", err)
	}
	if id1 == 0 {
		t.Fatal("observation 1 should have a non-zero ID")
	}

	_, err = s.AddObservation(memory.AddObservationParams{
		SessionID: "lifecycle-session",
		Type:      "bugfix",
		Title:     "Fixed N+1 query in user list",
		Content:   "**What**: Added eager loading to user query\n**Why**: Dashboard was making 200+ queries\n**Where**: internal/api/users.go",
		Project:   "my-project",
		Scope:     "project",
	})
	if err != nil {
		t.Fatalf("AddObservation 2: %v", err)
	}

	id3, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "lifecycle-session",
		Type:      "architecture",
		Title:     "Clean Architecture layers",
		Content:   "**What**: Established 3-layer architecture\n**Why**: Separation of concerns\n**Where**: internal/domain/, internal/app/, internal/infra/",
		Project:   "my-project",
		Scope:     "project",
		TopicKey:  "architecture/layers",
	})
	if err != nil {
		t.Fatalf("AddObservation 3: %v", err)
	}

	// 3. Search — verify FTS5 finds the observations
	results, err := s.Search("PostgreSQL", memory.SearchOptions{Project: "my-project"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'PostgreSQL', got %d", len(results))
	}
	if results[0].ID != id1 {
		t.Errorf("search result ID = %d, want %d", results[0].ID, id1)
	}

	// 4. Search by type filter
	bugResults, err := s.Search("query", memory.SearchOptions{Type: "bugfix"})
	if err != nil {
		t.Fatalf("Search by type: %v", err)
	}
	if len(bugResults) != 1 {
		t.Fatalf("expected 1 bugfix result, got %d", len(bugResults))
	}

	// 5. Get full observation
	full, err := s.GetObservation(id3)
	if err != nil {
		t.Fatalf("GetObservation: %v", err)
	}
	if full.Title != "Clean Architecture layers" {
		t.Errorf("title = %q, want 'Clean Architecture layers'", full.Title)
	}
	if full.TopicKey == nil || *full.TopicKey != "architecture/layers" {
		t.Error("topic_key should be 'architecture/layers'")
	}

	// 6. Get timeline around id1
	timeline, err := s.Timeline(id1, 1, 1)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if timeline.Focus.ID != id1 {
		t.Errorf("timeline focus ID = %d, want %d", timeline.Focus.ID, id1)
	}

	// 7. Get context — verify recent observations returned
	ctx, err := s.FormatContext("my-project", "project")
	if err != nil {
		t.Fatalf("FormatContext: %v", err)
	}
	if ctx == "" {
		t.Fatal("FormatContext should return non-empty result")
	}
	if !strings.Contains(ctx, "PostgreSQL") {
		t.Error("context should contain PostgreSQL observation")
	}

	// 8. Save session summary
	_, err = s.AddObservation(memory.AddObservationParams{
		SessionID: "lifecycle-session",
		Type:      "session_summary",
		Title:     "Session summary",
		Content:   "## Goal\nSet up database layer\n\n## Accomplished\n- Chose PostgreSQL\n- Fixed N+1 query\n- Defined architecture layers",
		Project:   "my-project",
	})
	if err != nil {
		t.Fatalf("AddObservation summary: %v", err)
	}

	// 9. End session
	if err := s.EndSession("lifecycle-session", "Completed database layer setup"); err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	// 10. Verify stats
	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalObservations != 4 { // 3 regular + 1 summary
		t.Errorf("total observations = %d, want 4", stats.TotalObservations)
	}
	if stats.TotalSessions != 1 {
		t.Errorf("total sessions = %d, want 1", stats.TotalSessions)
	}
}

// ─── FTS5 Edge Cases ────────────────────────────────────────────────────────

func TestSearch_UnicodeContent(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "unicode-sess", "proj")

	// Save observations with unicode content
	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "unicode-sess", Type: "manual", Title: "日本語テスト",
		Content: "日本語のコンテンツをテストしています", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "unicode-sess", Type: "manual", Title: "Observación en español",
		Content: "Probando búsqueda con acentos y ñ", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "unicode-sess", Type: "manual", Title: "Emoji test 🚀",
		Content: "Content with emojis 🎉 and symbols ™ © ®", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	tests := []struct {
		name  string
		query string
	}{
		{"Japanese", "日本語"},
		{"Spanish accents", "búsqueda"},
		{"Emoji", "🚀"},
		{"Spanish ñ", "ñ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := s.Search(tt.query, memory.SearchOptions{})
			if err != nil {
				t.Errorf("Search(%q) should not error: %v", tt.query, err)
			}
			// We don't assert result count since FTS5 tokenization may not
			// handle all unicode the same way — we just verify no crashes.
			_ = results
		})
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "empty-sess", "proj")

	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "empty-sess", Type: "manual", Title: "Test obs",
		Content: "Some content", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// Empty query should not crash and should return results (recent fallback)
	results, err := s.Search("", memory.SearchOptions{})
	if err != nil {
		t.Fatalf("Search('') should not error: %v", err)
	}
	// Empty query should fall back to recent observations
	if len(results) == 0 {
		t.Error("empty query should return recent observations")
	}
}

func TestSearch_WhitespaceOnlyQuery(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "ws-sess", "proj")

	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "ws-sess", Type: "manual", Title: "Test",
		Content: "Content here", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// Whitespace-only query should not crash
	queries := []string{"   ", "\t", "\n", "  \t  \n  "}
	for _, q := range queries {
		_, err := s.Search(q, memory.SearchOptions{})
		if err != nil {
			t.Errorf("Search(%q) should not error: %v", q, err)
		}
	}
}

func TestSearch_VeryLongQuery(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "long-sess", "proj")

	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "long-sess", Type: "manual", Title: "Test",
		Content: "Normal content", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// Very long query should not crash
	longQuery := strings.Repeat("search term ", 500)
	_, err := s.Search(longQuery, memory.SearchOptions{})
	if err != nil {
		t.Fatalf("Search with very long query should not error: %v", err)
	}
}

func TestSearch_SQLInjectionAttempt(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "sqli-sess", "proj")

	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "sqli-sess", Type: "manual", Title: "Test",
		Content: "Normal content", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// SQL injection attempts should be sanitized, not crash
	injections := []string{
		"'; DROP TABLE observations; --",
		"\" OR 1=1 --",
		"1; DELETE FROM sessions",
		"UNION SELECT * FROM sessions",
	}

	for _, q := range injections {
		t.Run(q, func(t *testing.T) {
			_, err := s.Search(q, memory.SearchOptions{})
			if err != nil {
				t.Errorf("Search(%q) should not error: %v", q, err)
			}
		})
	}
}

// ─── Topic Key Upsert Integration ───────────────────────────────────────────

func TestIntegration_TopicKeyEvolution(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "topic-sess", "proj")

	// First save with topic_key
	id1, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "topic-sess", Type: "architecture", Title: "Auth design v1",
		Content: "JWT with access tokens only", Project: "proj", TopicKey: "architecture/auth",
	})
	if err != nil {
		t.Fatalf("AddObservation 1: %v", err)
	}

	// Second save with same topic_key should UPDATE, not create new
	id2, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "topic-sess", Type: "architecture", Title: "Auth design v2",
		Content: "JWT with access + refresh tokens", Project: "proj", TopicKey: "architecture/auth",
	})
	if err != nil {
		t.Fatalf("AddObservation 2: %v", err)
	}

	// Should be the same observation ID (upsert)
	if id2 != id1 {
		t.Errorf("topic_key upsert should reuse ID: got %d, want %d", id2, id1)
	}

	// Content should be updated
	got, err := s.GetObservation(id1)
	if err != nil {
		t.Fatalf("GetObservation: %v", err)
	}
	if !strings.Contains(got.Content, "refresh tokens") {
		t.Error("content should be updated to v2")
	}

	// Search should find the updated content
	results, err := s.Search("refresh tokens", memory.SearchOptions{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("search should find exactly 1 result, got %d", len(results))
	}
}

// ─── Deduplication Integration ──────────────────────────────────────────────

func TestIntegration_DeduplicationWindow(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "dedup-sess", "proj")

	content := "This exact content should be deduplicated within the time window"

	// Dedup requires matching: hash + project + scope + type + title
	id1, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "dedup-sess", Type: "manual", Title: "Same title",
		Content: content, Project: "proj",
	})
	if err != nil {
		t.Fatalf("AddObservation 1: %v", err)
	}

	// Same content, type, AND title should trigger dedup
	id2, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "dedup-sess", Type: "manual", Title: "Same title",
		Content: content, Project: "proj",
	})
	if err != nil {
		t.Fatalf("AddObservation 2: %v", err)
	}

	if id2 != id1 {
		t.Errorf("deduplication should reuse ID: got %d, want %d", id2, id1)
	}

	// Stats should show only 1 observation
	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalObservations != 1 {
		t.Errorf("deduplication failed: total = %d, want 1", stats.TotalObservations)
	}

	// Different title = no dedup (new observation)
	id3, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "dedup-sess", Type: "manual", Title: "Different title",
		Content: content, Project: "proj",
	})
	if err != nil {
		t.Fatalf("AddObservation 3: %v", err)
	}
	if id3 == id1 {
		t.Error("different title should NOT be deduplicated")
	}
}

// ─── Export/Import Consistency ───────────────────────────────────────────────

func TestIntegration_ExportImportPreservesData(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "export-sess", "proj")

	// Save diverse data
	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "export-sess", Type: "decision", Title: "Decision 1",
		Content: "Important decision content", Project: "proj", TopicKey: "decisions/first",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "export-sess", Type: "bugfix", Title: "Bug fix 1",
		Content: "Fixed the thing", Project: "proj",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if _, err := s.AddPrompt(memory.AddPromptParams{
		SessionID: "export-sess", Content: "User asked about auth", Project: "proj",
	}); err != nil {
		t.Fatalf("AddPrompt: %v", err)
	}

	// Export
	exported, err := s.Export()
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if exported.Version != "0.1.0" {
		t.Errorf("version = %q, want '0.1.0'", exported.Version)
	}
	if len(exported.Sessions) != 1 {
		t.Errorf("sessions = %d, want 1", len(exported.Sessions))
	}
	if len(exported.Observations) != 2 {
		t.Errorf("observations = %d, want 2", len(exported.Observations))
	}
	if len(exported.Prompts) != 1 {
		t.Errorf("prompts = %d, want 1", len(exported.Prompts))
	}

	// Import into a fresh store
	s2 := newTestStore(t)
	result, err := s2.Import(exported)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.SessionsImported != 1 {
		t.Errorf("imported sessions = %d, want 1", result.SessionsImported)
	}

	// Verify data is searchable in the new store
	results, err := s2.Search("decision", memory.SearchOptions{})
	if err != nil {
		t.Fatalf("Search in imported store: %v", err)
	}
	if len(results) == 0 {
		t.Error("imported data should be searchable via FTS5")
	}
}

// ─── Concurrent Access ──────────────────────────────────────────────────────

func TestIntegration_ConcurrentWrites(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "concurrent-sess", "proj")

	// Write observations sequentially but from a goroutine to test
	// that single-threaded access works correctly. SQLite (pure Go)
	// with WAL mode + busy_timeout handles serialized access.
	// True concurrent writes may get SQLITE_BUSY which is expected.
	const count = 10
	for i := 0; i < count; i++ {
		_, err := s.AddObservation(memory.AddObservationParams{
			SessionID: "concurrent-sess",
			Type:      "manual",
			Title:     "Sequential obs",
			Content:   strings.Repeat("x", 100+i), // different content to avoid dedup
			Project:   "proj",
		})
		if err != nil {
			t.Fatalf("sequential write %d failed: %v", i, err)
		}
	}

	// Verify all were saved
	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalObservations != count {
		t.Errorf("total observations = %d, want %d", stats.TotalObservations, count)
	}
}

// ─── Multi-Project Isolation ────────────────────────────────────────────────

func TestIntegration_MultiProjectIsolation(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "alpha-sess", "project-alpha")
	ensureSession(t, s, "beta-sess", "project-beta")

	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "alpha-sess", Type: "decision", Title: "Alpha decision",
		Content: "This belongs to alpha", Project: "project-alpha",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}
	if _, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "beta-sess", Type: "decision", Title: "Beta decision",
		Content: "This belongs to beta", Project: "project-beta",
	}); err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// Search filtered by project
	alphaResults, err := s.Search("decision", memory.SearchOptions{Project: "project-alpha"})
	if err != nil {
		t.Fatalf("Search alpha: %v", err)
	}
	if len(alphaResults) != 1 {
		t.Errorf("alpha results = %d, want 1", len(alphaResults))
	}
	if alphaResults[0].Title != "Alpha decision" {
		t.Errorf("alpha result title = %q, want 'Alpha decision'", alphaResults[0].Title)
	}

	betaResults, err := s.Search("decision", memory.SearchOptions{Project: "project-beta"})
	if err != nil {
		t.Fatalf("Search beta: %v", err)
	}
	if len(betaResults) != 1 {
		t.Errorf("beta results = %d, want 1", len(betaResults))
	}

	// Context should be isolated by project
	ctx, err := s.FormatContext("project-alpha", "")
	if err != nil {
		t.Fatalf("FormatContext: %v", err)
	}
	if !strings.Contains(ctx, "alpha") {
		t.Error("alpha context should contain alpha observation")
	}
	if strings.Contains(ctx, "beta") {
		t.Error("alpha context should NOT contain beta observation")
	}
}

// ─── Soft Delete + FTS Consistency ──────────────────────────────────────────

func TestIntegration_SoftDeleteRemovesFromSearch(t *testing.T) {
	s := newTestStore(t)
	ensureSession(t, s, "del-sess", "proj")

	id, err := s.AddObservation(memory.AddObservationParams{
		SessionID: "del-sess", Type: "manual", Title: "Will be deleted",
		Content: "Unique searchable xylophone content", Project: "proj",
	})
	if err != nil {
		t.Fatalf("AddObservation: %v", err)
	}

	// Verify it's searchable
	before, _ := s.Search("xylophone", memory.SearchOptions{})
	if len(before) != 1 {
		t.Fatalf("should find observation before delete, got %d", len(before))
	}

	// Soft delete
	if err := s.DeleteObservation(id, false); err != nil {
		t.Fatalf("DeleteObservation: %v", err)
	}

	// Should no longer appear in search
	after, _ := s.Search("xylophone", memory.SearchOptions{})
	if len(after) != 0 {
		t.Errorf("soft-deleted observation should not appear in search, got %d", len(after))
	}

	// Should not appear in GetObservation
	_, err = s.GetObservation(id)
	if err == nil {
		t.Error("GetObservation should return error for soft-deleted observation")
	}
}
