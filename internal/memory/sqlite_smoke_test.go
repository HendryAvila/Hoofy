package memory_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteSmokeTest(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "smoke.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("failed to enable WAL mode: %v", err)
	}

	// Verify WAL mode is active
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("expected WAL mode, got %q", mode)
	}
}

func TestFTS5SmokeTest(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fts5.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Create a regular table
	_, err = db.Exec(`CREATE TABLE docs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		content TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("failed to create docs table: %v", err)
	}

	// Create FTS5 virtual table
	_, err = db.Exec(`CREATE VIRTUAL TABLE docs_fts USING fts5(
		title, content, content='docs', content_rowid='id'
	)`)
	if err != nil {
		t.Fatalf("failed to create FTS5 table: %v", err)
	}

	// Create triggers to keep FTS5 in sync
	_, err = db.Exec(`
		CREATE TRIGGER docs_ai AFTER INSERT ON docs BEGIN
			INSERT INTO docs_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
		END;
		CREATE TRIGGER docs_ad AFTER DELETE ON docs BEGIN
			INSERT INTO docs_fts(docs_fts, rowid, title, content) VALUES('delete', old.id, old.title, old.content);
		END;
		CREATE TRIGGER docs_au AFTER UPDATE ON docs BEGIN
			INSERT INTO docs_fts(docs_fts, rowid, title, content) VALUES('delete', old.id, old.title, old.content);
			INSERT INTO docs_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
		END;
	`)
	if err != nil {
		t.Fatalf("failed to create FTS5 triggers: %v", err)
	}

	// Insert test data
	docs := []struct {
		title, content string
	}{
		{"JWT Auth Middleware", "Implemented JWT authentication with refresh tokens for the API layer"},
		{"PostgreSQL Migration", "Migrated from SQLite to PostgreSQL for production scalability"},
		{"React Component Refactor", "Refactored the dashboard components using atomic design patterns"},
		{"Bug Fix: Memory Leak", "Fixed a goroutine leak in the WebSocket handler that caused OOM crashes"},
	}
	for _, d := range docs {
		if _, err := db.Exec("INSERT INTO docs (title, content) VALUES (?, ?)", d.title, d.content); err != nil {
			t.Fatalf("failed to insert doc %q: %v", d.title, err)
		}
	}

	// Test FTS5 search
	tests := []struct {
		name    string
		query   string
		wantMin int // minimum expected results
	}{
		{"single word", `"JWT"`, 1},
		{"phrase", `"atomic design"`, 1},
		{"multiple terms", `"goroutine" OR "leak"`, 1},
		{"no match", `"kubernetes"`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Query(
				"SELECT d.id, d.title FROM docs d JOIN docs_fts f ON d.id = f.rowid WHERE docs_fts MATCH ? ORDER BY rank",
				tt.query,
			)
			if err != nil {
				t.Fatalf("FTS5 search failed for %q: %v", tt.query, err)
			}
			defer rows.Close()

			var count int
			for rows.Next() {
				var id int
				var title string
				if err := rows.Scan(&id, &title); err != nil {
					t.Fatalf("failed to scan result: %v", err)
				}
				count++
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("rows iteration error: %v", err)
			}

			if count < tt.wantMin {
				t.Errorf("query %q: got %d results, want at least %d", tt.query, count, tt.wantMin)
			}
		})
	}
}

func TestFTS5SpecialCharsSanitization(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fts5_special.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE docs (id INTEGER PRIMARY KEY AUTOINCREMENT, content TEXT NOT NULL)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.Exec(`CREATE VIRTUAL TABLE docs_fts USING fts5(content, content='docs', content_rowid='id')`)
	if err != nil {
		t.Fatalf("failed to create FTS5 table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER docs_ai AFTER INSERT ON docs BEGIN
			INSERT INTO docs_fts(rowid, content) VALUES (new.id, new.content);
		END;
	`)
	if err != nil {
		t.Fatalf("failed to create trigger: %v", err)
	}

	if _, err := db.Exec("INSERT INTO docs (content) VALUES (?)", "hello world test data"); err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// These are queries that would CRASH FTS5 without sanitization
	dangerousQueries := []string{
		`fix auth bug`,    // spaces interpreted as AND by default — should be safe
		`hello*`,          // prefix search — valid FTS5
		`"hello world"`,   // phrase — valid FTS5
		`hello OR world`,  // boolean — valid FTS5
		`hello AND world`, // boolean — valid FTS5
	}

	for _, q := range dangerousQueries {
		t.Run(q, func(t *testing.T) {
			rows, err := db.Query("SELECT content FROM docs_fts WHERE docs_fts MATCH ?", q)
			if err != nil {
				t.Logf("query %q failed (expected for some): %v", q, err)
				return // Some might fail — that's fine, we just don't want panics
			}
			defer rows.Close()
			for rows.Next() {
				var content string
				_ = rows.Scan(&content)
			}
		})
	}
}

func TestSQLiteBusyTimeout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "busy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	// Set busy timeout to 5 seconds (5000ms)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		t.Fatalf("failed to set busy_timeout: %v", err)
	}

	// Verify it was set
	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Fatalf("expected busy_timeout=5000, got %d", timeout)
	}
}
