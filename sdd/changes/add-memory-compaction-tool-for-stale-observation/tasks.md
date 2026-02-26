# Tasks: Memory Compaction Tool (F4)

## Estimated Effort
Half day for a single developer.

## Task Breakdown

### TASK-001: Add `FindStaleObservations` store method
**Component**: `internal/memory/store.go`
**Covers**: FR-001
**Dependencies**: None
**Description**: Add `FindStaleObservations(project, scope string, olderThanDays, limit int) ([]Observation, error)` that queries observations with `created_at < datetime('now', '-N days')`, excludes soft-deleted, orders by created_at ASC. Limit defaults to 50, cap at 200.
**Acceptance Criteria**:
- [ ] Returns observations older than N days
- [ ] Filters by project and scope when provided
- [ ] Excludes soft-deleted observations
- [ ] Orders oldest first
- [ ] Respects limit with default 50, max 200

### TASK-002: Add `CompactObservations` store method
**Component**: `internal/memory/store.go`
**Covers**: FR-002, FR-004, FR-005, FR-006, NFR-001, NFR-002
**Dependencies**: TASK-001
**Description**: Add `CompactObservations(params CompactParams) (*CompactResult, error)` that wraps batch soft-delete + optional summary creation in a single SQL transaction. CompactParams: IDs []int64, SummaryTitle, SummaryContent, Project, Scope, SessionID. CompactResult: DeletedCount int, SummaryID *int64, TotalBefore int, TotalAfter int. Validates: IDs non-empty, IDs exist and not already deleted.
**Acceptance Criteria**:
- [ ] Soft-deletes all provided IDs in one transaction
- [ ] Creates summary observation with type "compaction_summary" if content provided
- [ ] Rolls back if summary creation fails (atomicity)
- [ ] Returns before/after counts using CountObservations
- [ ] Validates IDs are non-empty and not already deleted
- [ ] Completes in < 500ms for 200 observations

### TASK-003: Write store tests for FindStaleObservations and CompactObservations
**Component**: `internal/memory/store_test.go`
**Covers**: FR-001, FR-002, FR-004, FR-005, NFR-002
**Dependencies**: TASK-001, TASK-002
**Description**: Test cases: empty store, with data at various ages, project/scope filtering, limit cap, compact with summary, compact without summary, compact already-deleted IDs (error), empty IDs (error), transaction rollback on failure.
**Acceptance Criteria**:
- [ ] Test FindStaleObservations with no stale data
- [ ] Test FindStaleObservations with stale data, project filter, limit
- [ ] Test CompactObservations with summary
- [ ] Test CompactObservations without summary
- [ ] Test CompactObservations with invalid IDs (error)
- [ ] Test CompactObservations with empty IDs (error)

### TASK-004: Add `mem_compact` tool handler
**Component**: `internal/memtools/compact.go`
**Covers**: FR-002, FR-003, FR-004, FR-005
**Dependencies**: TASK-001, TASK-002
**Description**: Create `CompactTool` struct with `Definition()` and `Handle()`. Dual behavior: without `compact_ids` → identify mode (call FindStaleObservations, format candidates list). With `compact_ids` → execute mode (parse JSON array, call CompactObservations, format result). Parse `compact_ids` as JSON `[]int64`.
**Acceptance Criteria**:
- [ ] Identify mode returns formatted list of stale observations
- [ ] Execute mode soft-deletes and optionally creates summary
- [ ] Returns before/after counts
- [ ] Validates older_than_days > 0
- [ ] Validates compact_ids is valid JSON array
- [ ] Returns error when summary_content without summary_title

### TASK-005: Write handler tests for mem_compact
**Component**: `internal/memtools/memtools_test.go`
**Covers**: FR-002, FR-003, FR-004, FR-005
**Dependencies**: TASK-004
**Description**: Test identify mode (empty store, with stale data, with filters), execute mode (with summary, without summary, invalid IDs, missing older_than_days).
**Acceptance Criteria**:
- [ ] Test identify mode with no stale data
- [ ] Test identify mode with stale data
- [ ] Test execute mode with summary
- [ ] Test execute mode without summary
- [ ] Test missing older_than_days (error)
- [ ] Test invalid compact_ids JSON (error)

### TASK-006: Register tool and update server instructions + docs
**Component**: `internal/server/server.go`, `docs/research-foundations.md`, `README.md`, `AGENTS.md`, `docs/tool-reference.md`
**Covers**: FR-007, FR-008, FR-009
**Dependencies**: TASK-004
**Description**: Register CompactTool in `registerMemoryTools()`. Update serverInstructions with compaction workflow guidance. Update tool counts from 33 to 34. Add F4 entry to research-foundations.md.
**Acceptance Criteria**:
- [ ] Tool registered and accessible via MCP
- [ ] Server instructions document two-phase workflow
- [ ] Tool counts updated in README, AGENTS.md, tool-reference.md
- [ ] Research-foundations.md has F4 entry

## Execution Waves

**Wave 1** (no dependencies):
- TASK-001: FindStaleObservations store method

**Wave 2** (depends on Wave 1):
- TASK-002: CompactObservations store method
- TASK-003: Store tests (can write alongside TASK-002)

**Wave 3** (depends on Wave 2):
- TASK-004: mem_compact tool handler
- TASK-005: Handler tests

**Wave 4** (depends on Wave 3):
- TASK-006: Registration, docs, tool counts
