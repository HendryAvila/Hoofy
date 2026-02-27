# Design: sdd_reverse_engineer + sdd_change auto-trigger

## Architecture Overview

The reverse engineer feature consists of 3 components operating at different architectural layers:

1. **`ReverseEngineerTool`** — New MCP tool (`internal/tools/reverse_engineer.go`). Pure scanner — reads project files, builds structured markdown report. No write operations. Depends on no store interfaces.

2. **Shared Artifact Writers** — New module (`internal/tools/artifacts.go`). Extracted rendering+writing logic from existing pipeline tools. Pure functions: `RenderAndWriteRequirements`, `RenderAndWriteBusinessRules`, `RenderAndWriteDesign`. Each takes content parameters + project root + `templates.Renderer`, renders via template, writes to `sdd/`. No pipeline state machine interaction.

3. **`ChangeTool` modification** — Adds SDD artifact existence check to `sdd_change`. Fast filesystem stat of 3 files before creating the change record. Blocks medium/large, warns small.

### Data Flow

```
User calls sdd_reverse_engineer
    → Scanner reads project files (manifests, configs, entry points, schemas, ADRs, conventions)
    → Returns structured markdown report to AI
    → AI analyzes report, generates content for missing artifacts
    → AI calls existing pipeline tools (sdd_create_business_rules, sdd_create_design, sdd_generate_requirements)
        → Pipeline tools call shared artifact writers (no stage guards needed for the writer functions)
        → Artifacts written to sdd/
    → User calls sdd_change → context-check finds real artifacts ✅
```

**Important**: The reverse engineer tool does NOT write artifacts itself. It produces the evidence report. The AI then calls the existing pipeline tools (which now delegate to shared writers). BUT — the existing pipeline tools still have stage guards. So for the reverse engineer flow specifically, the AI must use a **new lightweight tool** or the server instructions must guide the AI to call the shared writers through a bridge tool.

**Revised approach after re-analyzing**: The cleanest path is:
1. Extract rendering logic to shared functions in `artifacts.go`
2. Existing pipeline tools call shared functions (after their stage guards)
3. Add a new `sdd_write_artifact` tool that calls the same shared functions **without** stage guards — specifically for the reverse engineer flow
4. Server instructions tell the AI: "After scanning with `sdd_reverse_engineer`, use `sdd_write_artifact` to save the generated artifacts"

Wait — this is Option D from the clarify stage, which was rejected. Let me reconsider.

**Final approach (ADR-001 compliant)**: The shared functions are the core. The reverse engineer tool's response INSTRUCTS the AI to write files. But the AI doesn't have a direct "write file" MCP tool. So we need a bridge. The simplest bridge:

→ **Add a `source` parameter to `sdd_reverse_engineer`** itself. When called with `source=scan` (default), it scans. When called with `source=write` and artifact parameters, it writes using the shared functions. This makes it a dual-mode tool: scan mode and write mode.

**No. Too complex. Let me simplify.** 🐴

**ACTUAL final approach**: The reverse engineer tool scans AND writes in a single call. The AI calls `sdd_reverse_engineer` which returns the scan report. The AI analyzes it, then calls `sdd_reverse_engineer` AGAIN with `action=generate` plus the content parameters. The tool writes the artifacts using shared functions.

**No. Still overcomplicating.** Let me think about this like a horse architect would.

---

### Simplified Architecture (final)

The reverse engineer tool has ONE job: **scan and report**. Period.

For writing artifacts, we add ONE new tool: **`sdd_bootstrap`**. This tool:
- Takes the same parameters as `sdd_create_business_rules`, `sdd_create_design`, and `sdd_generate_requirements` combined
- Calls the shared rendering functions
- Writes ONLY the missing artifacts (checks which exist first)
- Prepends the `> ⚡ Auto-generated` header to each artifact
- Does NOT interact with any pipeline state machine
- Does NOT require `sdd.json`

This is clean because:
- `sdd_reverse_engineer` = scanner (read-only)
- `sdd_bootstrap` = writer (write-only, no pipeline state)
- Existing pipeline tools = unchanged (read+validate+write+advance)
- All three use the same shared rendering functions

```
                      ┌─────────────────────────┐
                      │   Shared Artifact        │
                      │   Writers (artifacts.go) │
                      │                          │
                      │ RenderRequirements()     │
                      │ RenderBusinessRules()    │
                      │ RenderDesign()           │
                      └────────┬────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
    ┌─────────▼──────┐ ┌──────▼──────┐ ┌───────▼───────┐
    │ Pipeline Tools  │ │ sdd_bootstrap│ │  (future)     │
    │ (stage-guarded) │ │ (no guards)  │ │               │
    │ specify.go      │ │ bootstrap.go │ │               │
    │ design.go       │ │              │ │               │
    │ business_rules  │ │              │ │               │
    └────────────────┘ └──────────────┘ └───────────────┘
```

