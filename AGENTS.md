# AGENTS.md — Hoofy Development Guide

## Project Overview

Hoofy is an **AI development companion** — an MCP server written in Go 1.25 that provides persistent memory, adaptive change management, and spec-driven development. It prevents AI hallucinations by forcing structured requirements before coding, remembers context across sessions, and adapts its workflow to the size and type of each change.

**Binary**: `hoofy` (CLI with `serve`, `update`, `version` commands)
**Transport**: stdio (MCP protocol)
**Dependency**: `mcp-go v0.44.0` (Mark3Labs MCP SDK)

## Architecture

### Directory Structure

```
cmd/hoofy/              Entry point — CLI argument parsing, server startup, graceful shutdown
internal/
├── changes/            Change pipeline — types, flows, store, state machine
├── config/             Project config persistence (hoofy.json) — types, Store interface, FileStore
├── memory/             Persistent memory — SQLite store, FTS5 search, sessions, observations
├── memtools/           MCP memory tool handlers — 19 tools for save, search, context, sessions, relations, progress
├── pipeline/           Pipeline state machine — stage transitions, Clarity Gate thresholds
├── prompts/            MCP prompts — /sdd-start, /sdd-status, /sdd-stage-guide, /sdd-memory-guide, /sdd-change-guide, /sdd-bootstrap-guide
├── resources/          MCP resources — project status resource
├── server/             Composition root — wires all dependencies, registers tools/prompts/resources
├── templates/          Go templates for stage artifacts (guided + expert mode variants)
├── tools/              MCP tool handlers — one file per tool (init, principles, charter, specify, clarify, design, tasks, validate, context, change, adr, audit, bridge, suggest_context, review)
└── updater/            Self-update system — GitHub releases API, binary replacement
```

### Design Principles

- **SRP**: One file = one tool. Config handles persistence, pipeline handles business rules.
- **DIP**: Tools depend on interfaces (`config.Store`, template renderer), not concretions. `server.go` is the composition root that wires everything.
- **OCP**: New tools/stages are added without modifying existing ones.
- **Liskov**: Both modes (guided/expert) use the same pipeline interface, differing only in thresholds and templates.

### Key Abstractions

```
config.Store (interface)
├── config.Loader — reads hoofy.json
└── config.Saver — writes hoofy.json
    └── config.FileStore (concrete) — filesystem implementation

memory.Store (interface)
├── SaveObservation, SearchObservations, GetContext, GetTimeline
├── Session management (Start, End, Summary)
└── memory.SQLiteStore (concrete) — SQLite + FTS5 implementation
```

Tools receive `config.Store`, `memory.Store`, and `templates.Renderer` via constructor injection in `server.go`.

### Pipeline State Machine

```
init → principles → charter → specify → business-rules → clarify (Clarity Gate) → design → tasks → validate
```

- 9 stages, sequential — cannot skip ahead.
- The `principles` stage captures golden invariants and coding standards.
- The `charter` stage replaces the old `propose` stage with expanded enterprise-grade fields.
- The Clarity Gate at `clarify` blocks advancement until clarity score meets threshold:
  - Guided mode: 70/100
  - Expert mode: 50/100
- Stage status: `pending` → `in_progress` → `completed`
- State persisted in `docs/hoofy.json` in the user's project directory.

### Change Pipeline State Machine

```
create → context-check → [describe/charter/scope] → [spec] → [clarify] → [design] → tasks → verify
                          └─── stages selected automatically based on type × size ───┘
```

- 4 types (feature, fix, refactor, enhancement) × 3 sizes (small, medium, large) = 12 flow variants
- One active change at a time
- Artifacts stored in `docs/changes/<slug>/`
- Completed changes archived to `docs/history/<slug>/`

### MCP Components

| Type | Components |
|------|-----------|
| **Tools (Project)** | `sdd_init_project`, `sdd_create_principles`, `sdd_create_charter`, `sdd_generate_requirements`, `sdd_create_business_rules`, `sdd_clarify`, `sdd_create_design`, `sdd_create_tasks`, `sdd_validate`, `sdd_get_context`, `sdd_reverse_engineer`, `sdd_bootstrap` |
| **Tools (Change)** | `sdd_change`, `sdd_context_check`, `sdd_change_advance`, `sdd_change_status`, `sdd_adr` |
| **Tools (Standalone)** | `sdd_explore`, `sdd_suggest_context`, `sdd_review`, `sdd_audit` |
| **Tools (Memory)** | `mem_save`, `mem_save_prompt`, `mem_search`, `mem_context`, `mem_timeline`, `mem_get_observation`, `mem_relate`, `mem_unrelate`, `mem_build_context`, `mem_session_start`, `mem_session_end`, `mem_session_summary`, `mem_stats`, `mem_capture_passive`, `mem_delete`, `mem_update`, `mem_suggest_topic_key`, `mem_progress`, `mem_compact` |
| **Prompts** | `/sdd-start`, `/sdd-status`, `/sdd-stage-guide`, `/sdd-memory-guide`, `/sdd-change-guide`, `/sdd-bootstrap-guide` |
| **Resources** | Project status resource |

Tools are STORAGE tools — the AI generates content, tools save it to disk and advance the pipeline.

## Development

### Commands

```bash
make build          # Build binary to bin/hoofy (injects version via ldflags)
make test           # Run tests with race detector and coverage
make lint           # Run golangci-lint
make tidy           # Clean up go.mod/go.sum
make run            # Build and run the MCP server
```

### Testing

