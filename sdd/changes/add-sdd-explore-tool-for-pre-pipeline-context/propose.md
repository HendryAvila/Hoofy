# Proposal: sdd_explore — Pre-pipeline Context Capture

## Problem Statement

When users start a new project (`sdd_init_project`) or open a change (`sdd_change`), there's no structured way to capture the *thinking* that happens BEFORE the pipeline begins. Users discuss ideas, constraints, preferences, and context with the AI — but none of this is persisted or structured. The proposal/describe stage then starts from scratch, forcing the AI to re-ask questions or the user to repeat themselves.

This is the same gap that GSD's "discuss phase" addresses: capturing user intent, constraints, and context BEFORE planning begins, so that every subsequent pipeline stage is informed by this pre-work.

**Current gaps:**
- Between conversation start and `sdd_init_project`, nothing is saved
- Between conversation and `sdd_change`, the user must already know type/size — no mechanism to help figure this out
- `mem_save_prompt` exists but is raw text with no structured categories
- `mem_capture_passive` is post-hoc extraction, not intentional capture

## Proposed Solution (Hybrid Approach)

A **standalone `sdd_explore` tool** that captures structured pre-pipeline context, with **soft integration** into existing pipelines via `serverInstructions()`.

### How it works:

1. **Standalone tool**: `sdd_explore` can be called anytime, independent of any pipeline. It saves structured context (goals, constraints, preferences, unknowns, decisions) to Hoofy's memory system with a specific observation type (`explore`) and topic key.

2. **AI guidance integration**: `serverInstructions()` is updated to instruct the AI to call `sdd_explore` BEFORE starting `sdd_init_project` or `sdd_change`. This is a soft recommendation, not a hard gate — the pipeline state machines remain untouched.

3. **Context availability**: Pipeline tools (`sdd_create_proposal`, `sdd_change_advance`) can read explore context from memory (via `mem_search` with type filter) to inform their generated content.

### What it captures (structured categories):
- **Goals**: What the user wants to achieve
- **Constraints**: Technical, business, or time limitations
- **Preferences**: Architectural style, tech stack opinions, patterns preferred
- **Unknowns**: Things the user isn't sure about yet
- **Decisions**: Choices already made during exploration

## Scope

### In scope:
- New `sdd_explore` tool (standalone, no pipeline dependency)
- New observation type `explore` in memory system
- `serverInstructions()` updates for AI guidance
- Structured categories for context capture
- Topic key upsert support (explore context evolves over conversation)

### Out of scope:
- NO changes to project pipeline state machine
- NO changes to change pipeline state machine
- NO new pipeline stages
- NO mandatory gates or blocks
- NO template system needed (raw markdown via memory)
- NO UI/CLI changes

## Success Criteria

1. User can call `sdd_explore` at any point to capture structured context
2. AI is guided (via server instructions) to use `sdd_explore` before starting pipelines
3. Explore context is searchable via `mem_search` with type filter
4. Pipeline stage content (proposals, specs) is noticeably better-informed when explore context exists
5. Zero impact on existing pipeline flows — all current tests pass unchanged
6. Topic key upserts allow explore context to evolve during a conversation without creating duplicates

## Open Questions

- Should `sdd_explore` accept all categories at once, or have separate parameters for each category?
- Should there be a `sdd_get_explore_context` convenience tool, or is `mem_search(type=explore)` sufficient?
- Should explore context be auto-linked (via relations) to the change/project it informs?
