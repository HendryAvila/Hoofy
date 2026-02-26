# Proposal: Wave Metadata for Task Parallelization

## Problem

Hoofy's task output (both `sdd_create_tasks` in the project pipeline and `sdd_change_advance` at the tasks stage in the change pipeline) produces a flat, ordered list of tasks with a textual dependency graph. While the dependency graph shows which tasks depend on others, there's no structured, machine-readable metadata that tells AI clients **which tasks can run in parallel**.

Modern AI tools (Claude Code with subagents, Cursor with parallel edits, etc.) can execute multiple tasks concurrently — but only if they know which tasks are safe to parallelize. Currently, the AI has to manually parse the dependency graph text to figure this out.

GSD (Get Shit Done) solves this with "waves" — groups of tasks that can execute in parallel because they have no interdependencies. Tasks within the same wave are independent; wave N+1 depends on wave N being complete.

## Proposed Solution

Add **optional wave metadata** to the task output in both pipelines:

1. **New parameter `wave_assignments`** on `sdd_create_tasks` — a structured field where the AI specifies which wave each task belongs to
2. **Wave section in the task template** — renders wave assignments as a structured table/section in `tasks.md`
3. **Change pipeline awareness** — when `sdd_change_advance` saves a `tasks` stage, the content can include wave metadata (no tool changes needed here since content is freeform markdown)

The AI does the analysis (it already has the dependency graph); Hoofy structures and persists the output.

## Scope

### In Scope
- New optional `wave_assignments` parameter on `sdd_create_tasks`
- Updated `tasks.md.tmpl` template with wave section
- Updated `TasksData` struct to include wave data
- Updated `serverInstructions()` to guide the AI on wave assignment
- Tests for new parameter handling and template rendering

### Out of Scope
- Wave execution engine (Hoofy is a storage tool, not an orchestrator)
- Changes to the change pipeline tool signatures (content is freeform markdown)
- Automatic wave computation (the AI does the analysis)
- Breaking changes to existing task format (wave metadata is additive and optional)

## Success Criteria
- `sdd_create_tasks` accepts an optional `wave_assignments` parameter
- When provided, the tasks.md output includes a structured "Execution Waves" section
- When omitted, output is identical to current format (backwards compatible)
- `serverInstructions()` guides the AI to generate wave assignments
- All existing tests pass unchanged
- New tests cover wave metadata rendering