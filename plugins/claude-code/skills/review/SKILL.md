---
description: >
  Review the current status of SDD pipeline and active changes.
  Shows progress, completed stages, and next steps.
  Use when checking project status, pipeline progress, or asking "where are we?".
---

# Review SDD Status

Check the current state of the Hoofy SDD pipeline and any active changes.

## What to do

1. Call `mcp__hoofy__sdd_get_context` with `detail_level: "standard"` to see the project pipeline overview.

2. Call `mcp__hoofy__sdd_change_status` (no arguments) to check if there's an active change and its progress.

3. Present a clear summary to the user:
   - **Project Pipeline**: Which stages are complete, in progress, or pending
   - **Active Change**: Current change ID, type, size, stage progress
   - **Next Step**: What needs to happen next

4. If there's a completed change pipeline but no active change, mention that the user can start a new one with `/hoofy:change`.

5. If neither pipeline has been initialized, suggest starting with `/hoofy:init` for a new project or `/hoofy:change` for an existing one.

## Format

Present the status in a clear, visual format. Use checkmarks for completed stages, arrows for current, and empty circles for pending. Example:

```
Project Pipeline: ✅ init → ✅ propose → ✅ specify → 🔄 clarify → ⬜ design → ⬜ tasks → ⬜ validate

Active Change: feature/medium — "Add user authentication"
  ✅ propose → 🔄 spec → ⬜ tasks → ⬜ verify
```
