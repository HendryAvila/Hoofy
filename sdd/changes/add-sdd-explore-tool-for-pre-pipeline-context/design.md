# Design: sdd_explore

## Architecture Overview

`sdd_explore` is a single MCP tool that lives in `internal/tools/explore.go`. It depends ONLY on `*memory.Store` — no config, no changes, no templates. It saves structured exploration context as memory observations with `type=explore` and auto-generated `topic_key`.

The integration with pipelines is purely through `serverInstructions()` — the AI is told to call `sdd_explore` before starting pipelines, and pipeline stages can retrieve explore context via `mem_search(type=explore)`.

```
┌──────────────────────────────────────────────┐
│                AI Agent                       │
│                                               │
│  1. User discusses idea                       │
│  2. AI calls sdd_explore(title, goals, ...)   │
│  3. AI calls sdd_init_project or sdd_change   │
│  4. AI retrieves explore context via          │
│     mem_search(type=explore) to inform        │
│     proposal/spec/design content              │
└──────────┬───────────────────────┬────────────┘
           │                       │
    ┌──────▼──────┐         ┌──────▼──────┐
    │ ExploreTool  │         │ Pipeline    │
    │ (tools pkg)  │         │ Tools       │
    │              │         │ (unchanged) │
    └──────┬──────┘         └─────────────┘
           │
    ┌──────▼──────┐
    │ memory.Store │
    │ (SQLite)     │
    └─────────────┘
```

## Component Design

### ExploreTool (`internal/tools/explore.go`)

**Struct:**
```go
type ExploreTool struct {
    store *memory.Store
}
```

**Constructor:**
```go
func NewExploreTool(store *memory.Store) *ExploreTool
```

**Dependencies:** Only `*memory.Store` — injected in `server.go` composition root. Registered in the memory tools section (since it depends on memory), but listed as an SDD tool in the MCP definition name (`sdd_explore`).

**MCP Definition:**
- Name: `sdd_explore`
- Parameters:
  - `title` (string, required): Short title for the exploration context
  - `goals` (string, optional): What the user wants to achieve
  - `constraints` (string, optional): Technical, business, or time limitations  
  - `preferences` (string, optional): Architecture style, tech stack, patterns preferred
  - `unknowns` (string, optional): Things still unclear or undecided
  - `decisions` (string, optional): Choices already made during exploration
  - `context` (string, optional): Free-form additional context
  - `project` (string, optional): Project name for filtering
  - `scope` (string, optional): Defaults to "project"
  - `session_id` (string, optional): Defaults to "manual-save"

**Validation:**
- `title` is required (non-empty)
- At least one of the optional content parameters (goals, constraints, preferences, unknowns, decisions, context) must be provided
- Return descriptive error if no content params given: "At least one context field (goals, constraints, preferences, unknowns, decisions, context) is required"

### Content Formatting

The Handle method assembles the structured markdown content from provided parameters:

```markdown
## Goals
[content of goals param]

## Constraints
[content of constraints param]

## Preferences
[content of preferences param]

## Unknowns
[content of unknowns param]

## Decisions
[content of decisions param]

## Context
[content of context param]
```

Only sections with non-empty values are included. Sections appear in the fixed order above regardless of parameter order.

### Topic Key Generation

The tool auto-generates a `topic_key` using the existing `memory.SuggestTopicKey()` function with `type="explore"`. This requires adding `"explore"` to the `inferTopicFamily()` function in `store.go` so it maps to a dedicated family.

**New family mapping in `inferTopicFamily()`:**
```
case "explore", "exploration", "context", "discuss":
    return "explore"
```

This ensures topic keys follow the pattern `explore/<slug>`, e.g., `explore/user-auth-system`.

### Upsert Behavior

Since topic_key is always provided, repeated calls with the same title will UPDATE the existing observation instead of creating a new one. The `AddObservation` method already handles this:
1. Check for existing observation with same `topic_key + project + scope`
2. If found → UPDATE (increment `revision_count`)
3. If not found → INSERT new observation

This means the AI can call `sdd_explore` multiple times as the conversation evolves, and the context accumulates without creating duplicates.

### Response Format

The tool response includes:

```
## Exploration Context Saved

**Title:** {title}
**Topic Key:** {topic_key}
**Action:** Created | Updated (revision #{n})
**ID:** {observation_id}

### Captured Context
{formatted content sections — ALL of them, not just what was added}

### Suggested Next Steps
- When ready to start a new project: use `sdd_init_project`
- When ready to modify existing code: use `sdd_change`
- To add more context: call `sdd_explore` again with the same title

### Type/Size Suggestion
{only if enough signal exists in goals/constraints/context}
- **Suggested type:** {feature|fix|refactor|enhancement} — because {reason}
- **Suggested size:** {small|medium|large} — because {reason}
```

### FR-008: Auto-suggest Type/Size

