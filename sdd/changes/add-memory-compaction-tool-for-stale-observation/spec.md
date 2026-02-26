# Spec: Memory Compaction Tool (F4)

## Functional Requirements

### FR-001: `Store.FindStaleObservations(project, scope, olderThanDays, limit)` 
Returns observations older than N days, filtered by project and scope, ordered by age (oldest first). Excludes soft-deleted observations. Returns ID, type, title, project, scope, created_at, and content snippet (first 100 chars). Limit defaults to 50, max 200.

### FR-002: `mem_compact` tool — dual behavior (identify + execute)
**Identify mode** (no `compact_ids`): calls `FindStaleObservations` and returns a list of candidates for the AI to review. Includes observation count, age distribution, and navigation hint.

**Execute mode** (with `compact_ids`): accepts a JSON array of observation IDs to soft-delete. Optionally accepts `summary_title` and `summary_content` to create a replacement summary observation. Executes in a single SQL transaction: soft-delete all IDs, then insert summary if provided.

### FR-003: Tool parameters
- `project` (string, optional): Filter by project
- `scope` (string, optional): Filter by scope  
- `older_than_days` (number, required): Age threshold in days
- `compact_ids` (string, optional): JSON array of observation IDs to soft-delete `[1, 2, 3]`
- `summary_title` (string, optional): Title for replacement summary observation
- `summary_content` (string, optional): Content for replacement summary observation
- `session_id` (string, optional): Session to associate summary with (defaults to "manual-save")

### FR-004: Compaction execution response
Returns count of soft-deleted observations, the new summary observation ID (if created), and before/after total counts for the filtered project/scope.

### FR-005: Validation rules
- `older_than_days` must be > 0
- `compact_ids` must be valid JSON array of integers when provided
- If `summary_content` is provided, `summary_title` must also be provided
- Cannot compact observation IDs that are already soft-deleted
- Cannot compact if `compact_ids` is empty array

### FR-006: Summary observation metadata
When a summary is created during compaction, it has:
- `type`: "compaction_summary"
- `scope`: same as the compacted observations (or "project" default)
- `project`: same as filter project (if provided)
- Content written by the AI (includes whatever context the AI deems worth preserving)

### FR-007: Update server instructions
Add `mem_compact` usage guidance to serverInstructions in `server.go`. Document the two-phase workflow: identify first, then compact.

### FR-008: Update docs/research-foundations.md
Add F4 entry linking to Anthropic context engineering article's compaction recommendation.

### FR-009: Update tool count in README, AGENTS.md, docs/tool-reference.md
Tool count changes from 33 to 34.

## Non-Functional Requirements

### NFR-001: Batch delete performance
Soft-delete up to 200 observations in a single transaction — must complete in < 500ms on SQLite.

### NFR-002: Atomicity
If summary creation fails, no observations should be soft-deleted (transaction rollback).

### NFR-003: No schema changes
Use existing `observations` table and `deleted_at` column. No new tables or columns.

### NFR-004: All existing tests pass
Zero regressions after adding `mem_compact`.
