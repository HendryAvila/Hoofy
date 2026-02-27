# Verify: sdd_reverse_engineer + sdd_change auto-trigger

## Requirements Traceability

### Functional Requirements Coverage

| Requirement | Task(s) | Status |
|-------------|---------|--------|
| FR-001: detail_level parameter | TASK-004 | ✅ Covered |
| FR-002: Directory structure scanning | TASK-003 (scanStructure) | ✅ Covered |
| FR-003: Package manifest detection | TASK-003 (scanManifests) | ✅ Covered |
| FR-004: Convention files | TASK-003 (scanConventions) | ✅ Covered |
| FR-005: Configuration files | TASK-003 (scanConfigs) | ✅ Covered |
| FR-006: Entry points | TASK-003 (scanEntryPoints) | ✅ Covered |
| FR-007: ADR files | TASK-003 (scanADRs) | ✅ Covered |
| FR-008: Database schemas | TASK-003 (scanSchemas) | ✅ Covered |
| FR-009: API definitions | TASK-003 (scanAPIDefs) | ✅ Covered |
| FR-010: Structured markdown report | TASK-004 | ✅ Covered |
| FR-011: max_tokens parameter | TASK-004 | ✅ Covered |
| FR-012: Metadata header | TASK-004 | ✅ Covered |
| FR-013: sdd_change auto-trigger | TASK-006 | ✅ Covered |
| FR-014: Works without sdd_init | TASK-003, TASK-004 | ✅ Covered |
| FR-015: Graceful degradation | TASK-003 | ✅ Covered |
| FR-016: AI instructions + missing-only writes | TASK-004, TASK-005 | ✅ Covered |
| FR-017: Test directory detection | TASK-003 (scanTests) | ✅ Covered |
| FR-018: Monorepo detection | TASK-003 | ✅ Covered |
| FR-019: scan_path parameter | TASK-004 | ✅ Covered |
| FR-020: Exported symbols (Could Have) | — | ⏭️ Deferred to v2 |
| FR-021: Environment variables (Could Have) | — | ⏭️ Deferred to v2 |
| FR-022: Shared rendering functions | TASK-001, TASK-002 | ✅ Covered |

**Coverage: 20/20 Must+Should requirements covered. 2 Could Have deferred.**

### Non-Functional Requirements Coverage

| Requirement | Task(s) | Verification |
|-------------|---------|-------------|
| NFR-001: <5s scan for 10K files | TASK-003, TASK-004 | Integration test scans Hoofy codebase |
| NFR-002: <8K tokens at standard | TASK-004 | Token estimation in test |
| NFR-003: <50MB memory | TASK-003 | File size guard (skip >100KB) |
| NFR-004: CGO_ENABLED=0 | All tasks | `make build` verification |
| NFR-005: Go/Node/Python/Rust support | TASK-003 | Scanner fixture tests per ecosystem |
| NFR-006: <10ms artifact check | TASK-006 | 3 file stats — trivially fast |
| NFR-007: SRP one file per tool | TASK-003, TASK-004, TASK-005 | Code review |

**All 7 NFRs covered.**

## Component Coverage

| Component | Tasks |
|-----------|-------|
| ReverseEngineerTool | TASK-003, TASK-004 |
| SharedArtifactWriters | TASK-001 |
| BootstrapTool | TASK-005 |
| PipelineTools (refactor) | TASK-002 |
| ChangeTool (modification) | TASK-006 |
| Server (composition root) | TASK-007 |
| Server (instructions) | TASK-008 |

**All components have at least one task.**

## Consistency Check

### Cross-Artifact Alignment

1. ✅ **Proposal → Spec**: All proposal items captured as formal requirements. "What the scanner reads" maps to FR-002 through FR-009. "What it doesn't do" maps to FR-W01 through FR-W05.

2. ✅ **Spec → Design**: Every FR maps to a design component. The shared rendering functions (FR-022) introduced in clarify are fully designed in ADR-001 and component 2.

3. ✅ **Design → Tasks**: Every component in the design has tasks assigned. The dependency graph correctly reflects the extraction-before-refactoring order.

4. ✅ **Clarify answers → Spec updates**: Auto-trigger (Option B with warnings) reflected in updated FR-013 and TASK-006. Shared functions (A2) reflected in FR-022 and TASK-001/002. Missing-only writes reflected in updated FR-016 and TASK-005.

### Issues Found

1. **Minor: Design shows both functions and interface for scanners**
   The design initially described a `scanner interface` but then says "functions not interface — YAGNI". The tasks (TASK-003) correctly use functions. Consistent with the final decision.
   **Impact**: None — design narrative shows the thinking process, tasks are correct.

2. **Minor: BootstrapTool parameter count**
   The bootstrap tool has 10 parameters which is a lot for one MCP tool. This is acceptable because:
   - The AI sends multiple artifacts in one call (efficient)
   - At least one group must be provided (not all are required)
   - The alternative (3 separate calls) adds roundtrip overhead
   **Impact**: None — acceptable tradeoff.

3. **Observation: No proposal.md in bootstrap**
   The proposal is explicitly excluded (makes no sense for existing projects — you can't "propose" what already exists). This is consistent with the original proposal's scope.

## Verdict

**✅ PASS** — All requirements traced to tasks, all components covered, no consistency issues. The feature is ready for implementation.

## Implementation Order

1. **Wave 1** (start immediately, parallel):
   - TASK-001: `artifacts.go` — shared rendering functions
   - TASK-003: `reverse_engineer.go` — sub-scanners
   - TASK-006: `change.go` — artifact existence check

2. **Wave 2** (after Wave 1, parallel):
   - TASK-002: Refactor pipeline tools to use shared functions
   - TASK-004: Scanner MCP handler (assembles report)
   - TASK-005: `bootstrap.go` — bootstrap tool

3. **Wave 3**: TASK-007 — register in server.go
4. **Wave 4**: TASK-008 — server instructions
