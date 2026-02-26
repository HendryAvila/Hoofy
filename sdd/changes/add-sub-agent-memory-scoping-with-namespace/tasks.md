# Tasks: Sub-Agent Memory Scoping

## TASK-001: Schema migration — add namespace column
**Covers**: FR-001, NFR-002, NFR-004
**Dependencies**: None
**Description**: Add `namespace TEXT DEFAULT NULL` column to `observations` table and `user_prompts` table. Add index `idx_obs_namespace` on `observations(namespace)`. Migration must be non-destructive — check if column exists before ALTER TABLE.
**Acceptance Criteria**:
- [ ] `namespace` column exists on `observations` table after migration
- [ ] `namespace` column exists on `user_prompts` table after migration
- [ ] Index `idx_obs_namespace` created
- [ ] Existing databases upgrade cleanly (no data loss)
- [ ] New databases include column from initial schema

## TASK-002: Store layer — AddObservation and AddPrompt namespace support
**Covers**: FR-002, FR-007
**Dependencies**: TASK-001
**Description**: Add `Namespace string` field to `AddObservationParams` and `AddPromptParams`. Wire through to INSERT queries. When non-empty, store value; when empty, store NULL.
**Acceptance Criteria**:
- [ ] `AddObservationParams.Namespace` field exists
- [ ] `AddPromptParams.Namespace` field exists
- [ ] INSERT queries include `namespace` column
- [ ] Empty namespace stored as NULL

## TASK-003: Store layer — query methods gain namespace filter
**Covers**: FR-003, FR-004, FR-005
**Dependencies**: TASK-001
**Description**: Update all store query methods to accept and filter by namespace:
- `RecentObservations(project, scope, namespace, limit)`
- `FindStaleObservations(project, scope, namespace, olderThanDays, limit)`
- `CountObservations(project, scope, namespace)`
- `SearchOptions.Namespace string`
- `FormatContextDetailed(project, scope, namespace, opts)`
- `FormatContext(project, scope, namespace)` (wrapper)
- Timeline query methods
**Acceptance Criteria**:
- [ ] All methods accept namespace parameter
- [ ] When namespace is non-empty, `AND namespace = ?` added to WHERE clause
- [ ] When namespace is empty, no filter applied
- [ ] All callers updated (handlers that call these methods)
- [ ] Existing tests pass without modification (NFR-001)

## TASK-004: Store tests — namespace filtering
**Covers**: FR-003, FR-004, NFR-001
**Dependencies**: TASK-002, TASK-003
**Description**: Add tests for namespace filtering across store methods:
- AddObservation with namespace → RecentObservations filtered by namespace
- Search with namespace filter
- CountObservations with namespace
- FindStaleObservations with namespace
- Verify omitting namespace returns all observations
**Acceptance Criteria**:
- [ ] At least 5 new store tests covering namespace filtering
- [ ] Test that empty namespace returns all observations (backwards compat)
- [ ] All existing tests still pass

## TASK-005: Tool handlers — add namespace parameter to write tools
**Covers**: FR-006, FR-007, FR-008, FR-009
**Dependencies**: TASK-002
**Description**: Add `namespace` string parameter to:
- `mem_save` → pass to `AddObservation`
- `mem_save_prompt` → pass to `AddPrompt`
- `mem_session_summary` → pass to `AddObservation`
- `mem_progress` → modify topic_key to `progress/<namespace>/<project>` when namespace provided
**Acceptance Criteria**:
- [ ] All 4 tools accept `namespace` parameter
- [ ] Parameter description includes usage example
- [ ] `mem_progress` generates namespaced topic_key

## TASK-006: Tool handlers — add namespace parameter to read tools
**Covers**: FR-010, FR-011, FR-012, FR-013
**Dependencies**: TASK-003
**Description**: Add `namespace` string parameter to:
- `mem_search` → pass to `SearchOptions.Namespace`
- `mem_context` → pass to `FormatContextDetailed()` and `CountObservations()`
- `mem_timeline` → pass to timeline query
- `mem_compact` → pass to `FindStaleObservations()`
**Acceptance Criteria**:
- [ ] All 4 tools accept `namespace` parameter
- [ ] Parameter description includes usage example
- [ ] Namespace filters work correctly in each tool

## TASK-007: Handler tests — namespace parameter validation
**Covers**: FR-006 through FR-013
**Dependencies**: TASK-005, TASK-006
**Description**: Add handler tests for namespace parameter behavior in memtools_test.go. Test both with and without namespace to verify backwards compatibility.
**Acceptance Criteria**:
- [ ] At least 4 handler tests covering namespace in write/read tools
- [ ] Verify backward compatibility (no namespace = same behavior)

## TASK-008: Documentation and server instructions
**Covers**: FR-014, FR-015, FR-016
**Dependencies**: TASK-005, TASK-006
**Description**: 
- Update `serverInstructions()` in `server.go` with sub-agent scoping section
- Update `docs/research-foundations.md` with F5 entry
- Update `docs/tool-reference.md` with namespace parameter on affected tools
**Acceptance Criteria**:
- [ ] Server instructions include namespace workflow guidance
- [ ] research-foundations.md has F5 entry citing Multi-Agent Research System
- [ ] tool-reference.md shows namespace param on all affected tools

## Dependency Graph

```
TASK-001 → TASK-002 → TASK-005 → TASK-007
TASK-001 → TASK-003 → TASK-006 → TASK-007
TASK-002 + TASK-003 → TASK-004
TASK-005 + TASK-006 → TASK-008
```

## Wave Assignments

**Wave 1**: TASK-001 (schema migration)
**Wave 2**: TASK-002, TASK-003 (store layer, parallel)
**Wave 3**: TASK-004, TASK-005, TASK-006 (tests + handlers, parallel)
**Wave 4**: TASK-007, TASK-008 (handler tests + docs, parallel)

## Estimated Effort

8 tasks, ~2-3 hours for single developer. Most work is mechanical (adding a parameter to N methods/tools).
