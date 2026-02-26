# Context Check: Sub-Agent Memory Scoping

## Impact Analysis

### Existing Architecture Compatibility

**LOW RISK** — This change adds a new optional column and filter parameter. No existing behavior changes.

1. **Database schema**: Need to add `namespace` column to `observations` table. Uses `CREATE INDEX IF NOT EXISTS` migration pattern (same as relations, scope, topic_key). No destructive changes.

2. **Existing `scope` vs new `namespace`**: 
   - `scope` = visibility level (project vs personal) — WHO should see it
   - `namespace` = isolation boundary for parallel agents — WHICH AGENT owns it
   - These are orthogonal. A sub-agent can save a "personal" scoped observation within its own namespace.

3. **Store methods affected**: All methods that filter by `project` and `scope` also need optional `namespace` filtering:
   - `RecentObservations()` — used by `mem_context`
   - `Search()` — used by `mem_search`
   - `FindStaleObservations()` — used by `mem_compact`
   - `CountObservations()`, `CountSearchResults()` — used for navigation hints
   - `AddObservation()` — needs to accept namespace
   - `FormatContextDetailed()` — used by `mem_context`

4. **Tool handlers affected** (need `namespace` parameter):
   - Write tools: `mem_save`, `mem_save_prompt`, `mem_session_start`, `mem_session_summary`, `mem_progress`
   - Read tools: `mem_search`, `mem_context`, `mem_timeline`, `mem_compact`
   - NOT affected: `mem_get_observation`, `mem_relate`, `mem_unrelate`, `mem_build_context`, `mem_stats`, `mem_delete`, `mem_update`, `mem_suggest_topic_key`, `mem_capture_passive`

### Prior Changes Relevance

| Change | Relevance |
|--------|-----------|
| F4: Memory Compaction | Medium — `mem_compact` needs namespace filter for `FindStaleObservations` |
| F3: Navigation Hints | Low — `CountObservations` needs namespace param, but pattern is same |
| F1: Response Verbosity | None — detail_level is orthogonal to namespace |
| F2: Progress Tracking | Medium — `mem_progress` uses topic_key pattern; namespace adds isolation layer |
| Knowledge Graph Relations | Low — relations are between specific observation IDs, not namespace-scoped |

### Requirements Smells (IEEE 29148)

- **Ambiguity**: "namespace convention" is guidance, not enforced — acceptable since AI agents self-organize
- **Completeness**: All read/write tools covered in the proposal
- **Testability**: Namespace filtering is a standard SQL WHERE clause — easily testable

### Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Schema migration on existing DBs | Low | `ALTER TABLE ... ADD COLUMN` with default NULL, same pattern used for other columns |
| Tool parameter explosion | Low | `namespace` is optional, omitting it = no filter (backwards compat) |
| FTS5 index doesn't include namespace | Medium | Filter namespace in SQL WHERE after FTS5 MATCH, same as `scope` pattern |
| Breaking existing tests | None | All tests omit namespace → no filter → same behavior |
