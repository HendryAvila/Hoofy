# Proposal: Memory Compaction Tool (F4)

## Problem
As Hoofy accumulates observations across sessions, memory grows unbounded. When an AI agent calls `mem_context` or `mem_search`, it receives ALL qualifying results (up to limit). Over weeks of use, this means:
- **Stale observations** crowd out recent, relevant ones
- **Duplicate/superseded content** wastes tokens (topic_key upserts only keep the latest, but old versions remain soft-available)
- **Session bloat** — many short sessions with 0-2 observations add noise to session lists
- The agent has **no way to know** which observations are stale without reading them all

## Research Source
Anthropic's ["Effective Context Engineering for AI Agents"](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) (Sep 2025):
> "Compaction: summarize conversation near context limit, reinitiate with summary"
> "Context is a finite resource with diminishing marginal returns"

## Proposed Solution
Add a `mem_compact` tool that enables the AI agent to identify stale observations, batch soft-delete them, and optionally create a summary observation that replaces them. The tool is a **storage helper** — the AI generates the summary content, the tool handles the batch delete + summary save atomically.

### Core Behavior
1. **Identify candidates**: Query observations older than N days, filtered by project/scope
2. **Return candidates for review**: List stale observations with metadata (age, project, type)
3. **Execute compaction**: Accept a list of observation IDs to soft-delete + an optional summary observation to create as replacement
4. **Traceability**: The summary observation links to the compacted IDs via `compacted_from` metadata

### What This Is NOT
- Not automatic — the AI decides what to compact and writes the summary
- Not lossy — soft-delete only, observations recoverable
- Not a background job — explicit tool call, explicit action

## Out of Scope
- Automatic/scheduled compaction (no cron, no background workers)
- Hard deletion of compacted observations
- Session merging/era summaries (deferred — sessions are lightweight)
- Token counting or budget-based compaction (that's F6)

## Success Criteria
- AI can identify observations older than N days in a single tool call
- AI can batch soft-delete stale observations and create a replacement summary atomically
- Total tool count goes from 33 to 34 (one new tool: `mem_compact`)
- All existing tests pass, no regressions
