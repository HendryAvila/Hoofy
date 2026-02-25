# Tool Reference

Hoofy exposes **32 MCP tools** across three systems. The AI uses them proactively based on built-in server instructions — you don't need to call them manually.

---

## Memory (17 tools)

Persistent context across sessions. SQLite + FTS5 full-text search with a knowledge graph for connecting observations.

| Tool | Description |
|---|---|
| `mem_save` | Save an observation (decision, bugfix, pattern, discovery, config, architecture) |
| `mem_save_prompt` | Record user intent for future context |
| `mem_search` | Full-text search across all sessions |
| `mem_context` | Recent observations for session startup |
| `mem_timeline` | Chronological context around a specific event |
| `mem_get_observation` | Full content of a specific observation (includes direct relations) |
| `mem_relate` | Create a typed directional relation between two observations (`relates_to`, `depends_on`, `caused_by`, `implements`, `supersedes`, `part_of`) |
| `mem_unrelate` | Remove a relation by relation ID |
| `mem_build_context` | Traverse the knowledge graph from a starting observation with configurable depth |
| `mem_session_start` | Register a new coding session |
| `mem_session_end` | Close a session with summary |
| `mem_session_summary` | Save comprehensive end-of-session summary |
| `mem_stats` | Memory system statistics |
| `mem_capture_passive` | Passive observation capture from conversation content |
| `mem_delete` | Remove an observation |
| `mem_update` | Update an existing observation |
| `mem_suggest_topic_key` | Suggest stable key for upserts (evolving knowledge) |

## Change Pipeline (6 tools)

Adaptive workflow for ongoing development. Includes `sdd_explore` for pre-pipeline context capture and `sdd_context_check` for mandatory conflict scanning.

| Tool | Description |
|---|---|
| `sdd_explore` | Pre-pipeline context capture — saves goals, constraints, tech preferences, unknowns, and decisions to memory. Upserts via topic key (call multiple times as thinking evolves). Suggests change type/size based on keywords. Use before `sdd_change` or `sdd_init_project`. |
| `sdd_change` | Create a new change (feature, fix, refactor, enhancement) with size (small, medium, large) |
| `sdd_context_check` | Mandatory conflict scanner — scans existing specs, completed changes, memory observations, and convention files (`CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, etc.) for ambiguities and conflicts. Runs as the first stage in every change flow. Zero issues = advance. Issues found = must resolve. |
| `sdd_change_advance` | Save stage content and advance to next stage |
| `sdd_change_status` | View current change status, stage progress, and artifacts |
| `sdd_adr` | Capture Architecture Decision Records (context, decision, rationale, rejected alternatives) |

## Project Pipeline (9 tools)

Full greenfield specification — from vague idea to validated architecture. Now includes business rules extraction before the Clarity Gate.

| Tool | Description |
|---|---|
| `sdd_init_project` | Initialize SDD project structure (`sdd/` directory) |
| `sdd_create_proposal` | Save structured proposal (problem, users, solution, scope, success criteria) |
| `sdd_generate_requirements` | Save formal requirements with MoSCoW prioritization |
| `sdd_create_business_rules` | Extract declarative business rules from requirements using BRG taxonomy (Definitions, Facts, Constraints, Derivations) and DDD Ubiquitous Language. Runs between requirements and the Clarity Gate. |
| `sdd_clarify` | Run the Clarity Gate (8-dimension ambiguity analysis) |
| `sdd_create_design` | Save technical architecture (components, data model, APIs, security) |
| `sdd_create_tasks` | Save implementation task breakdown with dependency graph and optional wave assignments for parallel execution |
| `sdd_validate` | Cross-artifact consistency check (requirements ↔ design ↔ tasks) |
| `sdd_get_context` | View project state, pipeline status, and stage artifacts |

## Prompts

| Prompt | Description |
|---|---|
| `/sdd-start` | Start a new SDD project (guided conversation) |
| `/sdd-status` | Check current pipeline status |
