# Tasks: Research-Backed Pipeline Rigor

## Estimated Effort

3-4 days for a single developer. ~15 tasks across 5 execution waves.

## Task Breakdown

### TASK-001: Add `StageContextCheck` constant and filename mapping to changes package

**Component**: `internal/changes/types.go`, `internal/changes/flows.go`
**Covers**: FR-015
**Dependencies**: None
**Description**: Add `StageContextCheck ChangeStage = "context-check"` constant to `types.go`. Add `StageContextCheck: "context-check.md"` to `stageFilenames` map in `flows.go`. Do NOT update FlowRegistry yet (TASK-002 handles that).
**Acceptance Criteria**:
- [ ] New constant exists and is exported
- [ ] Filename mapping exists
- [ ] Existing tests pass (no flow changes yet)

---

### TASK-002: Update FlowRegistry with context-check in all 12 flows

**Component**: `internal/changes/flows.go`
**Covers**: FR-001, FR-015
**Dependencies**: TASK-001
**Description**: Insert `StageContextCheck` at index 1 (after initial stage, before next stage) in all 12 FlowRegistry entries. All small flows go from 3→4 stages, medium from 4→5, large from 5-6→6-7.
**Acceptance Criteria**:
- [ ] All 12 flow variants include `StageContextCheck` at position 1
- [ ] `StageFlow()` returns correct flows for all type×size combinations
- [ ] Flow tests updated to reflect new stage counts

---

### TASK-003: Update flow tests for new context-check stage

**Component**: `internal/changes/flows_test.go`
**Covers**: FR-017
**Dependencies**: TASK-002
**Description**: Update all existing flow tests to expect the new stage. Add test that verifies `StageContextCheck` is ALWAYS at index 1 in every flow. Add test for the new filename mapping.
**Acceptance Criteria**:
- [ ] All existing flow tests pass with updated expectations
- [ ] New test: `TestContextCheckAlwaysSecond` verifies position across all flows
- [ ] New test: `TestStageFilenameContextCheck` verifies "context-check.md" mapping
- [ ] `go test -race ./internal/changes/...` passes

---

### TASK-004: Add `StageBusinessRules` to greenfield pipeline config

**Component**: `internal/config/config.go`
**Covers**: FR-007, FR-016
**Dependencies**: None
**Description**: Add `StageBusinessRules Stage = "business-rules"` constant. Update `StageOrder` to insert it BEFORE `StageClarify`: `[init, propose, specify, business-rules, clarify, design, tasks, validate]`. Update `Stages` map with metadata (Order: 3, bump subsequent orders). Add `StageBusinessRules: "business-rules.md"` to `stageFilenames`.
**Acceptance Criteria**:
- [ ] New constant exists
- [ ] `StageOrder` has 8 elements in correct order
- [ ] `Stages` map has metadata for business-rules
- [ ] `stageFilenames` maps to "business-rules.md"
- [ ] `NewProjectConfig` creates status entry for the new stage

---

### TASK-005: Update greenfield pipeline tests

**Component**: `internal/config/config_test.go`, `internal/pipeline/state_test.go`
**Covers**: FR-017
**Dependencies**: TASK-004
**Description**: Update tests to expect 8 stages. Verify pipeline transitions work through the new business-rules stage. Verify `NewProjectConfig` includes the new stage in status map. Verify `CanAdvance` works correctly when at `StageBusinessRules`.
**Acceptance Criteria**:
- [ ] `TestStageOrder` expects 8 stages
- [ ] `TestNewProjectConfig` expects 8 stage statuses
- [ ] Pipeline state transition tests work through business-rules
- [ ] `go test -race ./internal/config/... ./internal/pipeline/...` passes

---

### TASK-006: Create `BusinessRulesData` template data structure

**Component**: `internal/templates/templates.go`
**Covers**: FR-008, FR-009, FR-010
**Dependencies**: None
**Description**: Add `BusinessRulesData` struct with fields: Definitions, Facts, Constraints, Derivations, Glossary (all strings). Add guided and expert template strings for business-rules.md rendering. Register templates in `NewRenderer()`.
**Acceptance Criteria**:
- [ ] `BusinessRulesData` struct defined with all 5 fields
- [ ] Guided template includes explanatory blockquotes with BRM/DDD citations
- [ ] Expert template is clean without explanations
- [ ] Templates render correctly with test data
- [ ] Conditional sections: Derivations and Glossary render only when non-empty

---

### TASK-007: Create `sdd_create_business_rules` tool

