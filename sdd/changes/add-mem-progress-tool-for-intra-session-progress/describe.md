# F2: mem_progress — Intra-Session Progress Tracking

## What
Add a `mem_progress` tool that persists structured progress documents for long-running agent sessions. Unlike session summaries (written at session END), progress docs track INTRA-session work-in-progress: current goal, completed steps, next steps, and blockers.

## Why
From [Anthropic's Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents):
> "Each session: read progress, read git log, run basic test, then start new work"

When an AI agent's context window compacts or a session spans multiple windows, the agent loses track of where it was. `mem_progress` solves this by providing a persistent, structured progress document that survives compaction. The key insight from the research: **use JSON, not Markdown** — "Feature list in JSON (not Markdown) — model less likely to inappropriately change JSON."

## How It Works

### Single Tool: `mem_progress`
One tool with dual behavior:
- **No `content` param** → READ current progress for the project (auto-read at session start)
- **With `content` param** → UPSERT progress for the project (one active progress per project)

### Data Model
Progress is stored as a regular observation with:
- `type`: `"progress"` (new observation type)
- `topic_key`: `"progress/<project>"` (upsert ensures ONE active progress per project)
- `content`: Structured JSON string with `goal`, `completed`, `next_steps`, `blockers` fields

The AI generates the JSON content. The tool validates it's valid JSON before saving.

### Parameters
| Parameter | Required | Description |
|-----------|----------|-------------|
| `project` | Yes | Project name — used as the topic_key suffix |
| `content` | No | JSON progress document. If omitted, reads current progress |
| `session_id` | No | Session ID to associate (default: manual-save) |

### Read Behavior
When called without `content`:
1. Search observations where `topic_key = "progress/<project>"` and `type = "progress"`
2. Return the latest one (topic_key upsert ensures only one exists)
3. If none found, return "No progress document found for project X"

### Write Behavior
When called with `content`:
1. Validate that `content` is valid JSON
2. Save as observation with `topic_key = "progress/<project>"`, `type = "progress"`
3. Topic key upsert automatically replaces the previous progress doc
4. Return confirmation with observation ID

## Integration with Server Instructions
Add guidance in `serverInstructions()`:
- At session start: call `mem_progress` (read mode) to check for existing WIP
- During work: call `mem_progress` (write mode) after completing significant steps
- Before session end: update progress with final state

## Scope
- One new file: `internal/memtools/progress.go`
- Register in `internal/server/server.go` (composition root)
- Update `serverInstructions()` with usage guidance
- Update `docs/research-foundations.md` with F2 mapping
- Tests in existing `internal/memtools/memtools_test.go`

## What This Is NOT
- NOT a replacement for `mem_session_summary` (that's end-of-session, structured differently)
- NOT a task tracker (no task IDs, no status per task — that's the SDD pipeline)
- NOT a todo list — it's a WIP state document for the AI agent itself
