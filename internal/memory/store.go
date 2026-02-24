// Package memory implements the persistent memory engine for SDD-Hoffy.
//
// It uses SQLite with FTS5 full-text search to store and retrieve
// observations from AI coding sessions. Adapted from Engram's store
// (github.com/Gentleman-Programming/engram) with full feature parity.
package memory

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// openDB is a package-level var to allow test injection.
var openDB = sql.Open

// ─── Types ───────────────────────────────────────────────────────────────────

// Session represents a coding session with start/end timestamps.
type Session struct {
	ID        string  `json:"id"`
	Project   string  `json:"project"`
	Directory string  `json:"directory"`
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at,omitempty"`
	Summary   *string `json:"summary,omitempty"`
}

// Observation represents a single memory entry — a decision, pattern, bug fix, etc.
type Observation struct {
	ID             int64   `json:"id"`
	SessionID      string  `json:"session_id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ToolName       *string `json:"tool_name,omitempty"`
	Project        *string `json:"project,omitempty"`
	Scope          string  `json:"scope"`
	TopicKey       *string `json:"topic_key,omitempty"`
	RevisionCount  int     `json:"revision_count"`
	DuplicateCount int     `json:"duplicate_count"`
	LastSeenAt     *string `json:"last_seen_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
}

// SearchResult embeds an Observation with FTS5 rank score.
type SearchResult struct {
	Observation
	Rank float64 `json:"rank"`
}

// Relation represents a typed directional edge between two observations.
type Relation struct {
	ID        int64  `json:"id"`
	FromID    int64  `json:"from_id"`
	ToID      int64  `json:"to_id"`
	Type      string `json:"type"`
	Note      string `json:"note,omitempty"`
	CreatedAt string `json:"created_at"`
}

// AddRelationParams holds input for creating a new relation.
type AddRelationParams struct {
	FromID        int64  `json:"from_id"`
	ToID          int64  `json:"to_id"`
	Type          string `json:"type"`
	Note          string `json:"note,omitempty"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
}

// ContextNode represents one node in a graph traversal result.
type ContextNode struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Project      string `json:"project,omitempty"`
	CreatedAt    string `json:"created_at"`
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"` // "outgoing" or "incoming"
	Note         string `json:"note,omitempty"`
	Depth        int    `json:"depth"`
}

// ContextResult holds the full graph traversal output.
type ContextResult struct {
	Root       Observation   `json:"root"`
	Connected  []ContextNode `json:"connected"`
	TotalNodes int           `json:"total_nodes"`
	MaxDepth   int           `json:"max_depth"`
}

// SessionSummary is a compact view of a session with observation count.
type SessionSummary struct {
	ID               string  `json:"id"`
	Project          string  `json:"project"`
	StartedAt        string  `json:"started_at"`
	EndedAt          *string `json:"ended_at,omitempty"`
	Summary          *string `json:"summary,omitempty"`
	ObservationCount int     `json:"observation_count"`
}

// Stats holds aggregate memory statistics.
type Stats struct {
	TotalSessions     int      `json:"total_sessions"`
	TotalObservations int      `json:"total_observations"`
	TotalPrompts      int      `json:"total_prompts"`
	Projects          []string `json:"projects"`
}

// TimelineEntry is an observation in a timeline with a focus flag.
type TimelineEntry struct {
	ID             int64   `json:"id"`
	SessionID      string  `json:"session_id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ToolName       *string `json:"tool_name,omitempty"`
	Project        *string `json:"project,omitempty"`
	Scope          string  `json:"scope"`
	TopicKey       *string `json:"topic_key,omitempty"`
	RevisionCount  int     `json:"revision_count"`
	DuplicateCount int     `json:"duplicate_count"`
	LastSeenAt     *string `json:"last_seen_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	IsFocus        bool    `json:"is_focus"`
}

// TimelineResult holds the progressive disclosure result around a focus observation.
type TimelineResult struct {
	Focus        Observation     `json:"focus"`
	Before       []TimelineEntry `json:"before"`
	After        []TimelineEntry `json:"after"`
	SessionInfo  *Session        `json:"session_info"`
	TotalInRange int             `json:"total_in_range"`
}

// SearchOptions holds filters for FTS5 search queries.
type SearchOptions struct {
	Type    string `json:"type,omitempty"`
	Project string `json:"project,omitempty"`
	Scope   string `json:"scope,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// AddObservationParams holds the input for creating a new observation.
type AddObservationParams struct {
	SessionID string `json:"session_id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	ToolName  string `json:"tool_name,omitempty"`
	Project   string `json:"project,omitempty"`
	Scope     string `json:"scope,omitempty"`
	TopicKey  string `json:"topic_key,omitempty"`
}

// UpdateObservationParams holds partial update fields for an observation.
type UpdateObservationParams struct {
	Type     *string `json:"type,omitempty"`
	Title    *string `json:"title,omitempty"`
	Content  *string `json:"content,omitempty"`
	Project  *string `json:"project,omitempty"`
	Scope    *string `json:"scope,omitempty"`
	TopicKey *string `json:"topic_key,omitempty"`
}

// Prompt represents a saved user prompt.
type Prompt struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
	CreatedAt string `json:"created_at"`
}

// AddPromptParams holds the input for saving a user prompt.
type AddPromptParams struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
}

// ExportData is the full serializable dump of the memory database.
type ExportData struct {
	Version      string        `json:"version"`
	ExportedAt   string        `json:"exported_at"`
	Sessions     []Session     `json:"sessions"`
	Observations []Observation `json:"observations"`
	Prompts      []Prompt      `json:"prompts"`
}

// ImportResult holds counts of imported records.
type ImportResult struct {
	SessionsImported     int `json:"sessions_imported"`
	ObservationsImported int `json:"observations_imported"`
	PromptsImported      int `json:"prompts_imported"`
}

// PassiveCaptureParams holds the input for passive memory capture.
type PassiveCaptureParams struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
	Source    string `json:"source,omitempty"`
}

// PassiveCaptureResult holds the output of passive memory capture.
type PassiveCaptureResult struct {
	Extracted  int `json:"extracted"`
	Saved      int `json:"saved"`
	Duplicates int `json:"duplicates"`
}

// ─── Config ──────────────────────────────────────────────────────────────────

// Config holds memory store configuration.
type Config struct {
	DataDir              string
	MaxObservationLength int
	MaxContextResults    int
	MaxSearchResults     int
	DedupeWindow         time.Duration
}

// DefaultConfig returns the default configuration for the memory store.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:              filepath.Join(home, ".hoofy"),
		MaxObservationLength: 2000,
		MaxContextResults:    20,
		MaxSearchResults:     20,
		DedupeWindow:         15 * time.Minute,
	}
}

// ─── Store ───────────────────────────────────────────────────────────────────

