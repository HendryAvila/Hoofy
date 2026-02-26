# Context Check: Memory Compaction Tool (F4)

## Existing Artifacts Analysis

### Alignment with Project Architecture
- **No conflicts** with existing SDD pipeline (proposal.md, requirements.md, design.md) — this is a new memory tool, not a pipeline stage
- Memory tools live in `internal/memtools/*.go`, registered in `server.go` composition root — follows established pattern
- Store methods live in `internal/memory/store.go` — follows existing `CountObservations`, `DeleteObservation`, `AddObservation` patterns

### Related Prior Changes
- **F1 (detail_level)**, **F2 (mem_progress)**, **F3 (navigation hints)** — all followed same pattern: store method + memtool handler + test + server instructions. F4 follows identical pattern.
- **Knowledge graph relations** — `mem_relate` uses `supersedes` relation type. F4's compacted-from traceability could use this existing mechanism OR a simpler metadata approach.

### Relevant Existing Mechanisms
1. **Soft-delete** (`DeleteObservation(id, false)`) — already exists, sets `deleted_at` timestamp. F4 uses this for batch cleanup.
2. **Topic-key upsert** — `AddObservation` with `topic_key` already increments `revision_count` and updates content. This is "implicit compaction" for evolving topics.
3. **`CountObservations(project, scope)`** — from F3, useful for showing "before/after" compaction counts.

## Ambiguity Analysis (IEEE 29148)

### Identified & Resolved
1. **"Older than N days"** — What's N? → Design decision: user-provided parameter `older_than_days`, no default. Forces explicit choice.
2. **"Batch soft-delete + summary atomically"** — True SQL transaction? → Yes, wrap in `BEGIN/COMMIT` for atomicity.
3. **"Compacted-from metadata"** — How to trace? → Simple approach: store the list of compacted IDs in the summary observation's content (the AI writes it). No new schema columns.

### No Issues Found
- No scope creep — proposal explicitly excludes auto-compaction, hard-delete, session merging, token budgets
- No conflicts with existing tools — `mem_delete` handles single observations, `mem_compact` handles batch + summary

## Impact Classification
- **Non-breaking**: Adds new tool (`mem_compact`), new store methods. No changes to existing tool behavior.
- **Tool count**: 33 → 34

## Verdict: ✅ Clear to proceed
