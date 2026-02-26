# Design: Knowledge Graph Relations Between Memory Observations

## Architecture Overview

The relations system is an **additive layer** on top of the existing memory engine. It introduces:

1. A new `relations` table in SQLite (non-destructive migration)
2. Three new Store methods (`AddRelation`, `RemoveRelation`, `BuildContext`)
3. One extended Store method (`GetObservationWithRelations` — wraps existing `GetObservation`)
4. Three new MCP tools in `internal/memtools/` (`mem_relate`, `mem_unrelate`, `mem_build_context`)
5. Enhancement to `GetObservationTool.Handle` to show direct relations
6. Enhancement to `DeleteObservation` to cascade-clean relations on hard delete

**No existing interfaces, types, or method signatures change.** All additions are strictly additive.

## Data Model

### New Table: `relations`

```sql
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
```

### Migration Strategy

The `CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS` statements are appended to the existing `migrate()` function in `store.go`. This is the same pattern used for all existing tables (`sessions`, `observations`, `user_prompts`). On an existing DB, the statements are no-ops. On a fresh DB, everything is created together.

`PRAGMA foreign_keys = ON` is already set (store.go:333), so `ON DELETE CASCADE` works natively. When an observation is hard-deleted, SQLite automatically removes all relations referencing it. No application-level cascade code needed.

## New Types (in `internal/memory/store.go`)

```go
// Relation represents a typed edge between two observations.
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
    ID           int64         `json:"id"`
    Title        string        `json:"title"`
    Type         string        `json:"type"`
    Project      *string       `json:"project,omitempty"`
    CreatedAt    string        `json:"created_at"`
    RelationType string        `json:"relation_type"`
    Direction    string        `json:"direction"` // "outgoing" or "incoming"
    Note         string        `json:"note,omitempty"`
    Depth        int           `json:"depth"`
    Children     []ContextNode `json:"children,omitempty"`
}

// ContextResult holds the full graph traversal output.
type ContextResult struct {
    Root       Observation   `json:"root"`
    Connected  []ContextNode `json:"connected"`
    TotalNodes int           `json:"total_nodes"`
    MaxDepth   int           `json:"max_depth"`
}
```

## New Store Methods

### `AddRelation(p AddRelationParams) ([]int64, error)`

1. Validate `from_id != to_id` (reject self-relations)
2. Validate both observations exist and are not hard-deleted (query `observations` table)
3. Insert into `relations` table
4. If `bidirectional == true`, insert reverse relation in same transaction
5. Return slice of created relation IDs (1 or 2)
6. On duplicate (`UNIQUE` constraint violation), return descriptive error

Uses `s.beginTxHook()` + `s.commitHook(tx)` for atomicity when bidirectional.

### `RemoveRelation(id int64) error`

1. `DELETE FROM relations WHERE id = ?`
2. Return error if no rows affected (relation not found)

### `GetRelations(observationID int64) ([]Relation, error)`

1. Query all relations where `from_id = ? OR to_id = ?`
2. Return as `[]Relation` — the caller determines direction by comparing IDs

### `BuildContext(observationID int64, maxDepth int) (*ContextResult, error)`

1. Get root observation via existing `GetObservation`
2. BFS traversal using a queue + visited set (`map[int64]bool`)
3. At each level (up to `maxDepth`), query relations for all nodes at current depth
4. Build `[]ContextNode` tree with depth, direction, and relation metadata
5. Cap `maxDepth` at 5 (clamp if higher)
6. Default `maxDepth` to 2 if <= 0
7. Return `ContextResult` with root, connected nodes, total count, and actual max depth reached

**BFS Algorithm (pseudocode):**
```
visited = {rootID}
queue = [(rootID, 0)]  // (observationID, currentDepth)
result = []

while queue is not empty:
    (nodeID, depth) = dequeue
    if depth >= maxDepth: continue
    
    relations = query relations WHERE from_id = nodeID OR to_id = nodeID
    for each relation:
        otherID = relation.from_id if relation.to_id == nodeID else relation.to_id
        direction = "outgoing" if relation.from_id == nodeID else "incoming"
        
        if otherID not in visited:
            visited.add(otherID)
            get observation title/type/project/created_at for otherID
            append to result as ContextNode{depth: depth+1, ...}
            enqueue (otherID, depth+1)

return ContextResult{root, result, len(visited)-1, maxDepthReached}
```

**Cycle detection**: The `visited` set prevents re-visiting nodes. A node appears at most once in the output, at the shallowest depth it was first reached (BFS property).

