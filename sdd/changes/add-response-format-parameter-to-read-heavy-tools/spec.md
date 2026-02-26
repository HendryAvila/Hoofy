# Spec: Response Verbosity Control

## Functional Requirements

### Must Have

- **FR-001**: `mem_context` tool accepts an optional `detail_level` parameter with values `summary`, `standard` (default), `full`
- **FR-002**: `mem_context` with `detail_level: summary` returns only session project names + started_at, observation titles + IDs + types (no content snippets, no prompt content)
- **FR-003**: `mem_context` with `detail_level: standard` returns current behavior (200-char session summaries, 200-char prompts, 300-char observation content)
- **FR-004**: `mem_context` with `detail_level: full` returns complete untruncated content for all sessions, prompts, and observations
- **FR-005**: `mem_context` `limit` parameter is functional — actually controls the number of observations returned (currently dead code)
- **FR-006**: `mem_search` tool accepts an optional `detail_level` parameter with values `summary`, `standard` (default), `full`
- **FR-007**: `mem_search` with `detail_level: summary` returns only observation IDs, types, and titles (no content)
- **FR-008**: `mem_search` with `detail_level: standard` returns current behavior (300-char content snippets)
- **FR-009**: `mem_search` with `detail_level: full` returns complete untruncated content per result
- **FR-010**: `mem_timeline` tool accepts an optional `detail_level` parameter with values `summary`, `standard` (default), `full`
- **FR-011**: `mem_timeline` with `detail_level: summary` returns only titles + timestamps for before/after/focus entries
- **FR-012**: `mem_timeline` with `detail_level: standard` returns current behavior (200-char before/after, full focus)
- **FR-013**: `mem_timeline` with `detail_level: full` returns complete untruncated content for ALL entries (before, focus, and after)
- **FR-014**: `sdd_context_check` tool accepts an optional `detail_level` parameter with values `summary`, `standard` (default), `full`
- **FR-015**: `sdd_context_check` with `detail_level: summary` returns only artifact filenames + sizes, change slugs, and memory observation titles (no content excerpts)
- **FR-016**: `sdd_context_check` with `detail_level: standard` returns current behavior (500-char artifact excerpts, 200-char memory content)
- **FR-017**: `sdd_context_check` with `detail_level: full` returns complete untruncated artifact content and full memory observation content
- **FR-018**: Server instructions in `server.go` updated to guide the AI on when to use each detail level

### Should Have

- **FR-019**: A shared `detailLevel` helper function (or constant set) in a common package to avoid duplicating enum parsing across 4 tools
- **FR-020**: Each tool's response includes a footer hint when in `summary` mode: "Use detail_level: standard or full for more detail"

### Could Have

- **FR-021**: Unify the two truncation functions (`memory.Truncate` and `tools.truncateContent`) into a single shared utility

## Non-Functional Requirements

- **NFR-001**: Default behavior is `standard` for all tools — no breaking changes for existing integrations
- **NFR-002**: All modified tools must have test coverage for all three detail levels
- **NFR-003**: `summary` mode responses must be at least 50% smaller (in characters) than `standard` mode for the same data
- **NFR-004**: No new dependencies — uses only stdlib and existing packages

## Acceptance Criteria

- [ ] All 4 tools accept `detail_level` parameter with enum constraint
- [ ] `summary` mode returns no content snippets (only metadata)
- [ ] `standard` mode matches current behavior exactly
- [ ] `full` mode returns complete untruncated content
- [ ] `mem_context` `limit` parameter actually works
- [ ] Server instructions guide AI on when to use each level
- [ ] All tests pass with `go test -race ./...`
- [ ] `make lint` passes

## Source

- [Writing Effective Tools for Agents](https://www.anthropic.com/engineering/writing-tools-for-agents)
- [Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
