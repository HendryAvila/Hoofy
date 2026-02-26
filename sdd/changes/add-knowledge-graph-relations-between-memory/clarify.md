# Clarifications: Knowledge Graph Relations

## Questions & Answers

### Q1: Cascade behavior on soft-delete
**Question**: When an observation is soft-deleted (`deleted_at` is set), should relations remain visible or be hidden?

**Answer**: **Option A — Relations remain visible and functional.** Soft-deleted observations keep their relations intact. If restored, the graph is preserved. Rationale from user: "¿para qué me sirven dormidas si no las uso?"

**Impact on design**: `mem_build_context` does NOT filter out soft-deleted observations from traversal. Relations to/from soft-deleted observations are still followed. This simplifies implementation — no need for deleted_at checks during graph traversal.

### Q2: Output format of `mem_build_context`
**Question**: What data to return for each connected observation in the context tree?

**Answer**: **Option A — Lightweight: `id`, `title`, `type`, `relation_type`, `direction` only.** The AI calls `mem_get_observation` for full details when needed. Rationale from user: "el contexto detallado reduce alucinaciones" — giving the AI structured pointers is more useful than dumping everything.

**Impact on design**: `mem_build_context` returns a compact tree structure. Each node has: observation metadata (id, title, type, project, created_at) + edge metadata (relation_type, direction, note). Full content is NOT included — keeps token usage low and lets the AI decide what to drill into.

### Q3: Default depth for graph traversal
**Question**: Default value for `depth` parameter in `mem_build_context`?

**Answer**: **Default depth = 2.** Rationale: depth 1 is too conservative (misses valuable one-hop-away context), depth 2 gives the "close circle" — decisions that impacted what you're looking at. Maximum remains 5 as escape hatch.

**Impact on design**: `mem_build_context(observation_id)` without explicit depth traverses 2 levels. This typically returns 10-20 observations — enough context without flooding the AI's context window.

## Design Decisions Locked

1. **Soft-delete transparency**: Relations survive soft-delete, are visible in traversal
2. **Lightweight output**: Compact node metadata, no full content in graph traversal
3. **Depth 2 default**: Balance between context richness and noise avoidance
4. **Hard-delete cascade**: `ON DELETE CASCADE` on foreign keys (already confirmed: `PRAGMA foreign_keys = ON` is set in `store.go:333`)
5. **No duplicate relations**: `UNIQUE INDEX` on `(from_id, to_id, type)` prevents duplicates at DB level
6. **No self-relations**: Application-level validation (`from_id != to_id`)
7. **Bidirectional flag**: Optional `bidirectional` parameter on `mem_relate` creates both directions atomically
