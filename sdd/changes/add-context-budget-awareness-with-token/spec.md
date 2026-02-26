# Spec — F6: Context Budget Awareness

## Functional Requirements

### Token Estimation

- **FR-001**: Add `EstimateTokens(text string) int` function in `memory` package. Uses `len(text) / 4` heuristic (standard approximation for GPT/Claude tokenizers). Returns at least 1 for non-empty strings.
- **FR-002**: Add `TokenFooter(estimatedTokens int) string` function in `memory` package. Returns `"\n📏 ~N tokens"` formatted string. Uses comma-separated numbers for readability (e.g., `~1,234 tokens`).

### Budget Parameter on Read Tools

- **FR-003**: Add `max_tokens` parameter (integer, optional) to `mem_context` tool definition. Description: "Approximate token budget for the response. When set, the tool stops adding content when estimated tokens would exceed this budget."
- **FR-004**: Add `max_tokens` parameter to `mem_search` tool definition.
- **FR-005**: Add `max_tokens` parameter to `mem_timeline` tool definition.
- **FR-006**: Add `max_tokens` parameter to `sdd_get_context` tool definition.
- **FR-007**: Add `max_tokens` parameter to `sdd_context_check` tool definition.

### Budget-Capped Response Behavior

- **FR-008**: When `max_tokens` is set on `mem_context`, the tool builds the response section by section (sessions, prompts, observations). After adding each observation, it checks `EstimateTokens(response_so_far)`. If the next observation would push estimated tokens beyond `max_tokens`, it stops adding observations and appends a budget footer: `"⚡ Budget: ~N/M tokens used. X of Y observations shown. Increase max_tokens or use detail_level=summary for more."`.
- **FR-009**: When `max_tokens` is set on `mem_search`, the tool builds search results one at a time. After each result, it checks token estimate. Stops when budget would be exceeded, appends budget footer.
- **FR-010**: When `max_tokens` is set on `mem_timeline`, same incremental-build + budget-check behavior.
- **FR-011**: When `max_tokens` is set on `sdd_get_context`, applies budget to artifact content sections.
- **FR-012**: When `max_tokens` is set on `sdd_context_check`, applies budget to the scan report.

### Token Count Footer

- **FR-013**: ALL 5 read-heavy tools append `TokenFooter(EstimateTokens(response))` to their final response, regardless of whether `max_tokens` was set. This gives the AI visibility into response cost.

### Documentation

- **FR-014**: Update server instructions in `server.go` with a "Context Budget Awareness" section explaining `max_tokens`, token estimation, and recommended workflow.
- **FR-015**: Update `docs/research-foundations.md` with F6 entry under "Effective Context Engineering" and "Writing Effective Tools".
- **FR-016**: Update `docs/tool-reference.md` to mention `max_tokens` parameter on the 5 affected tools.

## Non-Functional Requirements

- **NFR-001**: All existing tests pass without modification.
- **NFR-002**: No new tools — tool count stays at 34.
- **NFR-003**: No external dependencies added (token estimation is pure math).
- **NFR-004**: `EstimateTokens` must be O(1) — just `len(s)/4`, no iteration.
- **NFR-005**: Token footer adds minimal overhead — single `len()` call + string format.