// Store is the persistent memory engine backed by SQLite + FTS5.
type Store struct {
	db    *sql.DB
	cfg   Config
	hooks storeHooks
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

type sqlRowScanner struct {
	rows *sql.Rows
}

func (r sqlRowScanner) Next() bool             { return r.rows.Next() }
func (r sqlRowScanner) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r sqlRowScanner) Err() error             { return r.rows.Err() }
func (r sqlRowScanner) Close() error           { return r.rows.Close() }

type storeHooks struct {
	exec    func(db execer, query string, args ...any) (sql.Result, error)
	query   func(db queryer, query string, args ...any) (*sql.Rows, error)
	queryIt func(db queryer, query string, args ...any) (rowScanner, error)
	beginTx func(db *sql.DB) (*sql.Tx, error)
	commit  func(tx *sql.Tx) error
}

func defaultStoreHooks() storeHooks {
	return storeHooks{
		exec: func(db execer, query string, args ...any) (sql.Result, error) {
			return db.Exec(query, args...)
		},
		query: func(db queryer, query string, args ...any) (*sql.Rows, error) {
			return db.Query(query, args...)
		},
		queryIt: func(db queryer, query string, args ...any) (rowScanner, error) {
			rows, err := db.Query(query, args...)
			if err != nil {
				return nil, err
			}
			return sqlRowScanner{rows: rows}, nil
		},
		beginTx: func(db *sql.DB) (*sql.Tx, error) {
			return db.Begin()
		},
		commit: func(tx *sql.Tx) error {
			return tx.Commit()
		},
	}
}

func (s *Store) execHook(db execer, query string, args ...any) (sql.Result, error) {
	if s.hooks.exec != nil {
		return s.hooks.exec(db, query, args...)
	}
	return db.Exec(query, args...)
}

func (s *Store) queryHook(db queryer, query string, args ...any) (*sql.Rows, error) {
	if s.hooks.query != nil {
		return s.hooks.query(db, query, args...)
	}
	return db.Query(query, args...)
}

func (s *Store) queryItHook(db queryer, query string, args ...any) (rowScanner, error) {
	if s.hooks.queryIt != nil {
		return s.hooks.queryIt(db, query, args...)
	}
	rows, err := s.queryHook(db, query, args...)
	if err != nil {
		return nil, err
	}
	return sqlRowScanner{rows: rows}, nil
}

func (s *Store) beginTxHook() (*sql.Tx, error) {
	if s.hooks.beginTx != nil {
		return s.hooks.beginTx(s.db)
	}
	return s.db.Begin()
}

func (s *Store) commitHook(tx *sql.Tx) error {
	if s.hooks.commit != nil {
		return s.hooks.commit(tx)
	}
	return tx.Commit()
}

// New creates a new Store with the given configuration.
// It creates the data directory if needed, opens SQLite with WAL mode,
// and runs migrations.
func New(cfg Config) (*Store, error) {
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("memory: create data dir: %w", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "memory.db")
	db, err := openDB("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("memory: open database: %w", err)
	}

	// SQLite performance pragmas
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("memory: pragma %q: %w", p, err)
		}
	}

	s := &Store{db: db, cfg: cfg, hooks: defaultStoreHooks()}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("memory: migration: %w", err)
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ─── Migrations ──────────────────────────────────────────────────────────────

func (s *Store) migrate() error {
	schema := `
		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT PRIMARY KEY,
			project    TEXT NOT NULL,
			directory  TEXT NOT NULL,
			started_at TEXT NOT NULL DEFAULT (datetime('now')),
			ended_at   TEXT,
			summary    TEXT
		);

		CREATE TABLE IF NOT EXISTS observations (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id      TEXT    NOT NULL,
			type            TEXT    NOT NULL,
			title           TEXT    NOT NULL,
			content         TEXT    NOT NULL,
			tool_name       TEXT,
			project         TEXT,
			scope           TEXT    NOT NULL DEFAULT 'project',
			topic_key       TEXT,
			normalized_hash TEXT,
			revision_count  INTEGER NOT NULL DEFAULT 1,
			duplicate_count INTEGER NOT NULL DEFAULT 1,
			last_seen_at    TEXT,
			created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
			updated_at      TEXT    NOT NULL DEFAULT (datetime('now')),
			deleted_at      TEXT,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);

		CREATE INDEX IF NOT EXISTS idx_obs_session  ON observations(session_id);
		CREATE INDEX IF NOT EXISTS idx_obs_type     ON observations(type);
		CREATE INDEX IF NOT EXISTS idx_obs_project  ON observations(project);
		CREATE INDEX IF NOT EXISTS idx_obs_created  ON observations(created_at DESC);

		CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
			title,
			content,
			tool_name,
			type,
			project,
			content='observations',
			content_rowid='id'
		);

		CREATE TABLE IF NOT EXISTS user_prompts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT    NOT NULL,
			content    TEXT    NOT NULL,
			project    TEXT,
			created_at TEXT    NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);

		CREATE INDEX IF NOT EXISTS idx_prompts_session ON user_prompts(session_id);
		CREATE INDEX IF NOT EXISTS idx_prompts_project ON user_prompts(project);
		CREATE INDEX IF NOT EXISTS idx_prompts_created ON user_prompts(created_at DESC);

		CREATE VIRTUAL TABLE IF NOT EXISTS prompts_fts USING fts5(
			content,
			project,
			content='user_prompts',
			content_rowid='id'
		);
	`
	if _, err := s.execHook(s.db, schema); err != nil {
		return err
	}

	// Additional indexes for advanced features
	if _, err := s.execHook(s.db, `
		CREATE INDEX IF NOT EXISTS idx_obs_scope   ON observations(scope);
		CREATE INDEX IF NOT EXISTS idx_obs_topic   ON observations(topic_key, project, scope, updated_at DESC);
		CREATE INDEX IF NOT EXISTS idx_obs_deleted  ON observations(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_obs_dedupe   ON observations(normalized_hash, project, scope, type, title, created_at DESC);
	`); err != nil {
		return err
	}

	// Relations table — knowledge graph edges between observations.
	// Uses CREATE TABLE/INDEX IF NOT EXISTS for non-destructive migration:
	// existing databases gain the table on upgrade without data loss.
	if _, err := s.execHook(s.db, `
		CREATE TABLE IF NOT EXISTS relations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			from_id    INTEGER NOT NULL,
			to_id      INTEGER NOT NULL,
			type       TEXT    NOT NULL DEFAULT 'relates_to',
			note       TEXT,
			created_at TEXT    NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (from_id) REFERENCES observations(id) ON DELETE CASCADE,
			FOREIGN KEY (to_id)   REFERENCES observations(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_rel_from   ON relations(from_id);
		CREATE INDEX IF NOT EXISTS idx_rel_to     ON relations(to_id);
		CREATE INDEX IF NOT EXISTS idx_rel_type   ON relations(type);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_rel_unique ON relations(from_id, to_id, type);
	`); err != nil {
		return err
	}

	// Normalize existing data
	_, _ = s.execHook(s.db, `UPDATE observations SET scope = 'project' WHERE scope IS NULL OR scope = ''`)                      // best-effort migration cleanup
	_, _ = s.execHook(s.db, `UPDATE observations SET topic_key = NULL WHERE topic_key = ''`)                                    // best-effort migration cleanup
	_, _ = s.execHook(s.db, `UPDATE observations SET revision_count = 1 WHERE revision_count IS NULL OR revision_count < 1`)    // best-effort migration cleanup
	_, _ = s.execHook(s.db, `UPDATE observations SET duplicate_count = 1 WHERE duplicate_count IS NULL OR duplicate_count < 1`) // best-effort migration cleanup
	_, _ = s.execHook(s.db, `UPDATE observations SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = ''`)      // best-effort migration cleanup
	_, _ = s.execHook(s.db, `UPDATE user_prompts SET project = '' WHERE project IS NULL`)                                       // best-effort migration cleanup

	// Create FTS triggers (idempotent)
	var name string
	err := s.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='trigger' AND name='obs_fts_insert'",
	).Scan(&name)

	if err == sql.ErrNoRows {
		triggers := `
			CREATE TRIGGER obs_fts_insert AFTER INSERT ON observations BEGIN
				INSERT INTO observations_fts(rowid, title, content, tool_name, type, project)
				VALUES (new.id, new.title, new.content, new.tool_name, new.type, new.project);
			END;

			CREATE TRIGGER obs_fts_delete AFTER DELETE ON observations BEGIN
				INSERT INTO observations_fts(observations_fts, rowid, title, content, tool_name, type, project)
				VALUES ('delete', old.id, old.title, old.content, old.tool_name, old.type, old.project);
			END;

			CREATE TRIGGER obs_fts_update AFTER UPDATE ON observations BEGIN
				INSERT INTO observations_fts(observations_fts, rowid, title, content, tool_name, type, project)
				VALUES ('delete', old.id, old.title, old.content, old.tool_name, old.type, old.project);
				INSERT INTO observations_fts(rowid, title, content, tool_name, type, project)
				VALUES (new.id, new.title, new.content, new.tool_name, new.type, new.project);
			END;
		`
		if _, err := s.execHook(s.db, triggers); err != nil {
			return err
		}
	}

	// Prompts FTS triggers
	var promptTrigger string
	err = s.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='trigger' AND name='prompt_fts_insert'",
	).Scan(&promptTrigger)

	if err == sql.ErrNoRows {
		promptTriggers := `
			CREATE TRIGGER prompt_fts_insert AFTER INSERT ON user_prompts BEGIN
				INSERT INTO prompts_fts(rowid, content, project)
				VALUES (new.id, new.content, new.project);
			END;

			CREATE TRIGGER prompt_fts_delete AFTER DELETE ON user_prompts BEGIN
				INSERT INTO prompts_fts(prompts_fts, rowid, content, project)
				VALUES ('delete', old.id, old.content, old.project);
			END;

			CREATE TRIGGER prompt_fts_update AFTER UPDATE ON user_prompts BEGIN
				INSERT INTO prompts_fts(prompts_fts, rowid, content, project)
				VALUES ('delete', old.id, old.content, old.project);
				INSERT INTO prompts_fts(rowid, content, project)
				VALUES (new.id, new.content, new.project);
			END;
		`
		if _, err := s.execHook(s.db, promptTriggers); err != nil {
			return err
		}
	}

	return nil
}

