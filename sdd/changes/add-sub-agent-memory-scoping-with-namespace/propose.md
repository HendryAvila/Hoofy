# Proposal: Sub-Agent Memory Scoping

## Problem

When orchestrator agents (Claude Code, OpenCode, etc.) spawn multiple sub-agents in parallel, all sub-agents share the same Hoofy memory pool. This causes:

1. **Context pollution** — Sub-agent A (working on database migration) saves observations that sub-agent B (working on UI) reads, wasting tokens on irrelevant context
2. **Search noise** — `mem_search` returns mixed results from all sub-agents, reducing signal-to-noise ratio  
3. **Progress interference** — `mem_progress` tracks one active progress per project; parallel sub-agents overwrite each other's progress

## Research Foundation

Anthropic's ["How We Built Our Multi-Agent Research System"](https://www.anthropic.com/engineering/multi-agent-research-system) (Jun 2025):
- "Sub-agent architectures: specialized sub-agents with clean context windows, return condensed summaries"
- Token usage explains 80% of performance variance — noisy context from other sub-agents wastes tokens

Anthropic's ["Effective Context Engineering for AI Agents"](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) (Sep 2025):
- "Sub-agent architectures: specialized sub-agents with clean context windows"
- Context is a finite resource — loading another sub-agent's work into yours is wasteful

## Proposed Solution

Add a lightweight `namespace` parameter to memory tools that enables opt-in isolation:

1. **`mem_save`**, **`mem_save_prompt`**, **`mem_session_start`**, **`mem_session_summary`** — accept `namespace` to tag observations
2. **`mem_search`**, **`mem_context`**, **`mem_timeline`**, **`mem_compact`**, **`mem_progress`** — accept `namespace` to filter reads
3. **No namespace = no filter** — backwards compatible; unnamespaced queries see everything
4. **Namespace convention**: `subagent/<task-id>` or `agent/<role>` (e.g., `subagent/task-123`, `agent/researcher`)
5. **Cross-namespace reads** — orchestrator omits namespace to see all sub-agent work

**Key design decision**: Namespace is a convention-level filter using the existing `observations` table, NOT a hard isolation boundary. This is intentional — sub-agents should be able to read the shared pool when needed (e.g., reading project-level architecture decisions).

## Out of Scope

- Hard tenant isolation (separate databases per namespace)
- Namespace CRUD management (create/delete/list namespaces)
- Automatic namespace assignment (the AI decides which namespace to use)
- Namespace-aware `mem_compact` aggregation across namespaces
- Access control / permissions between namespaces

## Success Criteria

1. Sub-agents can save observations tagged with a namespace
2. Sub-agents can filter reads to their own namespace
3. Omitting namespace returns all observations (backwards compatible)
4. No schema changes needed (uses existing columns or adds one lightweight column)
5. All existing tests pass without modification
