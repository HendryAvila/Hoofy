# Clarification: sdd_explore

## Questions & Answers

### Q1: Dependency model — Memory vs Pipeline?
**Question**: `sdd_explore` depends ONLY on `memory.Store` (like memtools), not on `config.Store` or `changes.Store` (like SDD pipeline tools). This means it lives architecturally closer to the memory system, even though its name has the `sdd_` prefix. Is this correct?

**Answer**: Yes. User agrees that `sdd_explore` should be standalone and live closer to the memory world, even though it softly integrates into the pipeline via `serverInstructions()`. The `sdd_` prefix reflects its purpose (pre-pipeline context), not its dependency graph.

**Impact**: Tool file goes in `internal/tools/explore.go` (with other SDD tools), but constructor only needs `*memory.Store`. No `config.Store` or `changes.Store` dependency.

### Q2: FR-008 Auto-suggest type/size — Include in v1?
**Question**: Should the tool auto-suggest change type and size based on explore content analysis? This was listed as "Could Have" in the spec.

**Answer**: Yes, user wants this for v1. Promote FR-008 from "Could Have" to "Should Have".

**Impact**: The tool response should include a `suggested_type` and `suggested_size` section when the explore content contains enough signal. This is a heuristic analysis of the goals/constraints/context text — not ML, just keyword matching (e.g., "fix" → type=fix, "quick" → size=small, "new feature" → type=feature). Should be clearly labeled as a *suggestion*, not a decision.

### Q3: Tool name confirmation
**Question**: Is `sdd_explore` the right name, or should it be `sdd_discuss`, `sdd_discover`, or `sdd_context`?

**Answer**: `sdd_explore` is confirmed. No issues with the name.

**Impact**: None — proceed with `sdd_explore` as the tool name.

## Resolved Ambiguities from Proposal

| Open Question | Resolution |
|---------------|------------|
| Accept all categories at once or separate params? | Separate params (FR-001) — each category is an optional parameter |
| Separate `sdd_get_explore_context` tool? | No (WH-003) — `mem_search(type=explore)` is sufficient |
| Auto-link to changes/projects via relations? | No (WH-002) — manual linking with `mem_relate` if needed |

## Updated Requirement Status

- **FR-008** promoted from "Could Have" → **"Should Have"**
- All other requirements unchanged
- No new requirements identified

## Confidence Assessment

All dimensions are clear:
- **Scope**: Well-defined — single tool, memory-only dependency, serverInstructions update
- **Interface**: Clear — 8 parameters defined in FR-001, response format in FR-005/FR-007
- **Integration**: Clear — hybrid approach via serverInstructions, no state machine changes
- **Edge cases**: Covered — upsert behavior (FR-003), at-least-one-param validation (FR-001)
- **Dependencies**: Minimal — only `memory.Store`, existing upsert mechanism

No remaining ambiguities. Ready for design.
