# Tasks: Response Verbosity Control

## Task Breakdown

### TASK-001: Add shared `detail_level` helper
**Component**: internal/memtools (or internal/memory)
**Covers**: FR-019
**Dependencies**: None
**Description**: Create a shared helper for parsing the `detail_level` parameter — constants (`DetailSummary`, `DetailStandard`, `DetailFull`), a parsing function that defaults to `standard`, and the enum values list for tool definitions. This prevents duplicating the same logic across 4 tools.
**Acceptance Criteria**:
- [ ] Constants defined for all 3 levels
- [ ] Parser function handles empty string → standard default
- [ ] Parser function handles invalid values gracefully (fall back to standard)
- [ ] Enum values list available for tool definitions
- [ ] Unit tests for parser

### TASK-002: Add `detail_level` to `mem_context`
**Component**: internal/memtools/context.go, internal/memory/store.go
**Covers**: FR-001, FR-002, FR-003, FR-004, FR-005
**Dependencies**: TASK-001
**Description**: Add `detail_level` parameter to `mem_context` tool definition and Handle(). Modify `FormatContext()` in store.go (or add parallel methods) to support 3 tiers. Fix the dead `limit` parameter to actually control observation count. Summary mode: session names + observation titles only. Full mode: untruncated content.
**Acceptance Criteria**:
- [ ] `detail_level` parameter appears in tool definition with enum constraint
- [ ] `summary` returns no content snippets
- [ ] `standard` matches current behavior exactly
- [ ] `full` returns untruncated content
- [ ] `limit` parameter actually controls observation count
- [ ] Tests for all 3 levels + limit parameter

### TASK-003: Add `detail_level` to `mem_search`
**Component**: internal/memtools/search.go
**Covers**: FR-006, FR-007, FR-008, FR-009
**Dependencies**: TASK-001
**Description**: Add `detail_level` parameter to `mem_search` tool definition and Handle(). Summary: IDs + types + titles only. Standard: current 300-char snippets. Full: complete content per result.
**Acceptance Criteria**:
- [ ] `detail_level` parameter with enum constraint
- [ ] `summary` returns no content (only IDs, types, titles)
- [ ] `standard` matches current 300-char truncation
- [ ] `full` returns complete untruncated content
- [ ] Tests for all 3 levels

### TASK-004: Add `detail_level` to `mem_timeline`
**Component**: internal/memtools/timeline.go
**Covers**: FR-010, FR-011, FR-012, FR-013
**Dependencies**: TASK-001
**Description**: Add `detail_level` parameter to `mem_timeline` tool definition and Handle(). Summary: titles + timestamps only. Standard: current behavior (200-char before/after, full focus). Full: all entries untruncated.
**Acceptance Criteria**:
- [ ] `detail_level` parameter with enum constraint
- [ ] `summary` returns only metadata
- [ ] `standard` matches current behavior (200-char before/after, full focus)
- [ ] `full` returns untruncated content for ALL entries
- [ ] Tests for all 3 levels

### TASK-005: Add `detail_level` to `sdd_context_check`
**Component**: internal/tools/context_check.go
**Covers**: FR-014, FR-015, FR-016, FR-017
**Dependencies**: TASK-001
**Description**: Add `detail_level` parameter to `sdd_context_check` tool definition and Handle(). Summary: artifact filenames + sizes, change slugs only. Standard: current truncated excerpts. Full: complete artifact content and full memory content.
**Acceptance Criteria**:
- [ ] `detail_level` parameter with enum constraint
- [ ] `summary` returns only filenames, sizes, slugs
- [ ] `standard` matches current behavior (500-char excerpts)
- [ ] `full` returns untruncated content
- [ ] Tests for all 3 levels

### TASK-006: Add summary footer hints
**Component**: All 4 modified tools
**Covers**: FR-020
**Dependencies**: TASK-002, TASK-003, TASK-004, TASK-005
**Description**: When a tool returns `summary` mode, append a footer: "Use detail_level: standard or full for more detail." This guides the AI toward progressive disclosure.
**Acceptance Criteria**:
- [ ] Footer appears in `summary` mode only
- [ ] Footer does NOT appear in `standard` or `full` modes
- [ ] Tests verify footer presence/absence

### TASK-007: Update server instructions
**Component**: internal/server/server.go
**Covers**: FR-018
**Dependencies**: TASK-002, TASK-003, TASK-004, TASK-005
**Description**: Update `serverInstructions()` to guide the AI on when to use each detail level. Guidance: use `summary` for exploration and routing decisions, `standard` for general work, `full` only when deep analysis is needed.
**Acceptance Criteria**:
- [ ] Server instructions mention `detail_level` parameter
- [ ] Guidance on when to use each level
- [ ] No breakage of existing instructions

### TASK-008: Update docs/research-foundations.md
**Component**: docs/research-foundations.md
**Covers**: Documentation
**Dependencies**: TASK-007
**Description**: Add the new `detail_level` feature to the research foundations doc, mapping it to its Anthropic sources.
**Acceptance Criteria**:
- [ ] Feature listed under "Writing Effective Tools" section
- [ ] Feature listed under "Context Engineering" section
- [ ] Links to both source articles

## Dependency Graph

```
TASK-001 (shared helper)
├── TASK-002 (mem_context)     ← can parallel
├── TASK-003 (mem_search)      ← can parallel
├── TASK-004 (mem_timeline)    ← can parallel
└── TASK-005 (context_check)   ← can parallel
         ↓
    TASK-006 (footer hints)    ← after all tools done
    TASK-007 (server instrs)   ← after all tools done
         ↓
    TASK-008 (docs update)     ← final
```

## Wave Assignments

**Wave 1** (no dependencies):
- TASK-001: Shared detail_level helper

**Wave 2** (parallel — depends on Wave 1):
- TASK-002: mem_context
- TASK-003: mem_search
- TASK-004: mem_timeline
- TASK-005: sdd_context_check

**Wave 3** (parallel — depends on Wave 2):
- TASK-006: Summary footer hints
- TASK-007: Server instructions

**Wave 4** (depends on Wave 3):
- TASK-008: Docs update

## Estimated Effort

8 tasks across 4 waves. Estimated: **3-4 hours** for a single developer familiar with the codebase.
