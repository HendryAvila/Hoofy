# Tasks: Wave Orchestration Instructions

## TASK-001: Write wave execution orchestration instructions

**Component**: server.go (instructions block)
**Dependencies**: None
**Description**: Add a new section "### Wave Execution — Multi-Agent Orchestration" after the existing "Wave Assignments in Tasks Stage" block (line 801). The section must cover:

1. **When to orchestrate** — after `sdd_create_tasks` (project) or tasks stage (change pipeline) produces wave assignments, and the user requests implementation
2. **Capability detection** — describe three tiers generically:
   - Tier 1: Agent teams / shared task lists with inter-agent communication
   - Tier 2: Sub-agents / parallel workers that report back to caller
   - Tier 3: Sequential execution (single agent, no parallelization)
3. **Execution strategy per tier**:
   - Tier 1: Create a team, assign wave tasks as dependent tasks in the shared list, let agents self-coordinate
   - Tier 2: Launch one sub-agent per task in the wave, wait for all to complete, then start next wave
   - Tier 3: Execute tasks sequentially following dependency graph order
4. **Dependency enforcement** — NEVER start a task whose dependencies haven't completed
5. **File conflict prevention** — warn that tasks in the same wave should not modify the same files
6. **Memory isolation** — reference namespace parameter for parallel agents (each agent uses its own namespace)
7. **Progress tracking** — reference mem_progress for tracking wave completion state
8. **Ask the user** — if unsure which tier is available, ask the user before proceeding

**Acceptance Criteria**:
- [ ] New section added after line 801 in server.go
- [ ] Instructions are client-agnostic (no mention of Claude Code, Cursor, Codex, etc.)
- [ ] All three tiers described with clear strategy
- [ ] References namespace and mem_progress
- [ ] Instructions read naturally as guidance, not as rigid rules
- [ ] `go build ./...` passes (no syntax errors in the string)

## Execution Waves

**Wave 1**: TASK-001 (single task, no parallelization needed)

## Estimated Effort

15-20 minutes — it's writing ~30-40 lines of instructional text in a Go string literal.