**Performance**: Each depth level is one SQL query per node at that level. For depth=2 with average 5 relations per node, this is ~30 queries total — well within the 1-second NFR for 1000 observations. Can be optimized later with batch queries if needed.

## New MCP Tools (in `internal/memtools/`)

### `relate.go` — `mem_relate` + `mem_unrelate`

**`mem_relate`** parameters:
- `from_id` (number, required) — source observation ID
- `to_id` (number, required) — target observation ID
- `relation_type` (string, required) — edge type (e.g., `implements`, `depends_on`, `caused_by`, `relates_to`, `supersedes`, `part_of`)
- `note` (string, optional) — context about the relation
- `bidirectional` (boolean, optional, default false) — create both directions

**`mem_unrelate`** parameters:
- `id` (number, required) — relation ID to remove

### `build_context.go` — `mem_build_context`

**`mem_build_context`** parameters:
- `observation_id` (number, required) — starting node
- `depth` (number, optional, default 2, max 5) — traversal depth

**Output format (markdown):**
```
# Context Graph for #42: "JWT auth middleware"

## Direct Relations (depth 1)
- → #38 [decision] "Switched from sessions to JWT" (implements)
- ← #45 [bugfix] "Fixed token expiry race condition" (caused_by)
- → #41 [architecture] "Auth module design" (part_of)

## Extended Relations (depth 2)
- → #38 → #33 [discovery] "Session storage scaling issues" (caused_by)
- ← #45 → #47 [pattern] "Retry with backoff for token refresh" (relates_to)

Total: 5 connected observations across 2 levels
```

## Modifications to Existing Code

### `store.go` — `migrate()` function
**Change**: Append `CREATE TABLE IF NOT EXISTS relations (...)` and indexes after the existing schema statements. Same location, same pattern — additive only.

### `store.go` — `DeleteObservation()` function  
**Change**: No code change needed. `ON DELETE CASCADE` with `PRAGMA foreign_keys = ON` handles this at the SQLite level. When an observation is hard-deleted (`DELETE FROM observations WHERE id = ?`), SQLite automatically deletes all rows in `relations` where `from_id` or `to_id` matches.

### `memtools/timeline.go` — `GetObservationTool.Handle()`
**Change**: After rendering the observation content, query `GetRelations(obs.ID)` and append a "Relations" section to the output. If no relations exist, omit the section entirely (backwards-compatible output).

```
## Relations

**Outgoing:**
- → #38 [decision] "Switched from sessions to JWT" (implements)

**Incoming:**
- ← #45 [bugfix] "Fixed token expiry race condition" (caused_by)
```

### `server.go` — `registerMemoryTools()`
**Change**: Register 3 new tools after existing memory tools:
```go
relateTool := memtools.NewRelateTool(ms)
s.AddTool(relateTool.Definition(), relateTool.Handle)

unrelateTool := memtools.NewUnrelateTool(ms)
s.AddTool(unrelateTool.Definition(), unrelateTool.Handle)

buildCtx := memtools.NewBuildContextTool(ms)
s.AddTool(buildCtx.Definition(), buildCtx.Handle)
```

### `server.go` — `serverInstructions()`
**Change**: Add a section about relation tools to the AI instructions, so the AI knows WHEN and HOW to use them. Key guidance:
- Use `mem_relate` after saving related observations
- Use `mem_build_context` when exploring a topic to understand its connections
- Suggest relation types based on observation types (decision → implements, bugfix → caused_by, etc.)

## Files Changed

| File | Change Type | Description |
|------|------------|-------------|
| `internal/memory/store.go` | Modified | New types, new methods, migration addition |
| `internal/memtools/relate.go` | **New** | `mem_relate` + `mem_unrelate` tool handlers |
| `internal/memtools/build_context.go` | **New** | `mem_build_context` tool handler |
| `internal/memtools/timeline.go` | Modified | `GetObservationTool` shows direct relations |
| `internal/server/server.go` | Modified | Register 3 new tools, update instructions |
| `internal/memory/store_test.go` or new test file | **New/Modified** | Tests for relation operations |
| `internal/memtools/memtools_test.go` | Modified | Tests for new tool handlers |

## What Does NOT Change

- `Observation` struct — no new fields
- `SearchResult` struct — no changes (FR-011 is SHOULD HAVE, deferred)
- `AddObservation` — no auto-relation creation
- `FormatContext` — unchanged
- `Export`/`Import` — unchanged (FR-W04)
- FTS5 virtual tables — untouched
- All existing 14 memory tools — identical behavior
- Change pipeline tools — untouched
- SDD pipeline tools — untouched
