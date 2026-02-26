# Proposal: Response Verbosity Control (`detail_level` parameter)

## Problem Statement

Hoofy's read-heavy tools (`mem_context`, `mem_search`, `mem_timeline`, `sdd_context_check`) always return the same amount of data regardless of the AI's intent. When the AI just needs to know "what topics exist in memory" it gets the same 300-char content snippets as when it needs deep analysis. This wastes tokens — the scarcest resource in long-running agent sessions.

Anthropic's ["Writing Effective Tools for Agents"](https://www.anthropic.com/engineering/writing-tools-for-agents) explicitly recommends adding a `response_format` enum (concise vs detailed) to control verbosity. Their ["Effective Context Engineering"](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) article establishes that context is a finite resource with diminishing marginal returns.

## Proposed Change

Add a `detail_level` parameter (matching the existing pattern in `sdd_get_context`) to all read-heavy memory and pipeline tools. Three levels:

- **`summary`** — Minimal tokens. Titles, IDs, status only. For routing decisions.
- **`standard`** (default) — Current behavior. Truncated content snippets. For general use.
- **`full`** — Complete untruncated content. For deep analysis.

### Tools to modify

**Memory tools (Tier 1 — highest token savings):**
- `mem_context` — summary: session names + observation titles only; standard: current behavior; full: complete content
- `mem_search` — summary: titles + IDs only; standard: current 300-char snippets; full: complete content per result
- `mem_timeline` — summary: titles + timestamps only; standard: current (200-char before/after, full focus); full: all entries untruncated

**Pipeline tools (Tier 2):**
- `sdd_context_check` — summary: artifact names + change slugs only; standard: current truncated excerpts; full: complete artifact content

### Bug fix included
- Fix `mem_context` dead `limit` parameter — currently defined in schema but never read in Handle()

### NOT in scope
- `mem_get_observation` — intentionally always returns full content (that's its purpose)
- Write tools (propose, specify, etc.) — echo elimination is a separate feature
- `sdd_get_context` — already has `detail_level` implemented

## Impact

- **Token savings**: Estimated 50-70% reduction when using `summary` mode vs current `standard`
- **No breaking changes**: Default remains `standard` (current behavior)
- **Backward compatible**: Existing integrations unaffected
- **Server instructions update**: Guide the AI to use `summary` for exploration, `standard` for general work, `full` for deep analysis

## Source

- [Writing Effective Tools for Agents](https://www.anthropic.com/engineering/writing-tools-for-agents) — "response_format enum parameter"
- [Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — "context is a finite resource"
