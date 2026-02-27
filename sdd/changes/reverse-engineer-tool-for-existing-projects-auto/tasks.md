# Tasks: sdd_reverse_engineer + sdd_change auto-trigger

## Estimated Effort

**4-5 days for a single developer** — 8 tasks across 4 waves.

## Task Breakdown

### TASK-001: Extract shared artifact rendering functions
**Component**: SharedArtifactWriters
**Covers**: FR-022
**Dependencies**: None
**Description**: Create `internal/tools/artifacts.go` with three exported functions that encapsulate the rendering+writing logic currently embedded in `specify.go`, `business_rules.go`, and `design.go`:
- `RenderAndWriteRequirements(projectRoot string, renderer templates.Renderer, data templates.RequirementsData) (string, error)`
- `RenderAndWriteBusinessRules(projectRoot string, renderer templates.Renderer, data templates.BusinessRulesData) (string, error)`
- `RenderAndWriteDesign(projectRoot string, renderer templates.Renderer, data templates.DesignData) (string, error)`

Each function: render via template → write via `writeStageFile` → return rendered content.

**Acceptance Criteria**:
- [ ] `artifacts.go` created with 3 exported functions
- [ ] Each function renders via `templates.Renderer` and writes to the correct `sdd/` path
- [ ] Functions use `config.StagePath()` for path resolution
- [ ] Unit tests in `artifacts_test.go` verify rendering + file creation for all 3 artifact types
- [ ] Functions handle the `> ⚡ Auto-generated` header as an optional parameter (boolean flag)

---

### TASK-002: Refactor existing pipeline tools to use shared functions
**Component**: PipelineTools
**Covers**: FR-022
**Dependencies**: TASK-001
**Description**: Modify `specify.go`, `business_rules.go`, and `design.go` to delegate their rendering+writing to the shared functions from `artifacts.go`. Each tool keeps its stage validation, pipeline advancement, and bridge notification — only the rendering+writing moves to the shared function.

Before:
```go
content, err := t.renderer.Render(templates.Requirements, data)
writeStageFile(reqPath, content)
```

After:
```go
content, err := RenderAndWriteRequirements(projectRoot, t.renderer, data)
```

**Acceptance Criteria**:
- [ ] `specify.go` calls `RenderAndWriteRequirements` instead of inline rendering
- [ ] `business_rules.go` calls `RenderAndWriteBusinessRules` instead of inline rendering
- [ ] `design.go` calls `RenderAndWriteDesign` instead of inline rendering
- [ ] All existing tests pass without modification (behavior is unchanged)
- [ ] `make test` passes with race detector

---

### TASK-003: Implement scanner sub-scanners
**Component**: ReverseEngineerTool
**Covers**: FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-014, FR-015, FR-017, FR-018
**Dependencies**: None
**Description**: Create `internal/tools/reverse_engineer.go` with the scanner infrastructure and all sub-scanners. Each sub-scanner is a function (not interface — YAGNI) that takes `root string` and `detailLevel` and returns a `scanSection` struct:

```go
type scanSection struct {
    title       string
    content     string
    filesRead   int
    filesSkipped int
}
```

Sub-scanners to implement:
1. `scanManifests` — package.json, go.mod, etc. (FR-003)
2. `scanStructure` — directory tree with depth limit and ignore patterns (FR-002)
3. `scanConfigs` — tsconfig, eslint, Docker, CI (FR-005)
4. `scanEntryPoints` — main.go, index.ts, etc. (FR-006)
5. `scanConventions` — CLAUDE.md, AGENTS.md, linting configs (FR-004)
6. `scanSchemas` — migrations, ORM models, prisma schemas (FR-008)
7. `scanAPIDefs` — OpenAPI specs, route files (FR-009)
8. `scanADRs` — ADR directories (FR-007)
9. `scanTests` — test directories, framework detection (FR-017)

Include:
- `ignoreDirs` map for directory exclusion (node_modules, .git, etc.)
- File size guard: skip files > 100KB
- Monorepo detection via workspace patterns (FR-018)
- Graceful degradation: all scanner errors are caught and reported, never abort