// ─── Sessions ────────────────────────────────────────────────────────────────

// CreateSession registers a new coding session.
func (s *Store) CreateSession(id, project, directory string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO sessions (id, project, directory) VALUES (?, ?, ?)`,
		id, project, directory,
	)
	return err
}

// EndSession marks a session as completed with an optional summary.
func (s *Store) EndSession(id string, summary string) error {
	_, err := s.db.Exec(
		`UPDATE sessions SET ended_at = datetime('now'), summary = ? WHERE id = ?`,
		nullableString(summary), id,
	)
	return err
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(
		`SELECT id, project, directory, started_at, ended_at, summary FROM sessions WHERE id = ?`, id,
	)
	var sess Session
	if err := row.Scan(&sess.ID, &sess.Project, &sess.Directory, &sess.StartedAt, &sess.EndedAt, &sess.Summary); err != nil {
		return nil, err
	}
	return &sess, nil
}

// RecentSessions returns recent sessions with observation counts.
func (s *Store) RecentSessions(project string, limit int) ([]SessionSummary, error) {
	if limit <= 0 {
		limit = 5
	}

	query := `
		SELECT s.id, s.project, s.started_at, s.ended_at, s.summary,
		       COUNT(o.id) as observation_count
		FROM sessions s
		LEFT JOIN observations o ON o.session_id = s.id AND o.deleted_at IS NULL
		WHERE 1=1
	`
	args := []any{}

	if project != "" {
		query += " AND s.project = ?"
		args = append(args, project)
	}

	query += " GROUP BY s.id ORDER BY MAX(COALESCE(o.created_at, s.started_at)) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.queryItHook(s.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		if err := rows.Scan(&ss.ID, &ss.Project, &ss.StartedAt, &ss.EndedAt, &ss.Summary, &ss.ObservationCount); err != nil {
			return nil, err
		}
		results = append(results, ss)
	}
	return results, rows.Err()
}

// ─── Observations ────────────────────────────────────────────────────────────

// AddObservation creates a new observation with topic_key upsert and deduplication.
func (s *Store) AddObservation(p AddObservationParams) (int64, error) {
	// Strip <private>...</private> tags before persisting
	title := stripPrivateTags(p.Title)
	content := stripPrivateTags(p.Content)

	if len(content) > s.cfg.MaxObservationLength {
		content = content[:s.cfg.MaxObservationLength] + "... [truncated]"
	}
	scope := normalizeScope(p.Scope)
	normHash := hashNormalized(content)
	topicKey := normalizeTopicKey(p.TopicKey)

	// Topic key upsert: if topic_key matches, update the existing observation
	if topicKey != "" {
		var existingID int64
		err := s.db.QueryRow(
			`SELECT id FROM observations
			 WHERE topic_key = ?
			   AND ifnull(project, '') = ifnull(?, '')
			   AND scope = ?
			   AND deleted_at IS NULL
			 ORDER BY datetime(updated_at) DESC, datetime(created_at) DESC
			 LIMIT 1`,
			topicKey, nullableString(p.Project), scope,
		).Scan(&existingID)
		if err == nil {
			if _, err := s.execHook(s.db,
				`UPDATE observations
				 SET type = ?,
				     title = ?,
				     content = ?,
				     tool_name = ?,
				     topic_key = ?,
				     normalized_hash = ?,
				     revision_count = revision_count + 1,
				     last_seen_at = datetime('now'),
				     updated_at = datetime('now')
				 WHERE id = ?`,
				p.Type, title, content,
				nullableString(p.ToolName),
				nullableString(topicKey),
				normHash, existingID,
			); err != nil {
				return 0, err
			}
			return existingID, nil
		}
		if err != sql.ErrNoRows {
			return 0, err
		}
	}

	// Deduplication: same content hash within the dedup window
	window := dedupeWindowExpression(s.cfg.DedupeWindow)
	var existingID int64
	err := s.db.QueryRow(
		`SELECT id FROM observations
		 WHERE normalized_hash = ?
		   AND ifnull(project, '') = ifnull(?, '')
		   AND scope = ?
		   AND type = ?
		   AND title = ?
		   AND deleted_at IS NULL
		   AND datetime(created_at) >= datetime('now', ?)
		 ORDER BY created_at DESC
		 LIMIT 1`,
		normHash, nullableString(p.Project), scope, p.Type, title, window,
	).Scan(&existingID)
	if err == nil {
		if _, err := s.execHook(s.db,
			`UPDATE observations
			 SET duplicate_count = duplicate_count + 1,
			     last_seen_at = datetime('now'),
			     updated_at = datetime('now')
			 WHERE id = ?`,
			existingID,
		); err != nil {
			return 0, err
		}
		return existingID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// Insert new observation
	res, err := s.execHook(s.db,
		`INSERT INTO observations (session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1, datetime('now'), datetime('now'))`,
		p.SessionID, p.Type, title, content,
		nullableString(p.ToolName), nullableString(p.Project), scope, nullableString(topicKey), normHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RecentObservations returns recent observations filtered by project and scope.
func (s *Store) RecentObservations(project, scope string, limit int) ([]Observation, error) {
	if limit <= 0 {
		limit = s.cfg.MaxContextResults
	}

	query := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.tool_name, o.project,
		       o.scope, o.topic_key, o.revision_count, o.duplicate_count, o.last_seen_at, o.created_at, o.updated_at, o.deleted_at
		FROM observations o
		WHERE o.deleted_at IS NULL
	`
	args := []any{}

	if project != "" {
		query += " AND o.project = ?"
		args = append(args, project)
	}
	if scope != "" {
		query += " AND o.scope = ?"
		args = append(args, normalizeScope(scope))
	}

	query += " ORDER BY o.created_at DESC LIMIT ?"
	args = append(args, limit)

	return s.queryObservations(query, args...)
}