**Component**: `internal/tools/business_rules.go`
**Covers**: FR-011, FR-021
**Dependencies**: TASK-004, TASK-006
**Description**: Create new tool following existing pipeline tool pattern. Dependencies: `config.Store`, `templates.Renderer`. Parameters: definitions (required), facts (required), constraints (required), derivations (optional), glossary (optional). Handle: load config → require stage → validate params → render template → write file → advance pipeline → notify bridge.
**Acceptance Criteria**:
- [ ] Tool registered as `sdd_create_business_rules`
- [ ] Requires `StageBusinessRules` stage
- [ ] Validates required parameters
- [ ] Renders template (guided or expert based on config mode)
- [ ] Writes `sdd/business-rules.md`
- [ ] Advances pipeline to `StageClarify`
- [ ] Supports `SetBridge` for memory persistence

---

### TASK-008: Create `sdd_context_check` tool

**Component**: `internal/tools/context_check.go`
**Covers**: FR-002, FR-003, FR-004, FR-005, FR-006
**Dependencies**: TASK-001
**Description**: Create the scanner tool. Dependencies: `changes.Store` (required), `*memory.Store` (nullable). Parameters: `change_description` (required), `project_name` (optional).

Handle logic:
1. `findProjectRoot()`
2. Scan SDD artifacts: business-rules.md, requirements.md, proposal.md, design.md
3. List completed changes → keyword-match against change_description → max 10 results
4. If memStore != nil → search explore observations (type=explore, project filter, limit 5)
5. If no SDD artifacts found → scan convention files (CLAUDE.md, AGENTS.md, README.md, .cursor/rules/, CONTRIBUTING.md) — first 200 lines each
6. Build and return structured report

**Acceptance Criteria**:
- [ ] Tool registered as `sdd_context_check`
- [ ] Works with nil memory store (degrades gracefully)
- [ ] Scans filesystem artifacts correctly
- [ ] Keyword-matches completed changes (max 10)
- [ ] Searches explore observations when memory available
- [ ] Falls back to convention files when no SDD artifacts
- [ ] Returns structured markdown report
- [ ] Does NOT perform ambiguity analysis (scanner only)

---

### TASK-009: Write tests for context-check tool

**Component**: `internal/tools/context_check_test.go`
**Covers**: FR-017
**Dependencies**: TASK-008
**Description**: Test the context-check tool with various scenarios:
- Project with full SDD artifacts
- Project with no SDD artifacts (convention file fallback)
- Project with completed changes (keyword matching)
- With and without memory store
- Empty change description validation
**Acceptance Criteria**:
- [ ] Test: full SDD project with artifacts
- [ ] Test: empty project with CLAUDE.md fallback
- [ ] Test: keyword matching against completed changes
- [ ] Test: nil memory store degrades gracefully
- [ ] Test: empty change_description returns error
- [ ] `go test -race ./internal/tools/...` passes

---

### TASK-010: Write tests for business-rules tool

**Component**: `internal/tools/business_rules_test.go`
**Covers**: FR-017
**Dependencies**: TASK-007
**Description**: Test the business-rules tool following existing pipeline tool test patterns. Use temp directories for filesystem isolation.
**Acceptance Criteria**:
- [ ] Test: successful creation with all required params
- [ ] Test: missing required params returns error
- [ ] Test: wrong pipeline stage returns error
- [ ] Test: optional params (derivations, glossary) work when empty
- [ ] Test: file written to correct location
- [ ] Test: pipeline advances to clarify after completion
- [ ] `go test -race ./internal/tools/...` passes

---

### TASK-011: Register new tools in server.go composition root

**Component**: `internal/server/server.go`
**Covers**: FR-011, FR-015
**Dependencies**: TASK-007, TASK-008
**Description**: Wire both new tools in `server.go`:
- `contextCheckTool` — registered ONCE with nullable memStore (before/after memory init block)
- `businessRulesTool` — registered in SDD tools section, wired with bridge
Update tool count in any documentation/comments.
**Acceptance Criteria**:
- [ ] `contextCheckTool` registered with `changeStore` and `memStore`
- [ ] `businessRulesTool` registered with `store` and `renderer`
- [ ] `businessRulesTool` bridge wired when memory available
- [ ] Server starts successfully
- [ ] Tool count updated in docs/comments

---

### TASK-012: Rewrite `serverInstructions()` — change pipeline section

**Component**: `internal/server/server.go`
**Covers**: FR-012, FR-013, FR-014, FR-020
**Dependencies**: TASK-008, TASK-011
**Description**: Rewrite the "ADAPTIVE CHANGE PIPELINE" section of server instructions:
- Add context-check stage instructions with IEEE 29148 Requirements Smells heuristics
- Add impact classification guidance (SemVer model)
- Update all flow listings to include context-check
- Add examples of good vs bad questions (FR-020)
- Cite research sources for every framework
**Acceptance Criteria**:
- [ ] Context-check stage instructions present with IEEE 29148 citation
- [ ] Requirements Smells heuristics listed (subjective language, ambiguous adverbs, etc.)
- [ ] Impact classification guidance with SemVer model
- [ ] All 12 flow listings updated to show context-check
- [ ] Examples of good vs bad questions included
- [ ] Research sources cited inline

