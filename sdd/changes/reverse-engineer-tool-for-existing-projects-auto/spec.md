# Spec: sdd_reverse_engineer + sdd_change auto-trigger

> Based on: proposal (propose.md), context-check (context-check.md), market research (Repomix, Aider repo map, Anthropic prompt engineering)

---

## Functional Requirements

### Must Have

- **FR-001**: The tool `sdd_reverse_engineer` MUST accept a `detail_level` parameter with values `summary`, `standard` (default), and `full`, controlling the verbosity of the scan report — consistent with the pattern established by `sdd_context_check` and memory tools.

- **FR-002**: The scanner MUST detect and report the **directory structure** of the project, limited to a configurable depth (default: 3 levels), excluding common ignore patterns (node_modules, .git, __pycache__, vendor, dist, build, target, .next, .nuxt, venv, .venv).

- **FR-003**: The scanner MUST detect and read **package manifests** — at minimum: `package.json`, `go.mod`, `go.sum` (dependencies only), `requirements.txt`, `pyproject.toml`, `Cargo.toml`, `pom.xml`, `build.gradle`, `Gemfile`, `composer.json`. Found manifests are included in the report with key metadata (name, dependencies, scripts/commands).

- **FR-004**: The scanner MUST detect and read **convention files** — `CLAUDE.md`, `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, and files in `.cursor/rules/`, `.github/copilot-instructions.md`. Content is included up to 200 lines per file.

- **FR-005**: The scanner MUST detect and read **configuration files** — `tsconfig.json`, `.eslintrc*`, `Dockerfile`, `docker-compose.yml`, `.dockerignore`, CI config files (`.github/workflows/*.yml`, `.gitlab-ci.yml`, `Jenkinsfile`), `Makefile`, `.env.example`. Content included up to 100 lines per file.

- **FR-006**: The scanner MUST detect and read **entry points** — `main.go`, `cmd/*/main.go`, `index.ts`, `index.js`, `app.py`, `manage.py`, `main.py`, `src/main.rs`, `src/lib.rs`. Content is read at summary level (first 50 lines) to identify imports and setup patterns.

- **FR-007**: The scanner MUST detect and read **existing ADR files** — scanning directories `docs/adr/`, `adr/`, `doc/decisions/`, `docs/decisions/`, `architectural-decisions/`. ADR files are included in full (they're typically short).

- **FR-008**: The scanner MUST detect and read **database schemas** — migration directories (`migrations/`, `db/migrate/`, `alembic/versions/`, `prisma/schema.prisma`, `drizzle/`, `*.sql` in common schema directories), ORM model files. Content limited to schema definitions and model structures.

- **FR-009**: The scanner MUST detect and read **API definitions** — OpenAPI/Swagger files (`openapi.yaml`, `swagger.json`, `openapi.json`), route definition files (detected by filename patterns like `routes.*`, `router.*`, `urls.py`). Content included in the report for the AI to analyze.

- **FR-010**: The scanner MUST return a **structured markdown report** organized in sections: (1) Project Overview (manifest summary), (2) Directory Structure (tree), (3) Tech Stack Evidence (configs + manifests), (4) Architecture Evidence (entry points + directory patterns), (5) Convention & Style (convention files), (6) Data Model Evidence (schemas), (7) API Evidence (specs + routes), (8) Prior Decisions (ADRs), (9) Files Skipped (what was detected but not read, and why).

- **FR-011**: The scanner MUST support a `max_tokens` parameter for token budget control. When the budget is exceeded, sections are truncated with `[...truncated by token budget]` markers, prioritizing earlier sections (project overview and tech stack) over later ones (API evidence, ADRs).

- **FR-012**: The scanner report MUST include a **metadata header** with: project root path, total files scanned, total files skipped, scan duration, and detected primary language/framework (inferred from manifests).

- **FR-013**: When `sdd_change` is called and **no SDD artifacts exist** in the project's `sdd/` directory (none of: `business-rules.md`, `design.md`, `requirements.md`), the tool MUST **block the change creation** (refuse to create it) and return a response that: (a) explains that no SDD context exists, (b) recommends the user run `sdd_reverse_engineer` first, (c) includes the text: "Run `sdd_reverse_engineer` to scan your project and generate baseline SDD artifacts before creating a change."

- **FR-014**: The scanner MUST work **without `sdd_init_project` having been run** — it does not require `sdd/sdd.json` to exist. It uses `findProjectRoot()` which falls back to cwd when no SDD project exists.

- **FR-015**: The scanner MUST degrade gracefully — any file or directory that cannot be read (permissions, encoding, missing) is silently skipped and reported in the "Files Skipped" section. No scan failure should abort the entire report.

- **FR-016**: The scanner report MUST include a prominently placed note for the AI: "This report is raw evidence. Analyze it to generate 3 SDD artifacts: `sdd/business-rules.md` (BRG taxonomy), `sdd/design.md` (architecture, tech stack, components), `sdd/requirements.md` (what the system currently does, MoSCoW format). Mark each artifact with: `> ⚡ Auto-generated by sdd_reverse_engineer — review and refine as needed`"

### Should Have

- **FR-017**: The scanner SHOULD detect **test directories and frameworks** — `test/`, `tests/`, `__tests__/`, `spec/`, `*_test.go`, `*.spec.ts`, `*.test.js`. Report which testing framework is used (Jest, pytest, Go testing, RSpec, etc.) and approximate test file count. This helps the AI understand test coverage patterns without reading test file contents.

- **FR-018**: The scanner SHOULD detect **monorepo structure** — if `packages/`, `apps/`, `services/`, `libs/`, or workspace declarations exist in package.json/pnpm-workspace.yaml, report each workspace/package with its own manifest summary.

- **FR-019**: The scanner SHOULD support a `scan_path` parameter (optional) to scan a subdirectory instead of the project root. This enables scanning individual packages in a monorepo.

### Could Have

- **FR-020**: The scanner COULD extract **exported symbols** (function/class/interface names) from key source files using simple regex patterns (e.g., `^export`, `^func `, `^class `, `^def `). This would provide the "signatures > implementation" approach inspired by Aider's repo map. Limited to files in top-level source directories.

- **FR-021**: The scanner COULD detect and report **environment variable usage** by scanning `.env.example`, `docker-compose.yml` env sections, and `process.env.*`/`os.Getenv()` patterns — WITHOUT reading `.env` files (security). This helps the AI understand the system's external dependencies.

### Won't Have (v1)

- **FR-W01**: Will NOT read all source files — only entry points, schemas, routes, and optionally exported symbols
- **FR-W02**: Will NOT execute any code or run any project commands
- **FR-W03**: Will NOT use tree-sitter or AST parsing (too heavy for an MCP tool — simple regex/file reading is sufficient)
- **FR-W04**: Will NOT generate the SDD artifacts directly — it produces the REPORT, the AI generates the artifacts using existing tools or direct file writes
- **FR-W05**: Will NOT read `.env`, `credentials.json`, or any file likely to contain secrets

---

## Non-Functional Requirements

- **NFR-001**: The scan MUST complete in under **5 seconds** for projects with up to 10,000 files. Directory traversal uses efficient walk with early pruning of ignored directories.

- **NFR-002**: The scan report at `standard` detail level MUST stay under **8,000 tokens** for typical projects (< 5,000 files). The `summary` level MUST stay under **2,000 tokens**. The `full` level has no cap but respects `max_tokens` when set.

- **NFR-003**: The tool MUST NOT allocate more than **50MB of memory** during the scan. Large files are read with size limits (skip files > 100KB for content reading, count-only for directory traversal).

- **NFR-004**: The tool binary MUST continue to compile with `CGO_ENABLED=0` (static binary). No C dependencies, no tree-sitter, no external tools. Pure Go file I/O only.

- **NFR-005**: The tool MUST support at minimum these project ecosystems: **Go**, **Node.js/TypeScript**, **Python**, **Rust**. Additional ecosystems (Java, Ruby, PHP, C#) are detected via manifests but with less specialized scanning.

- **NFR-006**: The `sdd_change` auto-trigger check (FR-013) MUST add less than **10ms** of latency to the `sdd_change` call. It checks for the existence of 3 files — fast filesystem stat operations only.

- **NFR-007**: The scanner MUST follow Hoofy's **SRP convention** — one file per tool: `internal/tools/reverse_engineer.go`. Dependencies injected via constructor, registered in `server.go`.

---

## Assumptions

- Projects being scanned have a conventional directory structure recognizable by file patterns (manifests, configs, standard directory names).
- The AI calling the tool has enough context window to process the scan report plus generate 3 artifacts.
- Users will review and refine auto-generated artifacts — they are a starting point, not final specs.
- The project is accessible from the filesystem where Hoofy runs (local directory, not remote repos).

## Dependencies

- `findProjectRoot()` helper (already exists in `helpers.go`)
- `writeStageFile()` helper (already exists — but the reverse engineer tool itself does NOT write artifacts, it only produces the report)
- `memory.DetailLevel` types and `detail_level` parameter pattern (already established)
- `changes.Store` for `sdd_change` modification (already injected)
- Existing SDD artifact file paths match what `sdd_context_check` already reads (`business-rules.md`, `requirements.md`, `design.md`)

## Constraints

- No CGO — pure Go stdlib for all file operations
- No external tools — cannot shell out to `tree`, `find`, `fd`, or any CLI tool
- No network access — scanner operates on local filesystem only
- MCP tool response size is bounded by the client's context window — token budgeting is essential
- Must not modify any project files — read-only scanner

---

## Report Format Decision (from market research)

**Decision**: Structured Markdown with section headers (not XML/JSON).

**Rationale** (synthesized from Repomix, Aider, Anthropic research):

1. **Markdown over XML**: Repomix uses XML for broad AI compatibility, but Hoofy's response is always consumed by a single AI within an MCP tool call. Markdown section headers provide equivalent structure with ~30% less token overhead than XML tags.

2. **Sections by evidence type**: Inspired by Repomix's `<file_summary>` + `<files>` structure but adapted to Hoofy's domain. Sections map to the 3 target artifacts: tech stack/architecture evidence → `design.md`, schemas/routes → `requirements.md`, conventions/patterns → `business-rules.md`.

3. **Signatures > implementation**: Inspired by Aider's repo map — we don't dump all code, we capture structure, names, and relationships. Entry points are read for imports/setup (first 50 lines), not full implementation.

4. **Hierarchical detail levels**: Like Repomix's `--compress` mode, our `detail_level` parameter controls verbosity. `summary` = filenames only, `standard` = key excerpts, `full` = complete content where available.

5. **Token budgeting**: Both Repomix and Aider prioritize token efficiency. Our `max_tokens` parameter with section-priority truncation ensures the report fits the AI's context window.

### Report Template

```markdown
# Reverse Engineer Scan Report

