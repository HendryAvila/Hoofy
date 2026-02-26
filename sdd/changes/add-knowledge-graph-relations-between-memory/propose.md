# Proposal: Knowledge Graph Relations Between Memory Observations

## Problem Statement

Hoofy's memory system stores observations as **isolated records** — each observation is a flat row in SQLite with FTS5 search. When a user searches for "auth middleware", they get that observation but NOT the related architecture decision that led to it, the bug fix that changed it, or the ADR that justified the approach.

This is like having a filing cabinet where every document is in its own drawer with no cross-references. You find ONE document, but the CONTEXT around it — the web of related decisions, discoveries, and changes — is invisible.

Competitors like Basic Memory (2.6K stars) and ConPort (746 stars) have knowledge graphs with typed relations. This is the #1 feature that drives Basic Memory's adoption.

## Proposed Solution

Add a **relations system** to Hoofy's memory that connects observations with typed, directional edges. This transforms flat observations into a navigable knowledge graph.

- New `relations` table in SQLite (additive, non-destructive migration)
- New MCP tools: `mem_relate` (create relations), `mem_build_context` (graph traversal)
- Existing tools (`mem_save`, `mem_search`, `mem_context`) continue working unchanged
- Relations are **opt-in** — existing memory databases work without modification

## Core Requirements

### Backwards Compatibility (NON-NEGOTIABLE)
- Users upgrading from any previous Hoofy version MUST NOT experience breakage
- Existing `memory.db` files MUST work without manual migration
- Schema migration MUST be automatic and non-destructive (`CREATE TABLE IF NOT EXISTS`)
- All 14 existing memory tools MUST behave identically with or without relations
- Zero data loss on upgrade — observations, sessions, FTS5 index all preserved

### Relation Model
- Typed directional edges between observations (from → to with relation type)
- Built-in relation types: `relates_to`, `implements`, `depends_on`, `caused_by`, `supersedes`, `part_of`
- Custom relation types allowed (string field, not enum)
- Bidirectional traversal (find what points TO an observation, and what it points FROM)
- Optional metadata/note on each relation

### Graph Traversal
- `mem_build_context(observation_id, depth)` — follow relations N levels deep
- Returns a structured context tree, not just a flat list
- Depth limit to prevent runaway traversal (max 3-5 levels)
- Cycle detection to prevent infinite loops

### New Tools
- `mem_relate` — create a relation between two observations
- `mem_unrelate` — remove a relation
- `mem_build_context` — traverse the graph from a starting observation

### Integration with Existing Tools
- `mem_search` results MAY show relation count per observation (lightweight)
- `mem_get_observation` MAY show direct relations (1 level)
- `mem_save` does NOT auto-create relations (explicit only)

## Out of Scope
- Visualization / canvas rendering (that's a client concern)
- memory:// URL scheme (separate feature, can be added later)
- Relation inference / AI-suggested relations (future enhancement)
- Bulk import/export of relations
- Relation types as enum/config (just strings for now)

## Success Criteria
- Existing users upgrading to new version experience ZERO breakage
- `mem_relate` + `mem_build_context` work end-to-end
- Graph traversal with depth=3 returns structured context tree
- Relations survive `mem_delete` of one side (cascade or orphan cleanup)
- All existing tests continue passing without modification

## Open Questions
- Should `mem_delete` cascade-delete relations, or leave orphaned relations?
- Should `mem_build_context` output include the full observation content or just titles+IDs?
- Maximum depth limit: 3 or 5?
