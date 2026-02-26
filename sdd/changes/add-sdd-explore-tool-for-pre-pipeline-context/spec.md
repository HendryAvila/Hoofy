# Specification: sdd_explore

## Functional Requirements

### Must Have

- **FR-001**: `sdd_explore` tool accepts structured context with these parameters:
  - `title` (required): Short, searchable title for the exploration session
  - `goals` (optional): What the user wants to achieve
  - `constraints` (optional): Technical, business, or time limitations
  - `preferences` (optional): Architecture style, tech stack opinions, patterns
  - `unknowns` (optional): Things still unclear or undecided
  - `decisions` (optional): Choices already made during exploration
  - `context` (optional): Free-form additional context that doesn't fit other categories
  - At least one optional parameter must be provided

- **FR-002**: Tool saves exploration context as a memory observation with:
  - `type` = `"explore"`
  - `topic_key` = derived from title (using `mem_suggest_topic_key` logic)
  - `scope` = `"project"` (default, overridable)
  - Content formatted as structured markdown with category headers

- **FR-003**: Topic key upsert behavior — calling `sdd_explore` multiple times with the same title updates the existing observation instead of creating duplicates (leveraging existing `topic_key` upsert mechanism in memory store)

- **FR-004**: `serverInstructions()` updated with a new section instructing the AI to:
  - Call `sdd_explore` before `sdd_init_project` to capture project vision and constraints
  - Call `sdd_explore` before `sdd_change` to capture change context and help determine type/size
  - Use explore context (retrieved via `mem_search`) to inform proposal/spec/design content

- **FR-005**: Tool returns a structured response including:
  - Confirmation of save (observation ID)
  - Whether it was a create or update (upsert)
  - The topic key used
  - A hint about next steps (e.g., "Context captured. When ready, use `sdd_init_project` or `sdd_change` to start the pipeline.")

### Should Have

- **FR-006**: `project` parameter (optional) — associates the explore context with a specific project name for filtering. Defaults to auto-detected project name if in an SDD project directory.

- **FR-007**: Tool response includes a summary of ALL explore context for the current topic key, not just what was added — so the user sees the accumulated context.

### Could Have

- **FR-008**: Auto-suggest change type and size based on explore context content (e.g., if goals mention "fix", suggest `type=fix`; if constraints mention "quick", suggest `size=small`)

### Won't Have

- **WH-001**: No mandatory gate — explore is always optional, never blocks pipeline advancement
- **WH-002**: No auto-linking to changes/projects via relations (can be done manually with `mem_relate`)
- **WH-003**: No separate `sdd_get_explore_context` tool — `mem_search(type=explore)` is sufficient
- **WH-004**: No template system — content is raw markdown saved directly to memory

## Non-Functional Requirements

- **NFR-001**: Tool execution must complete in under 100ms (it's a memory write, not a computation)
- **NFR-002**: Zero impact on existing pipeline tests — no state machine changes
- **NFR-003**: Follows existing tool patterns: struct + `NewExploreTool(store)` + `Definition()` + `Handle()`
- **NFR-004**: Memory observation content must be human-readable when viewed via `mem_get_observation`
- **NFR-005**: Backwards compatible — existing memory databases work without migration (explore is just a new observation type value, not a schema change)

## Assumptions

- The existing `topic_key` upsert mechanism in `memory.Store` works correctly for the `explore` type (no type-specific upsert logic needed)
- `serverInstructions()` is the right place for AI guidance (the AI reads and follows these instructions)
- Users will interact with explore context primarily through the AI's summarization, not by reading raw observations

## Dependencies

- `memory.Store` — for saving/updating observations with topic key upserts
- `mem_suggest_topic_key` logic — for deriving stable topic keys from titles
- `serverInstructions()` in `server.go` — for AI guidance updates

## Constraints

- Must be a single Go file (`internal/tools/explore.go`) following existing patterns
- No new database tables or migrations required
- No CGO — must work with `CGO_ENABLED=0`
- Registration in `server.go` must follow existing composition root pattern
