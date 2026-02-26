# Spec: Token-Efficient Tool Responses

## Functional Requirements

### Must Have

- **FR-001**: `Store.CountObservations(project, scope string) (int, error)` — Returns total count of non-deleted observations matching filters. Used by `mem_context` and `mem_search` to report "Showing X of Y".

- **FR-002**: `Store.CountSearchResults(query string, opts SearchOptions) (int, error)` — Returns total count of FTS5 search matches (before LIMIT is applied). Used by `mem_search` to report accurate totals.

- **FR-003**: `mem_search` response MUST include a navigation footer when results are capped: `"Showing {returned} of {total} results. Use limit or mem_get_observation #ID for more."`. Only shown when returned < total.

- **FR-004**: `mem_context` response MUST include a navigation footer when observations are capped: `"Showing {returned} of {total} observations. Increase limit or use mem_get_observation #ID for details."`. Only shown when returned < total.

- **FR-005**: `mem_timeline` response MUST surface the existing `TotalInRange` field as a navigation hint: `"Showing {before+1+after} of {totalInRange} observations in session."`. Only shown when the window is smaller than the total.

- **FR-006**: `sdd_get_context` default `detail_level` MUST change from `"standard"` to `"summary"`. The tool description MUST be updated to reflect this.

- **FR-007**: Navigation hint helper `NavigationHint(showing, total int, hint string) string` in `memory` package. Returns empty string when showing >= total. Returns formatted hint when showing < total.

### Should Have

- **FR-008**: Server instructions in `serverInstructions()` SHOULD document that `sdd_get_context` now defaults to summary, and that navigation hints exist.

## Non-Functional Requirements

- **NFR-001**: Navigation hints MUST NOT add more than 1 line of text per response — minimal token overhead.
- **NFR-002**: Count queries MUST use efficient SQL (COUNT(*) with same WHERE/JOIN as the main query, no full scan).
- **NFR-003**: All changes MUST be backward-compatible — no existing tool signatures change.
- **NFR-004**: Test coverage for all new code paths (count methods, navigation hint helper, integration in tool handlers).