> **Project**: <root path>
> **Scanned**: <N> files | **Skipped**: <M> files | **Duration**: <Xms>
> **Primary ecosystem**: <Go|Node.js|Python|Rust|Unknown>

⚠️ **AI Instructions**: This report is raw evidence. Analyze it to generate 3 SDD artifacts:
1. `sdd/business-rules.md` — Domain terms, facts, constraints (BRG taxonomy)
2. `sdd/design.md` — Architecture overview, tech stack, components, data model
3. `sdd/requirements.md` — What the system currently does (MoSCoW format)

Mark each artifact with: `> ⚡ Auto-generated by sdd_reverse_engineer — review and refine as needed`

---

## 1. Project Overview
<manifest name, version, description, scripts>

## 2. Directory Structure
<tree output, depth-limited>

## 3. Tech Stack Evidence
<configs, manifests, dependency lists>

## 4. Architecture Evidence
<entry points, import patterns, directory organization>

## 5. Conventions & Style
<CLAUDE.md, AGENTS.md, linting configs>

## 6. Data Model Evidence
<schemas, migrations, ORM models>

## 7. API Evidence
<OpenAPI specs, route definitions>

## 8. Prior Decisions
<ADR files>

## 9. Files Skipped
<what was detected but not read, with reasons>
```