---

## Component Breakdown

### Component 1: ReverseEngineerTool (`reverse_engineer.go`)

**Responsibility**: Scan project filesystem, collect evidence, return structured markdown report.

**Dependencies**: None (no store, no renderer). Pure filesystem I/O.

**Covers**: FR-001 through FR-012, FR-014, FR-015, FR-017, FR-018, FR-019

**Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `detail_level` | string | No | `summary`, `standard` (default), `full` |
| `max_tokens` | number | No | Token budget cap |
| `scan_path` | string | No | Subdirectory to scan (default: project root) |
| `max_depth` | number | No | Directory tree depth (default: 3) |

**Report Structure** (9 sections):
1. Metadata header (root, file counts, duration, ecosystem)
2. Project Overview (manifest summary)
3. Directory Structure (tree, depth-limited)
4. Tech Stack Evidence (configs, deps)
5. Architecture Evidence (entry points, imports)
6. Conventions & Style (CLAUDE.md, AGENTS.md, linting)
7. Data Model Evidence (schemas, migrations)
8. API Evidence (OpenAPI, routes)
9. Prior Decisions (ADRs)
10. Files Skipped (what and why)

**Scanner Implementation**: Structured as a pipeline of sub-scanners, each responsible for one evidence category:

```go
type scanResult struct {
    section  string   // section name
    content  string   // markdown content
    filesRead int     // files actually read
    filesSkipped int  // files detected but skipped
}

type scanner interface {
    Scan(root string, detailLevel memory.DetailLevel) scanResult
}
```

Sub-scanners (each a simple struct with a `Scan` method):
- `manifestScanner` — package.json, go.mod, etc.
- `structureScanner` — directory tree with depth limit
- `configScanner` — tsconfig, eslint, Docker, CI
- `entryPointScanner` — main.go, index.ts, etc.
- `conventionScanner` — reuses existing `scanConventionFiles` from context_check.go
- `schemaScanner` — migrations, ORM models
- `apiScanner` — OpenAPI specs, route files
- `adrScanner` — ADR directories
- `testScanner` — test directories and frameworks (FR-017)

Each scanner is called sequentially (filesystem I/O is already sequential). Results are assembled into the final report.

**Ignored directories** (hardcoded, skipped during tree walk):
```go
var ignoreDirs = map[string]bool{
    "node_modules": true, ".git": true, "__pycache__": true,
    "vendor": true, "dist": true, "build": true, "target": true,
    ".next": true, ".nuxt": true, "venv": true, ".venv": true,
    ".idea": true, ".vscode": true, "coverage": true,
    ".cache": true, ".tmp": true,
}
```

**File size guard**: Skip reading any file > 100KB (report it in "Files Skipped").

### Component 2: Shared Artifact Writers (`artifacts.go`)

**Responsibility**: Render and write SDD artifacts using templates. No pipeline state management.

**Dependencies**: `templates.Renderer`

**Covers**: FR-022 (from clarify stage)

**Functions**:

```go
// RenderAndWriteRequirements renders requirements.md and writes it to sdd/.
func RenderAndWriteRequirements(projectRoot string, renderer templates.Renderer, data templates.RequirementsData) (string, error)

// RenderAndWriteBusinessRules renders business-rules.md and writes it to sdd/.
func RenderAndWriteBusinessRules(projectRoot string, renderer templates.Renderer, data templates.BusinessRulesData) (string, error)

// RenderAndWriteDesign renders design.md and writes it to sdd/.
func RenderAndWriteDesign(projectRoot string, renderer templates.Renderer, data templates.DesignData) (string, error)
```

Each function:
1. Calls `renderer.Render(templateName, data)` to get markdown content
2. Calls `writeStageFile(path, content)` to write to disk
3. Returns the rendered content for the caller to include in its response

**Refactoring of existing tools**: `specify.go`, `business_rules.go`, and `design.go` will be refactored to call these shared functions after their stage validation and before their pipeline advancement. The rendering+writing logic is extracted, not duplicated.

### Component 3: BootstrapTool (`bootstrap.go`)

**Responsibility**: Write SDD artifacts for projects that bypassed the greenfield pipeline. No stage guards, no pipeline state.

**Dependencies**: `templates.Renderer`

**Covers**: FR-016, FR-022

**Parameters**: Combines parameters from all 3 artifact tools:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `requirements_must_have` | string | No* | Must-have requirements |
| `requirements_should_have` | string | No* | Should-have requirements |
| `requirements_non_functional` | string | No* | NFRs |
| `business_rules_definitions` | string | No* | Domain terms |
| `business_rules_facts` | string | No* | Domain facts |
| `business_rules_constraints` | string | No* | Business constraints |
| `design_architecture` | string | No* | Architecture overview |
| `design_tech_stack` | string | No* | Tech stack |
| `design_components` | string | No* | Component breakdown |
| `design_data_model` | string | No* | Data model |

