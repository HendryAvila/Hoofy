# Context Check: Navigation Hints & Token-Efficient Defaults

## Impact Analysis

### Direct Impact — Files That Must Change

| File | Change Type | Rationale |
|------|-------------|-----------|
| `internal/memory/store.go` | ADD methods | Need `CountObservations()` and `CountSearchResults()` for "Showing X of Y" |
| `internal/memory/detail_level.go` | ADD constant | New `NavigationHint()` helper function |
| `internal/memtools/search.go` | MODIFY | Add total count query and navigation footer |
| `internal/memtools/context.go` | MODIFY | Add total count and navigation footer |
| `internal/memtools/timeline.go` | MODIFY | Already has `TotalInRange` — surface it as navigation hint |
| `internal/tools/context.go` | MODIFY | Change default `detail_level` from `standard` to `summary` |
| `internal/server/server.go` | MODIFY | Update server instructions to document new defaults |

### Indirect Impact — Files That Need Updates

| File | Change Type | Rationale |
|------|-------------|-----------|
| `internal/memtools/memtools_test.go` | ADD tests | Navigation hint formatting tests |
| `internal/memory/detail_level_test.go` | ADD tests | NavigationHint helper tests |
| `internal/memory/store_test.go` | ADD tests | CountObservations/CountSearchResults tests |
| `docs/research-foundations.md` | UPDATE | Add F3 research mapping |

### No Impact
- Tool count stays at 33 (no new tools, just enhanced responses)
- No new dependencies
- No schema changes
- No breaking changes to tool signatures

## Prior Changes Overlap

### F1: Response Verbosity Control (COMPLETED)
- **Relationship**: F3 BUILDS ON F1. F1 added `detail_level` parameter to 5 tools. F3 enhances those same tools with navigation hints and changes the default for `sdd_get_context`.
- **Risk**: None — F3 is additive, doesn't modify F1's core logic.

### F2: Progress Tracking (COMPLETED)
- **Relationship**: Independent — `mem_progress` doesn't return lists that need navigation hints.
- **Risk**: None.

## Convention Compliance

- Follows SRP: count queries go in Store, formatting goes in tools
- Follows established pattern: shared helpers in `internal/memory/detail_level.go`
- Test patterns: `newTestStore(t)`, `seedManualSession(t, store)`, `makeReq()`

## Requirements Smells Check (IEEE 29148)

No ambiguity detected:
- "Navigation hints" is well-defined: `"Showing X of Y. Use detail_level=full or mem_get_observation to see more."`
- "Token-efficient defaults" is concrete: change one default value from `standard` to `summary`
- All affected tools are explicitly enumerated
