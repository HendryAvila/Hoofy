# Tasks: Token-Efficient Tool Responses

## Task Breakdown

### TASK-001: Add NavigationHint helper to memory package
**Covers**: FR-007
**File**: `internal/memory/detail_level.go`
**Description**: Add `NavigationHint(showing, total int, hint string) string`. Returns empty when showing >= total. Returns `"\n📊 Showing {showing} of {total}. {hint}"` when capped.
**Tests**: Add to `internal/memory/detail_level_test.go`
**AC**:
- Returns empty string when showing >= total
- Returns formatted hint when showing < total
- Returns empty string when total is 0

### TASK-002: Add CountObservations method to Store
**Covers**: FR-001
**File**: `internal/memory/store.go`
**Description**: `CountObservations(project, scope string) (int, error)` — SELECT COUNT(*) from observations with project/scope filters, excluding soft-deleted.
**Tests**: Add to `internal/memory/store_test.go`
**AC**:
- Returns correct count for unfiltered
- Returns correct count filtered by project
- Returns correct count filtered by scope
- Excludes soft-deleted observations

### TASK-003: Add CountSearchResults method to Store
**Covers**: FR-002
**File**: `internal/memory/store.go`
**Description**: `CountSearchResults(query string, opts SearchOptions) (int, error)` — Same FTS5 JOIN + WHERE as Search() but SELECT COUNT(*) instead of rows. Uses same sanitizeFTS, same filters.
**Tests**: Add to `internal/memory/store_test.go`
**AC**:
- Returns total matching count (not capped by limit)
- Handles empty query (falls back to count all matching observations)
- Applies type/project/scope filters

### TASK-004: Add navigation hints to mem_search
**Covers**: FR-003
**File**: `internal/memtools/search.go`
**Description**: After getting results, call CountSearchResults to get total. If len(results) < total, append NavigationHint.
**Tests**: Add to `internal/memtools/memtools_test.go`
**AC**:
- Shows "Showing X of Y" when results are capped
- Does NOT show hint when all results fit within limit
- Hint includes guidance to use mem_get_observation

### TASK-005: Add navigation hints to mem_context
**Covers**: FR-004
**File**: `internal/memtools/context.go`
**Description**: After getting formatted context, call CountObservations to get total. If shown < total, append NavigationHint.
**Tests**: Add to `internal/memtools/memtools_test.go`
**AC**:
- Shows hint when observations are capped
- Does NOT show hint when all observations are returned
- Works correctly with project filter

### TASK-006: Add navigation hints to mem_timeline
**Covers**: FR-005
**File**: `internal/memtools/timeline.go`
**Description**: Use existing TotalInRange from TimelineResult. If before+1+after < TotalInRange, append NavigationHint.
**Tests**: Add to `internal/memtools/memtools_test.go`
**AC**:
- Shows hint when timeline window is smaller than session total
- Does NOT show hint when entire session fits in window

### TASK-007: Change sdd_get_context default to summary
**Covers**: FR-006
**File**: `internal/tools/context.go`
**Description**: Change `req.GetString("detail_level", "standard")` to `req.GetString("detail_level", "summary")`. Update tool description text to reflect new default.
**Tests**: Update existing test in `internal/tools/context_test.go` if it checks default behavior
**AC**:
- Default behavior returns summary output
- Explicit `detail_level=standard` still works
- Explicit `detail_level=full` still works

### TASK-008: Update server instructions and docs
**Covers**: FR-008
**File**: `internal/server/server.go`, `docs/research-foundations.md`
**Description**: Update serverInstructions to mention navigation hints and new sdd_get_context default. Add F3 to research-foundations.md.
**AC**:
- Server instructions mention navigation hints
- Server instructions mention sdd_get_context default is now summary
- research-foundations.md has F3 entry with Anthropic source links

## Dependency Graph

```
TASK-001 (NavigationHint helper)
TASK-002 (CountObservations)
TASK-003 (CountSearchResults)
    ↓ (all three are independent, can parallel)
TASK-004 (mem_search hints) depends on TASK-001, TASK-003
TASK-005 (mem_context hints) depends on TASK-001, TASK-002
TASK-006 (mem_timeline hints) depends on TASK-001
TASK-007 (sdd_get_context default) — independent
    ↓
TASK-008 (docs) depends on all above
```

## Waves

**Wave 1** (parallel):
- TASK-001, TASK-002, TASK-003, TASK-007

**Wave 2** (parallel, depends on Wave 1):
- TASK-004, TASK-005, TASK-006

**Wave 3** (sequential):
- TASK-008

## Estimated Effort
3-4 hours for a single developer. All changes are mechanical — well-defined patterns from F1.