**Acceptance Criteria**:
- [ ] All 9 sub-scanners implemented as functions
- [ ] Each scanner handles missing files/dirs gracefully (skip, don't error)
- [ ] `ignoreDirs` excludes common noise directories
- [ ] Files > 100KB are skipped with reason reported
- [ ] Monorepo workspaces detected from package.json/pnpm-workspace.yaml
- [ ] Unit tests for each scanner using temp directory fixtures
- [ ] Tests cover: Go project, Node.js project, Python project, empty directory

---

### TASK-004: Implement ReverseEngineerTool MCP handler
**Component**: ReverseEngineerTool
**Covers**: FR-001, FR-010, FR-011, FR-012, FR-016, FR-019, NFR-001, NFR-002, NFR-003
**Dependencies**: TASK-003
**Description**: Implement the `ReverseEngineerTool` struct with `Definition()` and `Handle()` methods. The handler orchestrates all sub-scanners, assembles the structured markdown report, applies token budgeting, and returns the result.

Parameters: `detail_level`, `max_tokens`, `scan_path`, `max_depth`.

The report assembles all `scanSection` results in order, adds metadata header (project root, file counts, scan duration, detected ecosystem), and appends the AI instruction block telling the AI what artifacts to generate.

**Acceptance Criteria**:
- [ ] Tool registered with `sdd_reverse_engineer` name
- [ ] `detail_level` parameter controls verbosity (summary/standard/full)
- [ ] `max_tokens` truncates with section priority (overview first, ADRs last)
- [ ] `scan_path` allows subdirectory scanning
- [ ] `max_depth` controls directory tree depth (default: 3)
- [ ] Metadata header includes: root, files scanned/skipped, duration, primary ecosystem
- [ ] AI instruction block at top of report per FR-016
- [ ] Report stays under 8K tokens at `standard` level for typical projects (NFR-002)
- [ ] Scan completes in <5 seconds for 10K file projects (NFR-001)
- [ ] Integration test: scan Hoofy's own codebase, verify report structure

---

### TASK-005: Implement BootstrapTool
**Component**: BootstrapTool
**Covers**: FR-016, FR-022
**Dependencies**: TASK-001
**Description**: Create `internal/tools/bootstrap.go` with the `BootstrapTool` struct. This tool writes SDD artifacts for projects that bypassed the greenfield pipeline.

Parameters: combined from all 3 artifact tools (requirements_*, business_rules_*, design_*). At least one artifact group must have content.

Logic:
1. Check which of the 3 artifacts already exist in `sdd/`
2. For each missing artifact where content was provided:
   - Call shared rendering function with auto-generated header flag
   - Report what was written
3. For existing artifacts: skip and report "already exists"
4. Create `sdd/` directory if needed

**Acceptance Criteria**:
- [ ] Tool registered with `sdd_bootstrap` name
- [ ] Only writes missing artifacts (skips existing ones per FR-016 updated)
- [ ] Prepends `> ⚡ Auto-generated by sdd_reverse_engineer` header
- [ ] Creates `sdd/` directory if it doesn't exist
- [ ] Returns clear summary: written vs skipped
- [ ] Requires at least one artifact group to have content
- [ ] Works without `sdd.json` existing
- [ ] Unit tests: write all 3, write 1 of 3 (2 exist), write 0 (all exist)

---

### TASK-006: Modify sdd_change for artifact existence check
**Component**: ChangeTool
**Covers**: FR-013 (updated)
**Dependencies**: None
**Description**: Add SDD artifact existence check to `change.go` between the active-change guard and the flow lookup.

Logic:
- Call `checkSDDArtifacts(projectRoot)` — stats 3 files, returns bool
- If no artifacts AND size is medium/large: return error with guidance to run `sdd_reverse_engineer`
- If no artifacts AND size is small: proceed but append warning to response

**Acceptance Criteria**:
- [ ] `checkSDDArtifacts` helper function implemented (3 file stats)
- [ ] Medium/large changes blocked with clear error message
- [ ] Small changes proceed with warning appended
- [ ] Warning text matches FR-013 updated spec
- [ ] Check adds <10ms latency (NFR-006)
- [ ] Unit tests: with artifacts (no change), without artifacts + small (warning), without artifacts + medium (block), without artifacts + large (block)

---

### TASK-007: Register new tools in composition root
**Component**: Server
**Covers**: NFR-007
**Dependencies**: TASK-004, TASK-005
**Description**: Register `ReverseEngineerTool` and `BootstrapTool` in `internal/server/server.go`. Wire the `templates.Renderer` dependency for `BootstrapTool`.

**Acceptance Criteria**:
- [ ] `ReverseEngineerTool` registered with no dependencies (pure scanner)
- [ ] `BootstrapTool` registered with `templates.Renderer` injected
- [ ] Server starts without errors
- [ ] `make build` succeeds
- [ ] `make test` passes

---

### TASK-008: Update server instructions for reverse engineer workflow
**Component**: Server
**Covers**: FR-016
**Dependencies**: TASK-007
**Description**: Add instructions to `serverInstructions()` in `server.go` explaining the reverse engineer workflow:

1. When to use `sdd_reverse_engineer` (existing projects without SDD artifacts)
2. How to analyze the scan report (what to look for per artifact type)
3. How to call `sdd_bootstrap` with the generated content
4. That auto-generated artifacts should be reviewed by the user
5. The relationship between reverse engineer → bootstrap → sdd_change flow

**Acceptance Criteria**:
- [ ] Server instructions include reverse engineer workflow section
- [ ] Instructions explain scan → analyze → bootstrap → change flow
- [ ] Instructions mention the auto-generated header
- [ ] Instructions guide the AI to only generate missing artifacts

---

## Dependency Graph

```
TASK-001 (shared writers) ──→ TASK-002 (refactor pipeline tools)
TASK-001 (shared writers) ──→ TASK-005 (bootstrap tool)
TASK-003 (sub-scanners) ───→ TASK-004 (scanner handler)
TASK-004 + TASK-005 ───────→ TASK-007 (register in server.go)
TASK-007 ──────────────────→ TASK-008 (server instructions)
TASK-006 (change guard) ───→ (independent)
```

## Wave Assignments

**Wave 1** (parallel — no dependencies):
- TASK-001: Extract shared artifact rendering functions
- TASK-003: Implement scanner sub-scanners
- TASK-006: Modify sdd_change for artifact existence check

**Wave 2** (parallel — depends on Wave 1):
- TASK-002: Refactor existing pipeline tools to use shared functions (needs TASK-001)
- TASK-004: Implement ReverseEngineerTool MCP handler (needs TASK-003)
- TASK-005: Implement BootstrapTool (needs TASK-001)

**Wave 3** (depends on Wave 2):
- TASK-007: Register new tools in composition root (needs TASK-004, TASK-005)

**Wave 4** (depends on Wave 3):
- TASK-008: Update server instructions (needs TASK-007)

## Global Acceptance Criteria

- All code compiles with `CGO_ENABLED=0`
- `make test` passes with race detector and no failures
- `make lint` passes with no new warnings
- All new tools follow SRP: one file per tool
- All new tools have unit tests with >80% coverage
- No existing tests broken by refactoring
