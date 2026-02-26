# Tasks: sdd_explore Implementation

## Estimated Effort
1-2 days for a single developer (6 tasks, all focused on a single tool + integration)

## Task Breakdown

### TASK-001: Add `explore` family to `inferTopicFamily()`
**Component**: memory/store.go
**Covers**: FR-002 (topic key generation)
**Dependencies**: None
**Description**: Add a new case to the `inferTopicFamily()` switch statement in `store.go` to map the `explore` observation type to its own topic family. This ensures `SuggestTopicKey("explore", "User auth system", "")` returns `explore/user-auth-system` instead of falling back to content heuristics.

**Implementation**:
- Add case `"explore", "exploration", "context", "discuss"` → return `"explore"` in the switch at line ~1827
- Add a test in `store_test.go` for `SuggestTopicKey` with `type=explore`

**Acceptance Criteria**:
- [ ] `SuggestTopicKey("explore", "My Feature", "")` returns `explore/my-feature`
- [ ] `SuggestTopicKey("exploration", "My Feature", "")` returns `explore/my-feature`
- [ ] `SuggestTopicKey("discuss", "My Feature", "")` returns `explore/my-feature`
- [ ] Existing topic key tests still pass
- [ ] No changes to any existing family mappings

---

### TASK-002: Create `ExploreTool` struct and `Definition()`
**Component**: internal/tools/explore.go
**Covers**: FR-001 (tool parameters), FR-006 (project param), NFR-003 (tool pattern)
**Dependencies**: None (can parallel with TASK-001)
**Description**: Create the new file `internal/tools/explore.go` with the `ExploreTool` struct, `NewExploreTool()` constructor, and `Definition()` method. The definition declares all 10 parameters (title required, 6 content params optional, project/scope/session_id optional).

**Implementation**:
- Struct with `store *memory.Store`
- Constructor `NewExploreTool(store *memory.Store) *ExploreTool`
- `Definition()` returns `mcp.NewTool("sdd_explore", ...)` with all params
- Tool description emphasizes standalone + pre-pipeline use

**Acceptance Criteria**:
- [ ] File compiles with no errors
- [ ] Definition includes all 10 parameters from FR-001/FR-006
- [ ] `title` is marked as `mcp.Required()`
- [ ] All 6 content params are optional
- [ ] Description mentions pre-pipeline context capture purpose

---

### TASK-003: Implement `Handle()` — core logic
**Component**: internal/tools/explore.go
**Covers**: FR-001 (validation), FR-002 (save with type=explore), FR-003 (upsert), FR-005 (response), FR-007 (accumulated context), FR-008 (type/size suggestion), NFR-001 (performance), NFR-004 (readability)
**Dependencies**: TASK-001 (needs explore topic family), TASK-002 (needs struct/definition)
**Description**: Implement the full Handle() method with:

