# Implementation Tasks: Wave Metadata

## Task Summary

**Total Tasks:** 5
**Estimated Effort:** 2-3 hours for a single developer

---

## Tasks

### TASK-001: Add `WaveAssignments` field to `TasksData` struct
**Component**: templates
**Covers**: FR-002
**Dependencies**: None
**Wave**: 1

**Description**: Add `WaveAssignments string` field to the `TasksData` struct in `internal/templates/templates.go`.

**Acceptance Criteria**:
- [ ] `TasksData` has `WaveAssignments string` field
- [ ] Existing template rendering still works (field is zero-value empty string by default)

---

### TASK-002: Update `tasks.md.tmpl` template with conditional wave section
**Component**: templates
**Covers**: FR-003, FR-004
**Dependencies**: TASK-001
**Wave**: 2

**Description**: Add a conditional `{{ if .WaveAssignments }}` block to `tasks.md.tmpl` that renders an "Execution Waves" section between the Dependency Graph and Acceptance Criteria sections. When `WaveAssignments` is empty, nothing renders. Include explanatory text that tasks within the same wave can execute in parallel.

**Acceptance Criteria**:
- [ ] Template renders wave section when `WaveAssignments` is non-empty
- [ ] Template renders nothing when `WaveAssignments` is empty (no empty header)
- [ ] Wave section includes explanatory note about parallelization
- [ ] Wave section appears between Dependency Graph and Acceptance Criteria
- [ ] Existing template tests pass unchanged

---

### TASK-003: Add `wave_assignments` parameter to `sdd_create_tasks` tool
**Component**: tools/tasks.go
**Covers**: FR-001
**Dependencies**: TASK-001
**Wave**: 2

**Description**: Add an optional `wave_assignments` string parameter to the `sdd_create_tasks` MCP tool definition. Read it in `Handle()`, pass it through to `TasksData`. No validation needed (it's optional freeform content from the AI).

**Acceptance Criteria**:
- [ ] Tool definition includes `wave_assignments` parameter (optional, not required)
- [ ] Parameter description explains wave concept and format
- [ ] Value passed to `TasksData.WaveAssignments`
- [ ] Tool works without the parameter (backwards compatible)

---

### TASK-004: Update `serverInstructions()` with wave assignment guidance
**Component**: server/server.go
**Covers**: FR-005, FR-006, FR-007
**Dependencies**: TASK-003
**Wave**: 3

**Description**: Add guidance to the `serverInstructions()` function that instructs the AI to:
1. Analyze task dependencies and assign wave numbers when calling `sdd_create_tasks`
2. Explain the wave algorithm (no deps = Wave 1, depends only on Wave 1 = Wave 2, etc.)
3. Include wave sections in change pipeline tasks stage content (freeform markdown guidance)

**Acceptance Criteria**:
- [ ] Instructions explain what waves are
- [ ] Instructions explain wave assignment algorithm
- [ ] Instructions mention wave_assignments parameter for sdd_create_tasks
- [ ] Instructions mention wave sections for change pipeline tasks

---

### TASK-005: Add tests for wave metadata
**Component**: templates, tools
**Covers**: NFR-001, NFR-002
**Dependencies**: TASK-002, TASK-003
**Wave**: 3

**Description**: Add tests covering:
1. Template test: wave section renders when WaveAssignments is non-empty
2. Template test: no wave section renders when WaveAssignments is empty (backwards compat)
3. Tool test: sdd_create_tasks works without wave_assignments parameter
4. Tool test: sdd_create_tasks includes wave content when parameter provided

**Acceptance Criteria**:
- [ ] All new tests pass with `-race`
- [ ] All existing tests pass unchanged
- [ ] Template rendering tested for both with/without wave data
- [ ] Tool handler tested for both with/without wave parameter

## Dependency Graph

```
TASK-001 (struct) ──┬──→ TASK-002 (template) ──┬──→ TASK-005 (tests)
                    └──→ TASK-003 (tool param)  ┘
                              │
                              └──→ TASK-004 (instructions)
```

Wave 1: TASK-001 (independent)
Wave 2: TASK-002, TASK-003 (depend on TASK-001, parallel with each other)
Wave 3: TASK-004, TASK-005 (depend on Wave 2)

## Acceptance Criteria

- All code passes `go test -race ./...`
- Zero regressions in existing tests
- Backwards compatibility verified: tool calls without `wave_assignments` produce identical output