# Spec: Knowledge Graph Relations Between Memory Observations

## Functional Requirements

### MUST HAVE

- **FR-001**: Add a `relations` table to the SQLite schema via non-destructive migration (`CREATE TABLE IF NOT EXISTS`). Existing `memory.db` files MUST continue working without manual intervention after upgrade.
- **FR-002**: Each relation connects two observations (`from_id` → `to_id`) with a `relation_type` string (e.g., `relates_to`, `implements`, `depends_on`, `caused_by`, `supersedes`, `part_of`). Custom types allowed — no enum restriction.
- **FR-003**: New MCP tool `mem_relate` — creates a typed directional relation between two existing observations. Parameters: `from_id` (int, required), `to_id` (int, required), `relation_type` (string, required), `note` (string, optional). Returns the created relation ID. Must validate both observation IDs exist and are not soft-deleted.
- **FR-004**: New MCP tool `mem_unrelate` — removes a relation by relation ID. Hard delete (no soft-delete for relations).
- **FR-005**: New MCP tool `mem_build_context` — given a starting `observation_id` and optional `depth` (default 1, max 5), traverses the relation graph bidirectionally and returns a structured context tree. Must include cycle detection to prevent infinite loops.
- **FR-006**: `mem_get_observation` MUST include direct relations (1 level) in its output — both outgoing ("this observation → X") and incoming ("Y → this observation"). This is additive; existing fields remain unchanged.
- **FR-007**: `DeleteObservation` (hard delete) MUST cascade-delete all relations where the deleted observation is either `from_id` or `to_id`. Soft-delete does NOT affect relations (the observation is still referenceable).
- **FR-008**: Relations MUST enforce referential integrity — cannot create a relation to a non-existent or hard-deleted observation.
- **FR-009**: Self-relations (`from_id == to_id`) MUST be rejected.
- **FR-010**: Duplicate relations (same `from_id`, `to_id`, `relation_type`) MUST be rejected with a clear error message.

### SHOULD HAVE

- **FR-011**: `mem_search` results SHOULD include a `relation_count` field showing how many relations each observation has (both directions combined). This helps the AI identify well-connected knowledge nodes.
- **FR-012**: `mem_build_context` output SHOULD include the relation type and direction on each edge, so the AI understands HOW observations are connected (not just THAT they are).
- **FR-013**: `mem_relate` SHOULD support an optional `bidirectional` flag (default `false`). When `true`, creates two relations: `A→B` and `B→A` with the same type and note.

### COULD HAVE

- **FR-014**: `mem_stats` COULD include relation statistics (total relations, most common relation types, most connected observations).
- **FR-015**: A `mem_relations` tool COULD list all relations for a given observation without traversing the full graph (lighter than `mem_build_context`).

### WON'T HAVE (this version)

- **FR-W01**: No visualization or canvas rendering — client responsibility.
- **FR-W02**: No `memory://` URL scheme — separate future feature.
- **FR-W03**: No AI-suggested/auto-inferred relations — explicit only.
- **FR-W04**: No bulk import/export of relations (existing `Export`/`Import` methods will be updated in a future version).
- **FR-W05**: No relation type validation against a predefined list — any string is valid.

## Non-Functional Requirements

- **NFR-001**: Schema migration MUST be automatic and idempotent. Running `migrate()` on an old DB adds the `relations` table without touching existing tables. Running it on a new DB that already has the table is a no-op.
- **NFR-002**: All existing tests (`go test -race ./...`) MUST pass without modification. New features get new tests.
- **NFR-003**: `mem_build_context` with depth=5 on a graph of 1000 observations MUST complete in under 1 second.
- **NFR-004**: Zero new dependencies — relations use the same `modernc.org/sqlite` driver already in `go.mod`.
- **NFR-005**: `CGO_ENABLED=0` static binary constraint MUST be preserved — no CGo SQLite extensions.
- **NFR-006**: All new Store methods MUST use the existing hook pattern (`s.execHook`, `s.queryHook`) for testability.
- **NFR-007**: All new MCP tools MUST follow the existing pattern: one struct per tool in `internal/memtools/`, with `Definition()` and `Handle()` methods, registered in `server.go`.

## Schema Design

```sql
CREATE TABLE IF NOT EXISTS relations (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id   INTEGER NOT NULL,
    to_id     INTEGER NOT NULL,
    type      TEXT    NOT NULL DEFAULT 'relates_to',
    note      TEXT,
    created_at TEXT   NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (from_id) REFERENCES observations(id) ON DELETE CASCADE,
    FOREIGN KEY (to_id)   REFERENCES observations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rel_from    ON relations(from_id);
CREATE INDEX IF NOT EXISTS idx_rel_to      ON relations(to_id);
CREATE INDEX IF NOT EXISTS idx_rel_type    ON relations(type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_rel_unique ON relations(from_id, to_id, type);
```

**Note on CASCADE**: SQLite foreign key enforcement requires `PRAGMA foreign_keys = ON`. This must be set per-connection. Verify current Hoofy behavior — if FKs are not enabled, cascade must be handled in application code (DELETE trigger or explicit cleanup in `DeleteObservation`).

## Constraints

- Must work with `modernc.org/sqlite` (pure Go, no CGo)
- Must work on all 6 release platforms (linux/darwin/windows × amd64/arm64)
- Memory subsystem must remain independent — if memory init fails, SDD tools still work
- New tools depend on `memory.Store` only (same DI pattern as existing memtools)

## Assumptions

- FTS5 virtual table is NOT affected by the new `relations` table
- The `observations` table schema is stable and won't change concurrently with this feature
- Observation IDs are stable (auto-increment, never reused after hard delete)
- The `execHook`/`queryHook` pattern provides sufficient test isolation for relation operations
