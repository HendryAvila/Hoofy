# Spec: Sub-Agent Memory Scoping

## Functional Requirements

### Database Layer

- **FR-001**: Add `namespace` column to `observations` table — `TEXT DEFAULT NULL`, nullable, with index `idx_obs_namespace`. Migration uses `ALTER TABLE ADD COLUMN` with `IF NOT EXISTS`-safe pattern (check if column exists first, add if not).

- **FR-002**: `AddObservation()` accepts `Namespace string` field in `AddObservationParams`. If non-empty, stored in the `namespace` column. If empty/omitted, stored as NULL (shared pool).

- **FR-003**: All store query methods that accept `(project, scope string)` filters ALSO accept `namespace string` as an additional filter:
  - `RecentObservations(project, scope, namespace string, limit int)`
  - `FindStaleObservations(project, scope, namespace string, olderThanDays, limit int)`
  - `CountObservations(project, scope, namespace string)`
  - `CountSearchResults(query string, opts SearchOptions)` — `SearchOptions` gains `Namespace string`
  - `Search(query string, opts SearchOptions)` — already uses `SearchOptions`

- **FR-004**: Namespace filtering logic: `AND namespace = ?` when namespace is non-empty. When empty, no filter is applied (returns all namespaces including NULL). This ensures backwards compatibility — omitting namespace returns everything.

- **FR-005**: `FormatContextDetailed(project, scope, namespace string, opts ContextFormatOptions)` passes namespace through to `RecentObservations()`.

### Tool Layer — Write Tools

- **FR-006**: `mem_save` gains `namespace` string parameter — "Namespace for sub-agent isolation (e.g. 'subagent/task-123'). Omit for shared memory."

- **FR-007**: `mem_save_prompt` gains `namespace` string parameter — passed through to `AddPrompt()`. Requires `AddPromptParams` to gain `Namespace string` and `user_prompts` table to gain `namespace` column.

- **FR-008**: `mem_session_summary` gains `namespace` string parameter — passed through to `AddObservation()`.

- **FR-009**: `mem_progress` gains `namespace` string parameter — when provided, the topic_key becomes `progress/<namespace>/<project>` instead of `progress/<project>`, enabling per-sub-agent progress tracking.

### Tool Layer — Read Tools

- **FR-010**: `mem_search` gains `namespace` string parameter — passed to `SearchOptions.Namespace`.

- **FR-011**: `mem_context` gains `namespace` string parameter — passed through to `FormatContextDetailed()` and `CountObservations()`.

- **FR-012**: `mem_timeline` gains `namespace` string parameter — the timeline query filters observations by namespace if provided (both before/after queries).

- **FR-013**: `mem_compact` gains `namespace` string parameter — passed to `FindStaleObservations()` for identify mode. In execute mode, compacted observations already selected by ID so no namespace filter needed.

### Documentation

- **FR-014**: Update server instructions in `server.go` with sub-agent memory scoping workflow section explaining namespace conventions and usage patterns.

- **FR-015**: Update `docs/research-foundations.md` with F5 entry linking to Anthropic's Multi-Agent Research System article.

- **FR-016**: Update tool counts in README.md, AGENTS.md, docs/tool-reference.md — no new tools, but mem tools parameter lists need updating in tool-reference.md.

## Non-Functional Requirements

- **NFR-001**: All existing tests pass without modification — namespace omission = no filter = identical behavior.

- **NFR-002**: Schema migration is non-destructive — `ALTER TABLE ADD COLUMN` only if column doesn't exist. No data loss on upgrade.

- **NFR-003**: No new tools — this adds parameters to existing tools, keeping the tool count at 34.

- **NFR-004**: Namespace filtering uses indexed column — `CREATE INDEX idx_obs_namespace ON observations(namespace)`.
