# Wave Orchestration Instructions for Multi-Agent Execution

## Intent

Make wave assignments from `sdd_create_tasks` actionable by adding server instructions that guide AI clients on HOW to execute tasks using their available parallelization capabilities (Agent Teams, sub-agents, or sequential fallback).

## Problem

Today, wave assignments in the task breakdown are **passive documentation** — they describe parallelization groups but don't tell the AI what to do with them. The AI has to figure out on its own whether to parallelize, how to respect dependencies, and what fallback to use. This leads to inconsistent execution across clients.

## Proposed Change

Add a new section to the server instructions in `server.go` that covers:

1. **Wave execution strategy** — how to interpret and execute wave assignments
2. **Capability-aware adaptation** — detect what the client can do (agent teams, sub-agents, sequential) and adapt
3. **Dependency enforcement** — never start Wave N+1 before Wave N completes
4. **File conflict prevention** — tasks in the same wave should not touch the same files
5. **Fallback strategy** — sequential execution respecting dependency graph when parallelization is unavailable

## Scope

- **In scope**: Server instructions in `server.go` only
- **Out of scope**: New tools, output format changes, client-specific integrations

## Client Agnosticism

The instructions must NOT reference specific clients (Claude Code, Cursor, Codex). They should describe capabilities generically:
- "If you can launch parallel agents..." (not "If you're Claude Code with Agent Teams...")
- "If you can create a shared task list..." (not "If you have CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS...")
- "If parallelization is not available..." (fallback)

## Success Criteria

- AI clients with agent team capabilities can read the waves and create parallel execution plans
- AI clients with sub-agent capabilities can delegate tasks by wave
- AI clients without parallelization can still execute tasks in correct dependency order
- No changes to tools, output format, or binary behavior
