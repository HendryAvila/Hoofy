# Tasks: Knowledge Graph Relations Between Memory Observations

## Task Breakdown

### TASK-001: Add relations table migration + Relation types
**Component**: `internal/memory/store.go`
**Covers**: FR-001, FR-002, NFR-001, NFR-004, NFR-005
**Dependencies**: None
**Wave**: 1
**Description**: Add the `relations` table, indexes, and unique constraint to the `migrate()` function. Add `Relation`, `AddRelationParams`, `ContextNode`, and `ContextResult` types to the types section of `store.go`.
**Acceptance Criteria**:
- [ ] `CREATE TABLE IF NOT EXISTS relations (...)` added to `migrate()` after existing schema
- [ ] All 4 indexes created (`idx_rel_from`, `idx_rel_to`, `idx_rel_type`, `idx_rel_unique`)
- [ ] `ON DELETE CASCADE` foreign keys reference `observations(id)`
- [ ] Running `migrate()` on existing DB with no `relations` table creates it
- [ ] Running `migrate()` on DB that already has `relations` table is a no-op
- [ ] All new types defined (`Relation`, `AddRelationParams`, `ContextNode`, `ContextResult`)
- [ ] `go build` succeeds with `CGO_ENABLED=0`
- [ ] All existing tests pass without modification

---

### TASK-002: Implement AddRelation Store method
**Component**: `internal/memory/store.go`
**Covers**: FR-003, FR-008, FR-009, FR-010, FR-013, NFR-006
**Dependencies**: TASK-001
**Wave**: 2
**Description**: Implement `AddRelation(p AddRelationParams) ([]int64, error)` on the Store. Validates both observations exist and are not hard-deleted, rejects self-relations, handles duplicate detection via UNIQUE constraint, and supports bidirectional creation in a transaction.
**Acceptance Criteria**:
- [ ] `AddRelation` validates `from_id != to_id` and returns error for self-relations
- [ ] `AddRelation` validates both observations exist (not hard-deleted)
- [ ] Single relation insert works and returns `[]int64{relationID}`
- [ ] Bidirectional flag creates both directions in one transaction, returns `[]int64{id1, id2}`
- [ ] Duplicate relation (same from_id, to_id, type) returns descriptive error
- [ ] Uses `s.execHook` / `s.beginTxHook` / `s.commitHook` pattern

---

### TASK-003: Implement RemoveRelation and GetRelations Store methods
**Component**: `internal/memory/store.go`
**Covers**: FR-004, FR-007, NFR-006
**Dependencies**: TASK-001
**Wave**: 2
**Description**: Implement `RemoveRelation(id int64) error` (hard delete) and `GetRelations(observationID int64) ([]Relation, error)` (bidirectional query). Verify that `ON DELETE CASCADE` works for hard-deleted observations.
**Acceptance Criteria**:
- [ ] `RemoveRelation` deletes relation by ID
- [ ] `RemoveRelation` returns error if relation not found
- [ ] `GetRelations` returns all relations where observation is `from_id` or `to_id`
- [ ] Hard-deleting an observation cascades to remove its relations
- [ ] Uses `s.execHook` / `s.queryHook` pattern

---

### TASK-004: Implement BuildContext Store method (BFS graph traversal)
**Component**: `internal/memory/store.go`
**Covers**: FR-005, FR-012, NFR-003, NFR-006
**Dependencies**: TASK-002, TASK-003
**Wave**: 3
**Description**: Implement `BuildContext(observationID int64, maxDepth int) (*ContextResult, error)`. BFS traversal with visited set for cycle detection. Default depth 2, max 5. Returns lightweight `ContextNode` tree (no full content).
**Acceptance Criteria**:
- [ ] Traverses graph bidirectionally using BFS
- [ ] Cycle detection prevents infinite loops
- [ ] `maxDepth` defaults to 2 when <= 0
- [ ] `maxDepth` clamped to 5 when > 5
- [ ] Each `ContextNode` includes `id`, `title`, `type`, `project`, `created_at`, `relation_type`, `direction`, `depth`
- [ ] Nodes appear at shallowest depth (BFS property)
- [ ] Returns `ContextResult` with root observation, connected nodes, total count, max depth reached
- [ ] Completes in <1s for depth=5 on a graph with 1000 observations

---

### TASK-005: Create mem_relate and mem_unrelate MCP tools
**Component**: `internal/memtools/relate.go` (new file)
**Covers**: FR-003, FR-004, FR-013, NFR-007
**Dependencies**: TASK-002, TASK-003
**Wave**: 3
**Description**: Create new file `relate.go` with `RelateTool` and `UnrelateTool` structs. Follow existing memtools pattern (struct + constructor + Definition + Handle). Wire parameter extraction using existing `intArg`/`boolArg` helpers.
**Acceptance Criteria**:
- [ ] `mem_relate` tool with parameters: `from_id` (number, required), `to_id` (number, required), `relation_type` (string, required), `note` (string, optional), `bidirectional` (boolean, optional)
- [ ] `mem_unrelate` tool with parameter: `id` (number, required)
- [ ] Both tools return descriptive success/error messages
- [ ] `mem_relate` output includes created relation ID(s)
- [ ] Input validation errors return `mcp.NewToolResultError` (not Go errors)
- [ ] Follows same pattern as `manage.go` and `save.go`

---

