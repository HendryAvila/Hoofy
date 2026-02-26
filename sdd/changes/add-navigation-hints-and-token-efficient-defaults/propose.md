# Proposal: Token-Efficient Tool Responses

## Problem

When tool responses hit internal limits (MaxSearchResults=20, MaxContextResults=20), the AI has no way to know there are MORE results available. The response just... ends. No hint, no count, no breadcrumb. This wastes AI reasoning cycles trying to figure out if the results are complete.

Additionally, `sdd_get_context` defaults to `standard` detail level, which returns full pipeline tables and artifact sizes when often just a quick status check is needed. This burns tokens unnecessarily on repeat calls.

## Source

- [Anthropic: Writing Effective Tools for Agents](https://www.anthropic.com/engineering/effective-tools)
- [Anthropic: Effective Context Engineering](https://www.anthropic.com/engineering/context-engineering)

Key quote: "Include helpful navigation hints when results are truncated — tell the agent how many total results exist and how to retrieve more."

## Scope

### In Scope
1. **Navigation hints on capped results** — When `mem_search`, `mem_context`, or `mem_timeline` results are capped by limits, append a "Showing X of Y" footer with guidance on how to fetch more (e.g., increase limit, use offset, drill down with `mem_get_observation`)
2. **Total count queries** — Add `CountObservations()` and `CountSearchResults()` methods to the Store so tools can report "Showing 10 of 47 results"
3. **Default detail_level change for sdd_get_context** — Change default from `standard` to `summary` to reduce token usage on the most frequently called SDD tool
4. **Search total count** — `mem_search` should report "Showing N of M total matches" when results are capped

### Out of Scope
- Pagination with offset/cursor (future F6 territory)
- Configurable token budgets (future F6)
- Response compression or encoding changes

## Success Criteria
- All capped responses include "Showing X of Y" navigation hints
- `sdd_get_context` defaults to summary mode
- No breaking changes to existing tool interfaces (all changes are additive)
- All existing tests continue to pass
- New tests cover the navigation hint formatting