1. **Validation**: title required, at least one content param non-empty
2. **Content formatting**: Assemble markdown from provided params (## Goals, ## Constraints, etc.)
3. **Topic key generation**: Use `memory.SuggestTopicKey("explore", title, "")` 
4. **Upsert with merge**: 
   - Try to find existing observation via AddObservation's topic_key upsert
   - BUT: to merge content, we need to read-then-write:
     a. Search for existing observation with same topic_key
     b. If found: parse existing content, merge new sections over old, format merged content
     c. If not found: use new content as-is
     d. Call AddObservation with the (merged) content
5. **Type/Size suggestion**: Run `suggestChangeType()` heuristic on merged text
6. **Response formatting**: Build structured response with title, topic key, action (create/update), content summary, and suggestion

**Private helper functions**:
- `formatExploreContent(sections map[string]string) string` — formats section map to markdown
- `parseExploreContent(markdown string) map[string]string` — parses markdown back to section map
- `mergeExploreSections(existing, new map[string]string) map[string]string` — merges (new overrides existing)
- `suggestChangeType(text string) (suggestedType, suggestedSize, reasoning string)` — keyword heuristic

**Acceptance Criteria**:
- [ ] Returns error when title is empty
- [ ] Returns error when all content params are empty
- [ ] First call creates new observation with type=explore
- [ ] Second call with same title upserts (updates existing)
- [ ] Upsert merges: new non-empty fields override, empty fields preserve existing
- [ ] Response shows ALL accumulated context, not just new additions
- [ ] Response includes type/size suggestion when keywords are present
- [ ] Response includes "no strong signal" note when keywords are absent
- [ ] Content is human-readable markdown (NFR-004)

---

### TASK-004: Write tests for `ExploreTool`
**Component**: internal/tools/explore_test.go
**Covers**: All FRs, NFR-002 (no pipeline impact)
**Dependencies**: TASK-003 (needs Handle implementation)
**Description**: Comprehensive test suite for the explore tool.

**Test cases**:
1. `TestExploreTool_Definition` — verifies tool name and parameter count
2. `TestExploreTool_Handle_BasicSave` — single call with goals+constraints, verify observation saved
3. `TestExploreTool_Handle_TitleRequired` — empty title returns error
4. `TestExploreTool_Handle_AtLeastOneContentField` — all content params empty returns error
5. `TestExploreTool_Handle_Upsert` — two calls with same title, verify content merged
6. `TestExploreTool_Handle_UpsertMerge` — first call with goals, second with constraints, verify both present
7. `TestExploreTool_Handle_UpsertOverride` — first call with goals="A", second with goals="B", verify goals="B"
8. `TestExploreTool_Handle_AllFields` — all 6 content params provided, all appear in content
9. `TestExploreTool_Handle_TypeSuggestion_Fix` — goals="fix the login bug" → suggests type=fix
10. `TestExploreTool_Handle_TypeSuggestion_Feature` — goals="add dark mode" → suggests type=feature
11. `TestExploreTool_Handle_TypeSuggestion_NoSignal` — goals="think about the system" → default suggestion
12. `TestExploreTool_Handle_ProjectParam` — project param is passed through to observation
13. `TestExploreTool_Handle_ScopeDefault` — scope defaults to "project"

**Acceptance Criteria**:
- [ ] All 13 test cases pass
- [ ] Tests use `t.TempDir()` for SQLite isolation (same pattern as memtools_test.go)
- [ ] No test modifies or depends on pipeline state
- [ ] Tests cover both create and upsert paths
- [ ] Tests verify response content (not just error/no-error)

---

### TASK-005: Register in `server.go` and update `serverInstructions()`
**Component**: internal/server/server.go
**Covers**: FR-004 (AI guidance), NFR-002 (no pipeline impact)
**Dependencies**: TASK-002 (needs constructor to exist)
**Description**: 
1. Register `ExploreTool` in the memory tools section of `server.go` (inside `if memErr == nil` block, after the bridge wiring)
2. Add the "PRE-PIPELINE EXPLORATION" section to `serverInstructions()` before the "## Pipeline" section

**Registration code** (inside `if memErr == nil`):
```go
// --- Register explore tool (SDD + Memory hybrid) ---
exploreTool := tools.NewExploreTool(memStore)
s.AddTool(exploreTool.Definition(), exploreTool.Handle)
```

**serverInstructions() update**:
- Add new section between "## WHEN TO ACTIVATE Hoofy" and "## What is SDD?"
- Content per design doc: when to use, how to use, important notes
- Update the "You do NOT need to activate Hoofy for" list to mention that sdd_explore CAN be used for these cases (it's always optional)

**Acceptance Criteria**:
- [ ] ExploreTool is registered and available when memory is enabled
- [ ] ExploreTool is NOT registered when memory is disabled (graceful degradation)
- [ ] serverInstructions includes the PRE-PIPELINE EXPLORATION section
- [ ] Instructions clearly state sdd_explore is OPTIONAL
- [ ] Instructions explain upsert behavior
- [ ] All existing tests pass (no pipeline changes)

---

### TASK-006: Update documentation (README + tool-reference)
**Component**: README.md, docs/tool-reference.md
**Covers**: Documentation
**Dependencies**: TASK-005 (need final tool count)
**Description**: Update documentation to reflect the new tool:
1. **README.md**: Update tool count (31 total, or whatever the new count is). Add `sdd_explore` to the tool overview table if one exists.
2. **docs/tool-reference.md**: Add entry for `sdd_explore` with full parameter description, example usage, and relationship to pipelines.

**Acceptance Criteria**:
- [ ] Tool count is accurate in README
- [ ] `sdd_explore` appears in tool reference with all parameters documented
- [ ] Documentation explains the hybrid nature (standalone + pipeline integration)
- [ ] Example usage shows a realistic exploration scenario

## Dependency Graph

```
TASK-001 ──┐
           ├──→ TASK-003 ──→ TASK-004
TASK-002 ──┘                    │
                                ▼
                           TASK-005 ──→ TASK-006
```

- TASK-001 and TASK-002 can run in parallel
- TASK-003 depends on both TASK-001 and TASK-002
- TASK-004 depends on TASK-003
- TASK-005 depends on TASK-002 (needs constructor) and should wait for TASK-004 (all tests pass)
- TASK-006 depends on TASK-005 (needs final tool count)