### TASK-006: Create mem_build_context MCP tool
**Component**: `internal/memtools/build_context.go` (new file)
**Covers**: FR-005, FR-012, NFR-007
**Dependencies**: TASK-004
**Wave**: 3
**Description**: Create new file `build_context.go` with `BuildContextTool` struct. Formats the `ContextResult` as a readable markdown output grouped by depth level.
**Acceptance Criteria**:
- [ ] `mem_build_context` tool with parameters: `observation_id` (number, required), `depth` (number, optional, default 2)
- [ ] Output format: markdown with root info, relations grouped by depth level, total summary
- [ ] Shows direction arrows (`→` outgoing, `←` incoming) and relation type per edge
- [ ] When no relations exist, returns clear message (not an error)
- [ ] Follows same pattern as `timeline.go`

---

### TASK-007: Enhance GetObservationTool to show direct relations
**Component**: `internal/memtools/timeline.go`
**Covers**: FR-006
**Dependencies**: TASK-003
**Wave**: 3
**Description**: Modify `GetObservationTool.Handle()` to call `GetRelations(obs.ID)` after rendering the existing output. Append a "Relations" section with outgoing and incoming relations. If no relations exist, omit the section entirely (backwards-compatible).
**Acceptance Criteria**:
- [ ] Observations WITH relations show a "## Relations" section
- [ ] Observations WITHOUT relations produce identical output to before (no empty section)
- [ ] Relations grouped into "Outgoing" and "Incoming" subsections
- [ ] Each relation shows: direction arrow, target ID, target type, target title, relation type
- [ ] Existing test assertions still pass

---

### TASK-008: Register new tools in server.go + update instructions
**Component**: `internal/server/server.go`
**Covers**: NFR-007
**Dependencies**: TASK-005, TASK-006
**Wave**: 4
**Description**: Register `mem_relate`, `mem_unrelate`, and `mem_build_context` in `registerMemoryTools()`. Add relation tool guidance to `serverInstructions()`.
**Acceptance Criteria**:
- [ ] All 3 new tools registered in `registerMemoryTools()`
- [ ] `serverInstructions()` includes guidance on when/how to use relation tools
- [ ] Tool count increases from 14 to 17 memory tools
- [ ] Server starts without errors

---

### TASK-009: Write tests for all relation functionality
**Component**: `internal/memory/store_test.go` (or new `relations_test.go`), `internal/memtools/memtools_test.go`
**Covers**: FR-001 through FR-013, NFR-001, NFR-002, NFR-003
**Dependencies**: TASK-001 through TASK-008
**Wave**: 5
**Description**: Comprehensive test coverage for relation Store methods and MCP tool handlers. Test migration idempotency, CRUD operations, validation rules, BFS traversal, cycle detection, cascade delete, bidirectional creation, and tool input/output formatting.
**Acceptance Criteria**:
- [ ] Test: migration on fresh DB creates relations table
- [ ] Test: migration on existing DB (no relations table) adds it without data loss
- [ ] Test: migration on DB with relations table is idempotent
- [ ] Test: AddRelation creates relation and returns ID
- [ ] Test: AddRelation with bidirectional creates two relations
- [ ] Test: AddRelation rejects self-relation (from_id == to_id)
- [ ] Test: AddRelation rejects duplicate relation
- [ ] Test: AddRelation rejects non-existent observation ID
- [ ] Test: RemoveRelation deletes by ID
- [ ] Test: RemoveRelation returns error for non-existent ID
- [ ] Test: GetRelations returns outgoing and incoming
- [ ] Test: Hard-delete observation cascades to relations
- [ ] Test: Soft-delete observation does NOT affect relations
- [ ] Test: BuildContext with depth=1 returns direct connections only
- [ ] Test: BuildContext with depth=2 returns two levels
- [ ] Test: BuildContext cycle detection (A→B→C→A does not loop)
- [ ] Test: BuildContext depth clamping (>5 becomes 5, <=0 becomes 2)
- [ ] Test: MCP tool handlers validate required parameters
- [ ] Test: MCP tool handlers format output correctly
- [ ] Test: `go test -race ./...` passes with all new + existing tests

## Dependency Graph

```
TASK-001 (schema + types)
├── TASK-002 (AddRelation)      ─┐
├── TASK-003 (RemoveRelation)   ─┤── TASK-004 (BuildContext) ──┐
│                                ├── TASK-005 (mem_relate tool) ┤
│                                └── TASK-007 (enhance get_obs) │
│                                                               ├── TASK-008 (server wiring) ──→ TASK-009 (tests)
│                                    TASK-006 (mem_build_ctx)  ─┘
```

**Wave 1**: TASK-001 (foundation)
**Wave 2**: TASK-002, TASK-003 (parallel — both need schema, independent of each other)
**Wave 3**: TASK-004, TASK-005, TASK-006, TASK-007 (parallel — each depends on wave 2, not each other)
**Wave 4**: TASK-008 (wiring — needs tools from wave 3)
**Wave 5**: TASK-009 (tests — needs everything)

## Estimated Effort

9 tasks, ~2-3 days for a single developer. Wave parallelization could compress to ~1.5 days with two devs.

## Global Acceptance Criteria

- All code passes `go test -race -cover ./...`
- All code passes `golangci-lint run`
- Binary builds with `CGO_ENABLED=0` on all 6 platforms
- Existing users upgrading experience zero breakage
- No new dependencies in `go.mod`