// GetObservation retrieves a single observation by ID (excludes soft-deleted).
func (s *Store) GetObservation(id int64) (*Observation, error) {
	row := s.db.QueryRow(
		`SELECT id, session_id, type, title, content, tool_name, project,
		        scope, topic_key, revision_count, duplicate_count, last_seen_at, created_at, updated_at, deleted_at
		 FROM observations WHERE id = ? AND deleted_at IS NULL`, id,
	)
	var o Observation
	if err := row.Scan(
		&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content,
		&o.ToolName, &o.Project, &o.Scope, &o.TopicKey, &o.RevisionCount, &o.DuplicateCount, &o.LastSeenAt,
		&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &o, nil
}

// UpdateObservation partially updates an observation by ID.
func (s *Store) UpdateObservation(id int64, p UpdateObservationParams) (*Observation, error) {
	obs, err := s.GetObservation(id)
	if err != nil {
		return nil, err
	}

	typ := obs.Type
	title := obs.Title
	content := obs.Content
	project := ""
	if obs.Project != nil {
		project = *obs.Project
	}
	scope := obs.Scope
	topicKey := ""
	if obs.TopicKey != nil {
		topicKey = *obs.TopicKey
	}

	if p.Type != nil {
		typ = *p.Type
	}
	if p.Title != nil {
		title = stripPrivateTags(*p.Title)
	}
	if p.Content != nil {
		content = stripPrivateTags(*p.Content)
		if len(content) > s.cfg.MaxObservationLength {
			content = content[:s.cfg.MaxObservationLength] + "... [truncated]"
		}
	}
	if p.Project != nil {
		project = *p.Project
	}
	if p.Scope != nil {
		scope = normalizeScope(*p.Scope)
	}
	if p.TopicKey != nil {
		topicKey = normalizeTopicKey(*p.TopicKey)
	}

	if _, err := s.execHook(s.db,
		`UPDATE observations
		 SET type = ?,
		     title = ?,
		     content = ?,
		     project = ?,
		     scope = ?,
		     topic_key = ?,
		     normalized_hash = ?,
		     revision_count = revision_count + 1,
		     updated_at = datetime('now')
		 WHERE id = ? AND deleted_at IS NULL`,
		typ, title, content,
		nullableString(project), scope, nullableString(topicKey),
		hashNormalized(content), id,
	); err != nil {
		return nil, err
	}

	return s.GetObservation(id)
}

// DeleteObservation soft-deletes (or hard-deletes) an observation by ID.
func (s *Store) DeleteObservation(id int64, hardDelete bool) error {
	if hardDelete {
		_, err := s.db.Exec(`DELETE FROM observations WHERE id = ?`, id)
		return err
	}
	_, err := s.db.Exec(
		`UPDATE observations
		 SET deleted_at = datetime('now'),
		     updated_at = datetime('now')
		 WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	return err
}

// ─── Relations ───────────────────────────────────────────────────────────────

// AddRelation creates a typed directional edge between two observations.
// If Bidirectional is true, both directions are created atomically.
// Returns the IDs of created relations (1 or 2).
func (s *Store) AddRelation(p AddRelationParams) ([]int64, error) {
	// Validate: no self-relations
	if p.FromID == p.ToID {
		return nil, fmt.Errorf("cannot create self-relation: from_id and to_id are both %d", p.FromID)
	}

	// Default relation type
	if p.Type == "" {
		p.Type = "relates_to"
	}

	// Validate both observations exist and are not soft-deleted
	for _, id := range []int64{p.FromID, p.ToID} {
		row, err := s.queryItHook(s.db,
			`SELECT 1 FROM observations WHERE id = ? AND deleted_at IS NULL`, id)
		if err != nil {
			return nil, fmt.Errorf("checking observation %d: %w", id, err)
		}
		found := row.Next()
		row.Close()
		if !found {
			return nil, fmt.Errorf("observation %d not found or is deleted", id)
		}
	}

	note := nullableString(p.Note)

	if !p.Bidirectional {
		// Single direction
		res, err := s.execHook(s.db,
			`INSERT INTO relations (from_id, to_id, type, note) VALUES (?, ?, ?, ?)`,
			p.FromID, p.ToID, p.Type, note,
		)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, fmt.Errorf("relation already exists: %d → %d (%s)", p.FromID, p.ToID, p.Type)
			}
			return nil, fmt.Errorf("creating relation: %w", err)
		}
		id, _ := res.LastInsertId()
		return []int64{id}, nil
	}

	// Bidirectional: create both directions in a transaction
	tx, err := s.beginTxHook()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res1, err := s.execHook(tx,
		`INSERT INTO relations (from_id, to_id, type, note) VALUES (?, ?, ?, ?)`,
		p.FromID, p.ToID, p.Type, note,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("relation already exists: %d → %d (%s)", p.FromID, p.ToID, p.Type)
		}
		return nil, fmt.Errorf("creating forward relation: %w", err)
	}

	res2, err := s.execHook(tx,
		`INSERT INTO relations (from_id, to_id, type, note) VALUES (?, ?, ?, ?)`,
		p.ToID, p.FromID, p.Type, note,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("reverse relation already exists: %d → %d (%s)", p.ToID, p.FromID, p.Type)
		}
		return nil, fmt.Errorf("creating reverse relation: %w", err)
	}

	if err := s.commitHook(tx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	id1, _ := res1.LastInsertId()
	id2, _ := res2.LastInsertId()
	return []int64{id1, id2}, nil
}

// RemoveRelation hard-deletes a relation by its ID.
func (s *Store) RemoveRelation(id int64) error {
	res, err := s.execHook(s.db, `DELETE FROM relations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting relation: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("relation %d not found", id)
	}
	return nil
}

// GetRelations returns all relations where the observation is either source or target.
func (s *Store) GetRelations(observationID int64) ([]Relation, error) {
	rows, err := s.queryHook(s.db,
		`SELECT id, from_id, to_id, type, COALESCE(note, ''), created_at
		 FROM relations
		 WHERE from_id = ? OR to_id = ?
		 ORDER BY created_at ASC`,
		observationID, observationID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying relations: %w", err)
	}
	defer rows.Close()

	var result []Relation
	for rows.Next() {
		var r Relation
		if err := rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &r.Note, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning relation: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// BuildContext traverses the relation graph from a starting observation using BFS.
// It returns a compact context tree with observation metadata (no full content).
// Default depth is 2, max is 5. Cycle detection prevents infinite loops.
func (s *Store) BuildContext(observationID int64, maxDepth int) (*ContextResult, error) {
	// Clamp depth
	if maxDepth <= 0 {
		maxDepth = 2
	}
	if maxDepth > 5 {
		maxDepth = 5
	}

	// Get root observation
	root, err := s.GetObservation(observationID)
	if err != nil {
		return nil, fmt.Errorf("root observation %d not found: %w", observationID, err)
	}

	// BFS traversal
	type queueItem struct {
		id    int64
		depth int
	}

	visited := map[int64]bool{observationID: true}
	queue := []queueItem{{id: observationID, depth: 0}}
	var connected []ContextNode
	actualMaxDepth := 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// Get all relations for this node
		rels, err := s.GetRelations(current.id)
		if err != nil {
			return nil, fmt.Errorf("getting relations for %d: %w", current.id, err)
		}

		for _, rel := range rels {
			// Determine the other side and direction
			otherID := rel.ToID
			direction := "outgoing"
			if rel.ToID == current.id {
				otherID = rel.FromID
				direction = "incoming"
			}

			if visited[otherID] {
				continue
			}
			visited[otherID] = true

			// Get lightweight metadata for the connected observation
			row, err := s.queryItHook(s.db,
				`SELECT id, title, type, COALESCE(project, ''), created_at
				 FROM observations WHERE id = ?`, otherID)
			if err != nil {
				continue // skip if observation was hard-deleted between queries
			}

			if !row.Next() {
				row.Close()
				continue // skip if observation was hard-deleted between queries
			}

			var node ContextNode
			if err := row.Scan(&node.ID, &node.Title, &node.Type, &node.Project, &node.CreatedAt); err != nil {
				row.Close()
				continue // skip if scan fails
			}
			row.Close()

			nodeDepth := current.depth + 1
			node.RelationType = rel.Type
			node.Direction = direction
			node.Note = rel.Note
			node.Depth = nodeDepth

			connected = append(connected, node)

			if nodeDepth > actualMaxDepth {
				actualMaxDepth = nodeDepth
			}

			// Enqueue for further traversal
			queue = append(queue, queueItem{id: otherID, depth: nodeDepth})
		}
	}

	return &ContextResult{
		Root:       *root,
		Connected:  connected,
		TotalNodes: len(connected),
		MaxDepth:   actualMaxDepth,
	}, nil
}

// ─── User Prompts ────────────────────────────────────────────────────────────

// AddPrompt saves a user prompt to persistent memory.
func (s *Store) AddPrompt(p AddPromptParams) (int64, error) {
	content := stripPrivateTags(p.Content)
	if len(content) > s.cfg.MaxObservationLength {
		content = content[:s.cfg.MaxObservationLength] + "... [truncated]"
	}

	res, err := s.execHook(s.db,
		`INSERT INTO user_prompts (session_id, content, project) VALUES (?, ?, ?)`,
		p.SessionID, content, nullableString(p.Project),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RecentPrompts returns recent user prompts filtered by project.
func (s *Store) RecentPrompts(project string, limit int) ([]Prompt, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `SELECT id, session_id, content, ifnull(project, '') as project, created_at FROM user_prompts`
	args := []any{}

	if project != "" {
		query += " WHERE project = ?"
		args = append(args, project)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.queryItHook(s.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.SessionID, &p.Content, &p.Project, &p.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// SearchPrompts searches user prompts using FTS5.
func (s *Store) SearchPrompts(query string, project string, limit int) ([]Prompt, error) {
	if limit <= 0 {
		limit = 10
	}

	ftsQuery := sanitizeFTS(query)

	sqlStr := `
		SELECT p.id, p.session_id, p.content, ifnull(p.project, '') as project, p.created_at
		FROM prompts_fts fts
		JOIN user_prompts p ON p.id = fts.rowid
		WHERE prompts_fts MATCH ?
	`
	args := []any{ftsQuery}

	if project != "" {
		sqlStr += " AND p.project = ?"
		args = append(args, project)
	}

	sqlStr += " ORDER BY fts.rank LIMIT ?"
	args = append(args, limit)

	rows, err := s.queryItHook(s.db, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search prompts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.SessionID, &p.Content, &p.Project, &p.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// ─── Timeline ────────────────────────────────────────────────────────────────

// Timeline provides chronological context around a specific observation.
// Progressive disclosure pattern: search → timeline → get_observation.
func (s *Store) Timeline(observationID int64, before, after int) (*TimelineResult, error) {
	if before <= 0 {
		before = 5
	}
	if after <= 0 {
		after = 5
	}

	// Get the focus observation
	focus, err := s.GetObservation(observationID)
	if err != nil {
		return nil, fmt.Errorf("timeline: observation #%d not found: %w", observationID, err)
	}

	// Get session info (may be nil for manual-save observations)
	session, _ := s.GetSession(focus.SessionID)

	// Get observations BEFORE the focus (same session, older)
	beforeRows, err := s.queryItHook(s.db, `
		SELECT id, session_id, type, title, content, tool_name, project,
		       scope, topic_key, revision_count, duplicate_count, last_seen_at, created_at, updated_at, deleted_at
		FROM observations
		WHERE session_id = ? AND id < ? AND deleted_at IS NULL
		ORDER BY id DESC
		LIMIT ?
	`, focus.SessionID, observationID, before)
	if err != nil {
		return nil, fmt.Errorf("timeline: before query: %w", err)
	}
	defer func() { _ = beforeRows.Close() }()

	var beforeEntries []TimelineEntry
	for beforeRows.Next() {
		var e TimelineEntry
		if err := beforeRows.Scan(
			&e.ID, &e.SessionID, &e.Type, &e.Title, &e.Content,
			&e.ToolName, &e.Project, &e.Scope, &e.TopicKey, &e.RevisionCount, &e.DuplicateCount, &e.LastSeenAt,
			&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
		); err != nil {
			return nil, err
		}
		beforeEntries = append(beforeEntries, e)
	}
	if err := beforeRows.Err(); err != nil {
		return nil, err
	}
	// Reverse to chronological order
	for i, j := 0, len(beforeEntries)-1; i < j; i, j = i+1, j-1 {
		beforeEntries[i], beforeEntries[j] = beforeEntries[j], beforeEntries[i]
	}

	// Get observations AFTER the focus (same session, newer)
	afterRows, err := s.queryItHook(s.db, `
		SELECT id, session_id, type, title, content, tool_name, project,
		       scope, topic_key, revision_count, duplicate_count, last_seen_at, created_at, updated_at, deleted_at
		FROM observations
		WHERE session_id = ? AND id > ? AND deleted_at IS NULL
		ORDER BY id ASC
		LIMIT ?
	`, focus.SessionID, observationID, after)
	if err != nil {
		return nil, fmt.Errorf("timeline: after query: %w", err)
	}
	defer func() { _ = afterRows.Close() }()

	var afterEntries []TimelineEntry
	for afterRows.Next() {
		var e TimelineEntry
		if err := afterRows.Scan(
			&e.ID, &e.SessionID, &e.Type, &e.Title, &e.Content,
			&e.ToolName, &e.Project, &e.Scope, &e.TopicKey, &e.RevisionCount, &e.DuplicateCount, &e.LastSeenAt,
			&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
		); err != nil {
			return nil, err
		}
		afterEntries = append(afterEntries, e)
	}
	if err := afterRows.Err(); err != nil {
		return nil, err
	}

	// Count total observations in session
	var totalInRange int
	_ = s.db.QueryRow(
		"SELECT COUNT(*) FROM observations WHERE session_id = ? AND deleted_at IS NULL", focus.SessionID,
	).Scan(&totalInRange)

	return &TimelineResult{
		Focus:        *focus,
		Before:       beforeEntries,
		After:        afterEntries,
		SessionInfo:  session,
		TotalInRange: totalInRange,
	}, nil
}

// ─── Search (FTS5) ───────────────────────────────────────────────────────────

// Search performs full-text search across observations with filters.
// If the query is empty or whitespace-only, falls back to returning recent observations.
func (s *Store) Search(query string, opts SearchOptions) ([]SearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > s.cfg.MaxSearchResults {
		limit = s.cfg.MaxSearchResults
	}

	ftsQuery := sanitizeFTS(query)

	// Empty or whitespace-only query: fall back to recent observations (no FTS).
	if ftsQuery == "" {
		return s.searchRecent(opts, limit)
	}

	sqlStr := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.tool_name, o.project,
		       o.scope, o.topic_key, o.revision_count, o.duplicate_count, o.last_seen_at, o.created_at, o.updated_at, o.deleted_at,
		       fts.rank
		FROM observations_fts fts
		JOIN observations o ON o.id = fts.rowid
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL
	`
	args := []any{ftsQuery}

	if opts.Type != "" {
		sqlStr += " AND o.type = ?"
		args = append(args, opts.Type)
	}
	if opts.Project != "" {
		sqlStr += " AND o.project = ?"
		args = append(args, opts.Project)
	}
	if opts.Scope != "" {
		sqlStr += " AND o.scope = ?"
		args = append(args, normalizeScope(opts.Scope))
	}

	sqlStr += " ORDER BY fts.rank LIMIT ?"
	args = append(args, limit)

	rows, err := s.queryItHook(s.db, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(
			&sr.ID, &sr.SessionID, &sr.Type, &sr.Title, &sr.Content,
			&sr.ToolName, &sr.Project, &sr.Scope, &sr.TopicKey, &sr.RevisionCount, &sr.DuplicateCount,
			&sr.LastSeenAt, &sr.CreatedAt, &sr.UpdatedAt, &sr.DeletedAt,
			&sr.Rank,
		); err != nil {
			return nil, err
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// searchRecent returns the most recent observations without FTS, used as
// fallback when the query is empty or whitespace-only.
func (s *Store) searchRecent(opts SearchOptions, limit int) ([]SearchResult, error) {
	sqlStr := `
		SELECT id, session_id, type, title, content, tool_name, project,
		       scope, topic_key, revision_count, duplicate_count, last_seen_at, created_at, updated_at, deleted_at,
		       0 AS rank
		FROM observations
		WHERE deleted_at IS NULL
	`
	var args []any

	if opts.Type != "" {
		sqlStr += " AND type = ?"
		args = append(args, opts.Type)
	}
	if opts.Project != "" {
		sqlStr += " AND project = ?"
		args = append(args, opts.Project)
	}
	if opts.Scope != "" {
		sqlStr += " AND scope = ?"
		args = append(args, normalizeScope(opts.Scope))
	}

	sqlStr += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.queryItHook(s.db, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search recent: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(
			&sr.ID, &sr.SessionID, &sr.Type, &sr.Title, &sr.Content,
			&sr.ToolName, &sr.Project, &sr.Scope, &sr.TopicKey, &sr.RevisionCount, &sr.DuplicateCount,
			&sr.LastSeenAt, &sr.CreatedAt, &sr.UpdatedAt, &sr.DeletedAt,
			&sr.Rank,
		); err != nil {
			return nil, err
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// ─── Stats ───────────────────────────────────────────────────────────────────

// Stats returns aggregate memory statistics.
func (s *Store) Stats() (*Stats, error) {
	stats := &Stats{}

	_ = s.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&stats.TotalSessions)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM observations WHERE deleted_at IS NULL").Scan(&stats.TotalObservations)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM user_prompts").Scan(&stats.TotalPrompts)

	rows, err := s.queryItHook(s.db, "SELECT project FROM observations WHERE project IS NOT NULL AND deleted_at IS NULL GROUP BY project ORDER BY MAX(created_at) DESC")
	if err != nil {
		return stats, nil
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			stats.Projects = append(stats.Projects, p)
		}
	}

	return stats, nil
}

// ─── Context Formatting ─────────────────────────────────────────────────────

// FormatContext returns a markdown-formatted summary of recent memory.
func (s *Store) FormatContext(project, scope string) (string, error) {
	sessions, err := s.RecentSessions(project, 5)
	if err != nil {
		return "", err
	}

	observations, err := s.RecentObservations(project, scope, s.cfg.MaxContextResults)
	if err != nil {
		return "", err
	}

	prompts, err := s.RecentPrompts(project, 10)
	if err != nil {
		return "", err
	}

	if len(sessions) == 0 && len(observations) == 0 && len(prompts) == 0 {
		return "", nil
	}

	var b strings.Builder
	b.WriteString("## Memory from Previous Sessions\n\n")

	if len(sessions) > 0 {
		b.WriteString("### Recent Sessions\n")
		for _, sess := range sessions {
			summary := ""
			if sess.Summary != nil {
				summary = fmt.Sprintf(": %s", Truncate(*sess.Summary, 200))
			}
			fmt.Fprintf(&b, "- **%s** (%s)%s [%d observations]\n",
				sess.Project, sess.StartedAt, summary, sess.ObservationCount)
		}
		b.WriteString("\n")
	}

	if len(prompts) > 0 {
		b.WriteString("### Recent User Prompts\n")
		for _, p := range prompts {
			fmt.Fprintf(&b, "- %s: %s\n", p.CreatedAt, Truncate(p.Content, 200))
		}
		b.WriteString("\n")
	}

	if len(observations) > 0 {
		b.WriteString("### Recent Observations\n")
		for _, obs := range observations {
			fmt.Fprintf(&b, "- [%s] **%s**: %s\n",
				obs.Type, obs.Title, Truncate(obs.Content, 300))
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

// ─── Export / Import ─────────────────────────────────────────────────────────

// Export dumps the entire memory database as a serializable struct.
func (s *Store) Export() (*ExportData, error) {
	data := &ExportData{
		Version:    "0.1.0",
		ExportedAt: Now(),
	}

	// Sessions
	rows, err := s.queryItHook(s.db,
		"SELECT id, project, directory, started_at, ended_at, summary FROM sessions ORDER BY started_at",
	)
	if err != nil {
		return nil, fmt.Errorf("export sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Project, &sess.Directory, &sess.StartedAt, &sess.EndedAt, &sess.Summary); err != nil {
			return nil, err
		}
		data.Sessions = append(data.Sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Observations
	obsRows, err := s.queryItHook(s.db,
		`SELECT id, session_id, type, title, content, tool_name, project,
		        scope, topic_key, revision_count, duplicate_count, last_seen_at, created_at, updated_at, deleted_at
		 FROM observations ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("export observations: %w", err)
	}
	defer func() { _ = obsRows.Close() }()
	for obsRows.Next() {
		var o Observation
		if err := obsRows.Scan(
			&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content,
			&o.ToolName, &o.Project, &o.Scope, &o.TopicKey, &o.RevisionCount, &o.DuplicateCount, &o.LastSeenAt,
			&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
		); err != nil {
			return nil, err
		}
		data.Observations = append(data.Observations, o)
	}
	if err := obsRows.Err(); err != nil {
		return nil, err
	}

	// Prompts
	promptRows, err := s.queryItHook(s.db,
		"SELECT id, session_id, content, ifnull(project, '') as project, created_at FROM user_prompts ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("export prompts: %w", err)
	}
	defer func() { _ = promptRows.Close() }()
	for promptRows.Next() {
		var p Prompt
		if err := promptRows.Scan(&p.ID, &p.SessionID, &p.Content, &p.Project, &p.CreatedAt); err != nil {
			return nil, err
		}
		data.Prompts = append(data.Prompts, p)
	}
	if err := promptRows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// Import loads exported data into the memory database.
func (s *Store) Import(data *ExportData) (*ImportResult, error) {
	tx, err := s.beginTxHook()
	if err != nil {
		return nil, fmt.Errorf("import: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result := &ImportResult{}

	for _, sess := range data.Sessions {
		res, err := s.execHook(tx,
			`INSERT OR IGNORE INTO sessions (id, project, directory, started_at, ended_at, summary)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			sess.ID, sess.Project, sess.Directory, sess.StartedAt, sess.EndedAt, sess.Summary,
		)
		if err != nil {
			return nil, fmt.Errorf("import session %s: %w", sess.ID, err)
		}
		n, _ := res.RowsAffected()
		result.SessionsImported += int(n)
	}

	for _, obs := range data.Observations {
		_, err := s.execHook(tx,
			`INSERT INTO observations (session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at, created_at, updated_at, deleted_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			obs.SessionID, obs.Type, obs.Title, obs.Content,
			obs.ToolName, obs.Project,
			normalizeScope(obs.Scope),
			nullableString(normalizeTopicKey(derefString(obs.TopicKey))),
			hashNormalized(obs.Content),
			maxInt(obs.RevisionCount, 1),
			maxInt(obs.DuplicateCount, 1),
			obs.LastSeenAt, obs.CreatedAt, obs.UpdatedAt, obs.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("import observation %d: %w", obs.ID, err)
		}
		result.ObservationsImported++
	}

	for _, p := range data.Prompts {
		_, err := s.execHook(tx,
			`INSERT INTO user_prompts (session_id, content, project, created_at)
			 VALUES (?, ?, ?, ?)`,
			p.SessionID, p.Content, p.Project, p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("import prompt %d: %w", p.ID, err)
		}
		result.PromptsImported++
	}

	if err := s.commitHook(tx); err != nil {
		return nil, fmt.Errorf("import: commit: %w", err)
	}

	return result, nil
}

// ─── Passive Capture ─────────────────────────────────────────────────────────

// learningHeaderPattern matches section headers for learnings in both English and Spanish.
var learningHeaderPattern = regexp.MustCompile(
	`(?im)^#{2,3}\s+(?:Aprendizajes(?:\s+Clave)?|Key\s+Learnings?|Learnings?):?\s*$`,
)

// minLearningLength is the minimum character length for a valid learning.
const minLearningLength = 20

// ExtractLearnings parses structured learning items from text.
// Looks for "## Key Learnings:" or "## Aprendizajes Clave:" sections
// and extracts numbered or bullet items.
func ExtractLearnings(text string) []string {
	matches := learningHeaderPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	// Process sections in reverse — use first valid one (most recent)
	for i := len(matches) - 1; i >= 0; i-- {
		sectionStart := matches[i][1]
		sectionText := text[sectionStart:]

		// Cut off at next major section header
		if nextHeader := regexp.MustCompile(`\n#{1,3} `).FindStringIndex(sectionText); nextHeader != nil {
			sectionText = sectionText[:nextHeader[0]]
		}

		var learnings []string

		// Try numbered items: "1. text" or "1) text"
		numbered := regexp.MustCompile(`(?m)^\s*\d+[.)]\s+(.+)`).FindAllStringSubmatch(sectionText, -1)
		if len(numbered) > 0 {
			for _, m := range numbered {
				cleaned := cleanMarkdown(m[1])
				if len(cleaned) >= minLearningLength {
					learnings = append(learnings, cleaned)
				}
			}
		}

		// Fall back to bullet items
		if len(learnings) == 0 {
			bullets := regexp.MustCompile(`(?m)^\s*[-*]\s+(.+)`).FindAllStringSubmatch(sectionText, -1)
			for _, m := range bullets {
				cleaned := cleanMarkdown(m[1])
				if len(cleaned) >= minLearningLength {
					learnings = append(learnings, cleaned)
				}
			}
		}

		if len(learnings) > 0 {
			return learnings
		}
	}

	return nil
}

// cleanMarkdown strips basic markdown formatting.
func cleanMarkdown(text string) string {
	text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "$1") // bold
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "$1")       // inline code
	text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(text, "$1")     // italic
	return strings.TrimSpace(strings.Join(strings.Fields(text), " "))
}

// PassiveCapture extracts learnings from text and saves them as observations.
func (s *Store) PassiveCapture(p PassiveCaptureParams) (*PassiveCaptureResult, error) {
	result := &PassiveCaptureResult{}

	learnings := ExtractLearnings(p.Content)
	result.Extracted = len(learnings)

	if len(learnings) == 0 {
		return result, nil
	}

	for _, learning := range learnings {
		normHash := hashNormalized(learning)
		var existingID int64
		err := s.db.QueryRow(
			`SELECT id FROM observations
			 WHERE normalized_hash = ?
			   AND ifnull(project, '') = ifnull(?, '')
			   AND deleted_at IS NULL
			 LIMIT 1`,
			normHash, nullableString(p.Project),
		).Scan(&existingID)

		if err == nil {
			result.Duplicates++
			continue
		}

		title := learning
		if len(title) > 60 {
			title = title[:60] + "..."
		}

		_, err = s.AddObservation(AddObservationParams{
			SessionID: p.SessionID,
			Type:      "passive",
			Title:     title,
			Content:   learning,
			Project:   p.Project,
			Scope:     "project",
			ToolName:  p.Source,
		})
		if err != nil {
			return result, fmt.Errorf("passive capture save: %w", err)
		}
		result.Saved++
	}

	return result, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *Store) queryObservations(query string, args ...any) ([]Observation, error) {
	rows, err := s.queryItHook(s.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []Observation
	for rows.Next() {
		var o Observation
		if err := rows.Scan(
			&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content,
			&o.ToolName, &o.Project, &o.Scope, &o.TopicKey, &o.RevisionCount, &o.DuplicateCount, &o.LastSeenAt,
			&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, o)
	}
	return results, rows.Err()
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Truncate shortens a string to max length with ellipsis.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func normalizeScope(scope string) string {
	v := strings.TrimSpace(strings.ToLower(scope))
	if v == "personal" {
		return "personal"
	}
	return "project"
}

// SuggestTopicKey generates a stable topic key suggestion from type/title/content.
func SuggestTopicKey(typ, title, content string) string {
	family := inferTopicFamily(typ, title, content)
	cleanTitle := stripPrivateTags(title)
	segment := normalizeTopicSegment(cleanTitle)

	if segment == "" {
		cleanContent := stripPrivateTags(content)
		words := strings.Fields(strings.ToLower(cleanContent))
		if len(words) > 8 {
			words = words[:8]
		}
		segment = normalizeTopicSegment(strings.Join(words, " "))
	}

	if segment == "" {
		segment = "general"
	}

	if strings.HasPrefix(segment, family+"-") {
		segment = strings.TrimPrefix(segment, family+"-")
	}
	if segment == "" || segment == family {
		segment = "general"
	}

	return family + "/" + segment
}

func inferTopicFamily(typ, title, content string) string {
	t := strings.TrimSpace(strings.ToLower(typ))
	switch t {
	case "architecture", "design", "adr", "refactor":
		return "architecture"
	case "bug", "bugfix", "fix", "incident", "hotfix":
		return "bug"
	case "decision":
		return "decision"
	case "pattern", "convention", "guideline":
		return "pattern"
	case "config", "setup", "infra", "infrastructure", "ci":
		return "config"
	case "discovery", "investigation", "root_cause", "root-cause":
		return "discovery"
	case "learning", "learn":
		return "learning"
	case "session_summary":
		return "session"
	}

	text := strings.ToLower(title + " " + content)
	if hasAny(text, "bug", "fix", "panic", "error", "crash", "regression", "incident", "hotfix") {
		return "bug"
	}
	if hasAny(text, "architecture", "design", "adr", "boundary", "hexagonal", "refactor") {
		return "architecture"
	}
	if hasAny(text, "decision", "tradeoff", "chose", "choose", "decide") {
		return "decision"
	}
	if hasAny(text, "pattern", "convention", "naming", "guideline") {
		return "pattern"
	}
	if hasAny(text, "config", "setup", "environment", "env", "docker", "pipeline") {
		return "config"
	}
	if hasAny(text, "discovery", "investigate", "investigation", "found", "root cause") {
		return "discovery"
	}
	if hasAny(text, "learned", "learning") {
		return "learning"
	}

	if t != "" && t != "manual" {
		return normalizeTopicSegment(t)
	}

	return "topic"
}

func hasAny(text string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

func normalizeTopicSegment(s string) string {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`[^a-z0-9]+`)
	v = re.ReplaceAllString(v, " ")
	v = strings.Join(strings.Fields(v), "-")
	if len(v) > 100 {
		v = v[:100]
	}
	return v
}

func normalizeTopicKey(topic string) string {
	v := strings.TrimSpace(strings.ToLower(topic))
	if v == "" {
		return ""
	}
	v = strings.Join(strings.Fields(v), "-")
	if len(v) > 120 {
		v = v[:120]
	}
	return v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func hashNormalized(content string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(content), " "))
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

func dedupeWindowExpression(window time.Duration) string {
	if window <= 0 {
		window = 15 * time.Minute
	}
	minutes := int(window.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return "-" + strconv.Itoa(minutes) + " minutes"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// privateTagRegex matches <private>...</private> tags and their contents.
var privateTagRegex = regexp.MustCompile(`(?is)<private>.*?</private>`)

// stripPrivateTags removes all <private>...</private> content from a string.
func stripPrivateTags(s string) string {
	result := privateTagRegex.ReplaceAllString(s, "[REDACTED]")
	return strings.TrimSpace(result)
}

// sanitizeFTS wraps each word in quotes for safe FTS5 queries.
// "fix auth bug" → `"fix" "auth" "bug"`
func sanitizeFTS(query string) string {
	words := strings.Fields(query)
	for i, w := range words {
		w = strings.Trim(w, `"`)
		words[i] = `"` + w + `"`
	}
	return strings.Join(words, " ")
}

// isUniqueViolation checks if an error is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// ClassifyTool returns the observation type for a given tool name.
func ClassifyTool(toolName string) string {
	switch toolName {
	case "write", "edit", "patch":
		return "file_change"
	case "bash":
		return "command"
	case "read", "view":
		return "file_read"
	case "grep", "glob", "ls":
		return "search"
	default:
		return "tool_use"
	}
}

// Now returns the current time formatted for SQLite.
func Now() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