The suggestion logic is a simple keyword heuristic in a private function:

```go
func suggestChangeType(goals, constraints, context string) (string, string)
```

**Type heuristics** (first match wins, scanned across goals+constraints+context):
- Contains "fix", "bug", "crash", "error", "broken" → `fix`
- Contains "refactor", "restructure", "reorganize", "clean up" → `refactor`
- Contains "improve", "enhance", "optimize", "better", "upgrade" → `enhancement`
- Contains "new", "add", "create", "build", "implement" → `feature`
- Default: `feature`

**Size heuristics** (scanned across goals+constraints+context):
- Contains "quick", "small", "simple", "trivial", "one-liner", "minor" → `small`
- Contains "complex", "large", "major", "big", "rewrite", "overhaul" → `large`
- Default: `medium`

The suggestion is ALWAYS labeled as a suggestion, never presented as a decision. If no keywords match, the default values are used with a note like "Based on limited context — adjust as needed."

### Retrieving Full Context on Upsert

For the response to show ALL captured context (not just what was added in this call), on upsert the tool reads back the observation using `store.GetObservation(id)` (already exists) to get the full content. However, since we're REPLACING the content on upsert (not appending), the tool must:

1. On first call: Format content from parameters, save it
2. On subsequent calls: Read existing observation, MERGE new sections with existing ones (new values override, missing values preserve existing), format merged content, save it

**Merge logic:**
- For each category (goals, constraints, preferences, unknowns, decisions, context):
  - If new value is non-empty → use new value (overrides existing)
  - If new value is empty → preserve existing value from stored observation
- This requires parsing the stored markdown back into sections. Since we control the format (## headers), this is reliable.

**Implementation:** A private `parseExploreContent(markdown string) map[string]string` function extracts sections by splitting on `## ` headers. Then `mergeExploreContext(existing map[string]string, new map[string]string) map[string]string` merges them.

## Registration in server.go

The tool is registered in the memory tools section (since it depends on `*memory.Store`), inside the `if memErr == nil` block:

```go
// --- Register explore tool (SDD + Memory hybrid) ---
exploreTool := tools.NewExploreTool(memStore)
s.AddTool(exploreTool.Definition(), exploreTool.Handle)
```

Note: The tool lives in `internal/tools/` (not `internal/memtools/`) because it's an SDD-prefixed tool that happens to use memory, not a memory management tool. This follows the principle that package placement reflects purpose, not dependency.

## serverInstructions() Updates

A new section is added BEFORE the "## Pipeline" section:

```
## PRE-PIPELINE EXPLORATION

Before starting any pipeline (project or change), use sdd_explore to capture
the user's context, goals, and constraints. This ensures every subsequent
stage is informed by structured pre-work rather than ad-hoc conversation.

### When to Use sdd_explore
- Before sdd_init_project: Capture project vision, user constraints, tech preferences
- Before sdd_change: Capture change context, help determine type and size
- During any open-ended discussion about features, architecture, or direction
- When the user is "thinking out loud" and you want to preserve their reasoning

### How to Use sdd_explore
1. Discuss the idea with the user — ask clarifying questions
2. Call sdd_explore with structured categories:
   - goals: What they want to achieve
   - constraints: Limitations (technical, business, time)
   - preferences: Architecture, tech stack, patterns they prefer
   - unknowns: Things they're unsure about
   - decisions: Choices already made
   - context: Any additional context
3. The tool saves to memory with topic_key upsert — call it again as context evolves
4. When ready, start the pipeline — retrieve explore context with mem_search(type=explore)
   to inform your proposal/spec/design content

### Important
- sdd_explore is OPTIONAL — it never blocks pipeline advancement
- It uses memory, not the pipeline state machine — no stage gates
- Call it multiple times as the conversation evolves — it upserts, not duplicates
- The type/size suggestion is a HINT — the user decides
```

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `internal/tools/explore.go` | CREATE | New ExploreTool implementation |
| `internal/tools/explore_test.go` | CREATE | Tests for ExploreTool |
| `internal/memory/store.go` | MODIFY | Add `explore` family to `inferTopicFamily()` |
| `internal/memory/store_test.go` | MODIFY | Test `SuggestTopicKey` with `explore` type |
| `internal/server/server.go` | MODIFY | Register ExploreTool + update `serverInstructions()` |

## Edge Cases

1. **Empty title**: Return error "title is required"
2. **All content params empty**: Return error "At least one context field is required"
3. **Memory subsystem disabled**: Tool is not registered (same as all memory-dependent tools)
4. **Very long content in a category**: Handled by `AddObservation`'s truncation logic (already exists)
5. **Concurrent upserts with same topic_key**: SQLite serializes writes — safe
6. **Parsing stored content with unexpected format**: `parseExploreContent` should be defensive — if parsing fails, treat existing content as empty and just save new content (no crash)