- Tests use `go test -race -cover ./...`
- Pipeline tests use `time.go` with injectable clock for deterministic timestamps.
- Config tests use temp directories for filesystem isolation.
- Memory tests use `t.TempDir()` for SQLite DB isolation.
- Template tests verify both guided and expert mode rendering.

### CI/CD

- **CI** (`.github/workflows/ci.yml`): test → lint → build on push/PR to main.
- **Release** (`.github/workflows/release.yml`): GoReleaser on tag push, builds for linux/darwin/windows on amd64/arm64.
- **Version**: injected at build time via `-ldflags -X server.Version=<tag>`.

### Adding a New Tool

1. Create `internal/tools/<name>.go` with a struct holding dependencies.
2. Implement `Definition()` returning `mcp.Tool` and `Handle()` with `server.ToolHandlerFunc` signature.
3. Register in `internal/server/server.go` (composition root).
4. If the tool needs a new stage, add it to `config.StageOrder` and `config.Stages`.

### Adding a Memory Tool

1. Create `internal/memtools/<name>.go` with a struct holding `memory.Store`.
2. Implement `Definition()` returning `mcp.Tool` and `Handle()`.
3. Register in `internal/server/server.go` (composition root).

### Adding a New Stage

1. Add the `Stage` constant in `config/config.go`.
2. Add it to `StageOrder` and `Stages` map.
3. Add the filename mapping to `stageFilenames`.
4. Create the tool handler in `internal/tools/`.
5. Create templates in `internal/templates/` if needed.

## Conventions

- **Commit style**: Conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `ci:`, `chore:`).
- **Error handling**: Wrap errors with `fmt.Errorf("context: %w", err)` for chain.
- **File output**: Stage artifacts go to `docs/<stage>.md` in the user's project. ADRs go to `docs/adrs/NNN-slug.md`.
- **No CGO**: All builds use `CGO_ENABLED=0` for static binaries.
- **Stderr for UI**: All user-facing messages (update notices, progress) go to stderr to keep stdout clean for MCP stdio transport.

## Release Checklist

**⚠️ ALWAYS check this before ending a session where features were merged to main.**

Users running `hoofy update` only see published GitHub Releases. If you merge features but don't tag a release, **users don't get them**. This has happened before (14 commits sat unreleased between v0.8.0 and v0.9.0).

### Pre-Release

1. **Verify tests pass**: `go test -race ./...` — all packages must pass.
2. **Check unreleased commits**: `git log --oneline $(git describe --tags --abbrev=0)..HEAD` — if there are meaningful commits (feat/fix), a release is needed.
3. **Verify working tree is clean**: `git status` — no uncommitted changes.

### Creating a Release

1. **Choose version**: Follow semver — `patch` for fixes, `minor` for features, `major` for breaking changes.
2. **Create annotated tag**: `git tag -a v<X.Y.Z> -m "Release v<X.Y.Z> — <summary>"` with categorized release notes.
3. **Push the tag**: `git push origin v<X.Y.Z>` — this triggers the GoReleaser workflow.
4. **Verify on GitHub**: Check Actions tab for successful release build (builds 6 platform binaries + updates Homebrew tap).

### Post-Release Verification

1. **GitHub Release page**: Confirm release appears at `https://github.com/HendryAvila/Hoofy/releases`.
2. **Binary availability**: Spot-check that platform archives are attached (e.g., `hoofy_X.Y.Z_linux_amd64.tar.gz`).
3. **Update check**: Run `hoofy update` from an older version to verify it detects the new release.

### When NOT to Release

- Only `docs:`, `test:`, `ci:`, or `chore:` commits — these don't affect the binary.
- Work-in-progress features that aren't ready for users.

### Release History

| Version | Date | Highlights |
|---------|------|------------|
| v1.0.0 | 2026-03-05 | Identity redesign: sdd/ → docs/, hoofy.json, principles + charter stages, sdd_audit, unified ADRs, auto-gen agent instructions |
| v0.13.0 | 2026-03-01 | Context Engineering v2: hot/cold instructions, sdd_suggest_context, sdd_review |
| v0.12.0 | 2026-03-01 | Structural quality analysis in design and validate stages |
| v0.11.0 | 2026-02-27 | Existing project bootstrap (sdd_reverse_engineer + sdd_bootstrap) |
| v0.10.0 | 2026-02-26 | Wave execution orchestration for multi-agent parallelization |
| v0.9.0 | 2026-02-26 | Context Engineering (F1-F6), Interactive Docs site |
| v0.8.0 | 2026-02-25 | Change pipeline, ADRs, context-check |

## Important Gotchas

- `findProjectRoot()` walks up directories looking for `docs/hoofy.json` (with `docs/specs/hoofy.json` fallback) — tools work from any subdirectory.
- Server instructions in `server.go` tell the AI HOW to use tools (generate content first, then save). This is critical — tools are dumb storage, not AI.
- The Clarity Gate is the core innovation — NEVER bypass or weaken it without understanding why it exists.
- Templates have guided and expert variants — always test both modes when modifying templates.
- Memory DB path defaults to `~/.hoofy/memory.db` — ensure directory exists before first write.
- Memory tools use FTS5 for search — always sanitize user input before passing to MATCH queries.
- ADRs are stored in `docs/adrs/` with sequential `NNN-slug.md` naming. The `nextADRNumber()` function handles gaps.
- `sdd_init_project` now auto-generates an SDD section in CLAUDE.md/AGENTS.md — it's idempotent (checks for existing header).
- Bridge topic keys use `sdd/` prefix (semantic, NOT filesystem path) — do not change to `docs/`.