---

### TASK-013: Rewrite `serverInstructions()` — greenfield pipeline section

**Component**: `internal/server/server.go`
**Covers**: FR-012, FR-013, FR-020
**Dependencies**: TASK-007, TASK-011
**Description**: Rewrite the "Pipeline" and "Stage-by-Stage Workflow" sections:
- Update pipeline listing to 8 stages with business-rules
- Add business-rules stage instructions with BRG/BRM/DDD citations
- Add EARS syntax reference to clarify stage
- Update specify stage with IEEE 29148 quality attributes
- Add research citations to all existing stages
- Keep total instructions within 462-line budget (NFR-004)
**Acceptance Criteria**:
- [ ] Pipeline listing shows 8 stages
- [ ] Business-rules stage workflow documented
- [ ] BRG taxonomy cited for business rules
- [ ] EARS patterns cited for clarify stage
- [ ] IEEE 29148 cited for specify stage
- [ ] Total instructions ≤ 462 lines
- [ ] All existing functionality still documented

---

### TASK-014: End-to-end integration test

**Component**: `internal/tools/change_integration_test.go` (extend existing)
**Covers**: FR-017
**Dependencies**: TASK-011
**Description**: Add integration test that creates a change, runs through all stages including context-check, and completes the change. Verify the state machine transitions correctly with the new stage.
**Acceptance Criteria**:
- [ ] Test creates a change (feature/small)
- [ ] Test advances through describe → context-check → tasks → verify
- [ ] Test verifies all artifacts written
- [ ] Test verifies change completes successfully
- [ ] `go test -race ./internal/tools/...` passes

---

### TASK-015: Final verification — full test suite + lint

**Component**: Project-wide
**Covers**: FR-017, NFR-001, NFR-003, NFR-006
**Dependencies**: All previous tasks
**Description**: Run full test suite, linter, and build. Verify:
- All tests pass with race detector
- No lint warnings
- Binary builds successfully with CGO_ENABLED=0
- No new dependencies in go.mod
- Backward compatibility — no existing tool signatures changed
**Acceptance Criteria**:
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make build` succeeds
- [ ] `go.mod` has no new dependencies
- [ ] All existing MCP tool signatures unchanged (NFR-006)

---

## Dependency Graph

```
TASK-001 ──→ TASK-002 ──→ TASK-003
         └──→ TASK-008 ──→ TASK-009
TASK-004 ──→ TASK-005
         └──→ TASK-007 ──→ TASK-010
TASK-006 ──→ TASK-007
TASK-007 ─┐
TASK-008 ─┤──→ TASK-011 ──→ TASK-012 ──→ TASK-014
          │              └──→ TASK-013 ──↗
          └──→ TASK-014 ──→ TASK-015
```

## Execution Waves

**Wave 1** (parallel — no dependencies):
- TASK-001: Add StageContextCheck constant
- TASK-004: Add StageBusinessRules to config
- TASK-006: Create BusinessRulesData template

**Wave 2** (parallel — depends on Wave 1):
- TASK-002: Update FlowRegistry (depends: TASK-001)
- TASK-005: Update greenfield pipeline tests (depends: TASK-004)
- TASK-007: Create business-rules tool (depends: TASK-004, TASK-006)
- TASK-008: Create context-check tool (depends: TASK-001)

**Wave 3** (parallel — depends on Wave 2):
- TASK-003: Update flow tests (depends: TASK-002)
- TASK-009: Context-check tool tests (depends: TASK-008)
- TASK-010: Business-rules tool tests (depends: TASK-007)
- TASK-011: Register tools in server.go (depends: TASK-007, TASK-008)

**Wave 4** (parallel — depends on Wave 3):
- TASK-012: Rewrite server instructions — change pipeline (depends: TASK-011)
- TASK-013: Rewrite server instructions — greenfield pipeline (depends: TASK-011)

**Wave 5** (sequential — depends on all):
- TASK-014: End-to-end integration test (depends: TASK-011)
- TASK-015: Final verification (depends: ALL)

## Global Acceptance Criteria

- All code must pass `golangci-lint` with zero warnings
- Test coverage for new code must be ≥ 80%
- All existing tests must continue to pass
- No new external dependencies in `go.mod`
- CGO_ENABLED=0 build must succeed
- All MCP tool signatures remain backward-compatible
