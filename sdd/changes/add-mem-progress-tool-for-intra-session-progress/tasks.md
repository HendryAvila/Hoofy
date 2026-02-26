# Tasks: mem_progress Tool

## Task Breakdown

### TASK-001: Create ProgressTool in memtools package
**File**: `internal/memtools/progress.go`
**Description**: Create the `ProgressTool` struct with `Definition()` and `Handle()` methods.

**Definition**:
- Tool name: `mem_progress`
- Description: explains dual read/write behavior
- Parameters: `project` (required), `content` (optional — JSON progress doc), `session_id` (optional)

**Handle logic**:
- If `content` is empty → READ mode: search for observation with `topic_key = "progress/<project>"` and return it
- If `content` is provided → WRITE mode:
  1. Validate `content` is valid JSON via `json.Valid([]byte(content))`
  2. Save as observation with `type = "progress"`, `topic_key = "progress/<project>"`, `scope = "project"`
  3. Title auto-generated: `"Progress: <project>"`
  4. Return confirmation with observation ID

**Read mode implementation**:
- Use `store.SearchObservations(query, project, scope, limit)` with query matching the topic key
- OR: use `store.GetLatestByTopicKey(topicKey, project, scope)` if available
- Needs investigation: check if Store has a method to get by topic_key directly

**Acceptance Criteria**:
- [ ] Read mode returns current progress or "no progress found" message
- [ ] Write mode validates JSON and saves with correct topic_key
- [ ] Invalid JSON returns clear error message
- [ ] Missing `project` returns error

### TASK-002: Register ProgressTool in server.go
**File**: `internal/server/server.go`
**Description**: Register the new tool in `registerMemoryTools()` function.
- Add to "Save & capture" or create a "Progress tracking" section
- Update tool count comment if it references a number

**Acceptance Criteria**:
- [ ] Tool appears in MCP tool listing
- [ ] Registered only when memory subsystem is available

### TASK-003: Add tests for ProgressTool
**File**: `internal/memtools/memtools_test.go`
**Description**: Add tests covering all behaviors.

**Test cases**:
1. `TestProgressTool_Definition` — verify tool name, required params
2. `TestProgressTool_ReadEmpty` — read when no progress exists → "no progress found"
3. `TestProgressTool_WriteAndRead` — write valid JSON, then read it back
4. `TestProgressTool_InvalidJSON` — write invalid JSON → error
5. `TestProgressTool_MissingProject` — call without project → error
6. `TestProgressTool_Upsert` — write twice, verify second overwrites first (topic_key upsert)

**Acceptance Criteria**:
- [ ] All 6 test cases pass
- [ ] Tests use isolated temp DB (existing pattern in memtools_test.go)

### TASK-004: Update serverInstructions with mem_progress guidance
**File**: `internal/server/server.go`
**Description**: Add guidance in `serverInstructions()` for when and how the AI should use `mem_progress`.

**Content**:
- At session start: check for existing progress via `mem_progress(project=X)`
- After completing significant work: update progress
- The JSON structure should include: `goal`, `completed`, `next_steps`, `blockers`
- Progress is for the AI agent, not the user

**Acceptance Criteria**:
- [ ] Instructions clearly explain read vs write mode
- [ ] JSON structure example included

### TASK-005: Update docs/research-foundations.md
**File**: `docs/research-foundations.md`
**Description**: Add F2 mapping to the "Effective Harnesses" section.

**New row**:
| "Each session: read progress..." | `mem_progress` tool persists structured JSON progress docs. Auto-read at session start, upserted during work. One active progress per project via topic_key |

**Acceptance Criteria**:
- [ ] Research source cited correctly
- [ ] Implementation description matches actual tool behavior

## Dependency Graph
```
TASK-001 → TASK-002 → TASK-003
TASK-001 → TASK-004
TASK-001 → TASK-005
```

## Wave Assignments
**Wave 1**: TASK-001 (tool implementation)
**Wave 2**: TASK-002, TASK-004, TASK-005 (parallel — registration, instructions, docs)
**Wave 3**: TASK-003 (tests — needs tool registered and working)

## Estimated Effort
~1 hour for a single developer. Small, well-scoped change.