*At least one artifact group must be provided. The tool checks which artifacts already exist and only writes the missing ones.

**Behavior**:
1. Check which of the 3 artifacts already exist in `sdd/`
2. For each missing artifact where the AI provided content:
   - Prepend `> ⚡ Auto-generated by sdd_reverse_engineer — review and refine as needed\n\n` to the rendered content
   - Call the shared rendering function
   - Write to `sdd/`
3. Return summary of what was written and what was skipped (already existed)
4. Create `sdd/` directory if it doesn't exist

### Component 4: ChangeTool modification (`change.go`)

**Responsibility**: Add SDD artifact existence check before creating a change.

**Covers**: FR-013 (updated in clarify)

**Logic** (inserted after active-change guard, before flow lookup):

```go
// Check for SDD artifacts (for context-aware changes).
hasArtifacts := checkSDDArtifacts(projectRoot)
if !hasArtifacts {
    if changeSize == changes.SizeMedium || changeSize == changes.SizeLarge {
        return mcp.NewToolResultError(
            "❌ No SDD artifacts found in this project.\n\n" +
            "Medium and large changes require project context for accurate " +
            "architecture-aware suggestions.\n\n" +
            "**Run `sdd_reverse_engineer` first** to scan your project and " +
            "generate baseline SDD artifacts (`business-rules.md`, `design.md`, " +
            "`requirements.md`). Then retry this change.",
        ), nil
    }
    // Small changes proceed but with warning appended to response.
    // Warning text added to the response builder below.
}
```

**Helper function**:
```go
// checkSDDArtifacts checks if any SDD artifacts exist in the project.
func checkSDDArtifacts(projectRoot string) bool {
    artifacts := []string{"business-rules.md", "requirements.md", "design.md"}
    sddDir := filepath.Join(projectRoot, "sdd")
    for _, name := range artifacts {
        path := filepath.Join(sddDir, name)
        if _, err := os.Stat(path); err == nil {
            return true // at least one exists
        }
    }
    return false
}
```

**Warning for small changes** (appended to existing response):
```
⚠️ **No SDD artifacts found.** Consider running `sdd_reverse_engineer` to scan
your project and generate baseline specs. The `context-check` stage will have
limited context information without them.
```

---

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/tools/reverse_engineer.go` | **NEW** | Scanner tool — 9 sub-scanners, structured report |
| `internal/tools/artifacts.go` | **NEW** | Shared rendering functions for 3 artifact types |
| `internal/tools/bootstrap.go` | **NEW** | Bootstrap tool — writes missing artifacts without pipeline |
| `internal/tools/change.go` | **MODIFY** | Add artifact existence check (block/warn) |
| `internal/tools/specify.go` | **MODIFY** | Extract rendering to shared function call |
| `internal/tools/business_rules.go` | **MODIFY** | Extract rendering to shared function call |
| `internal/tools/design.go` | **MODIFY** | Extract rendering to shared function call |
| `internal/server/server.go` | **MODIFY** | Register 2 new tools, wire renderer dependency |

---

## Design Decisions

### ADR-001: Shared rendering functions over pipeline bypass
(See `adrs/ADR-001.md` — captured during clarify stage)

### ADR-002: sdd_bootstrap as separate tool from sdd_reverse_engineer
**Context**: The reverse engineer could scan AND write artifacts in one call, or they could be separate tools.
**Decision**: Separate tools. `sdd_reverse_engineer` scans (read-only), `sdd_bootstrap` writes (write-only).
**Rationale**: SRP — scanner has zero write side effects, making it safe to call repeatedly. The AI needs to analyze the scan results before deciding what to write. Combining scan+write would bypass the AI's analytical step, which is the entire point of "tools are dumb, AI is smart."

### ADR-003: Sub-scanner architecture over monolithic scan function
**Context**: The scanner could be one big function reading all file types, or decomposed into sub-scanners.
**Decision**: Sub-scanner structs with a common interface.
**Rationale**: Each evidence category (manifests, configs, schemas, etc.) has different file patterns, depth rules, and content extraction logic. Sub-scanners are independently testable and can be added without modifying existing ones (OCP). If a user reports "Go scanning doesn't detect my project", we fix `manifestScanner` without touching `schemaScanner`.

### ADR-004: Markdown over XML for scan report
**Context**: Repomix uses XML tags for AI-friendly formatting. Should we do the same?
**Decision**: Structured markdown with clear section headers.
**Rationale**: The scan report is consumed within an MCP tool response where the consumer is already an AI in a structured conversation. Markdown provides equivalent structure at ~30% less token cost than XML wrapping. Clear `##` headers with consistent formatting achieves the same structured parsing benefit that Anthropic recommends XML for in unstructured prompt contexts.
