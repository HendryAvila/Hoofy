# Tasks — F6: Context Budget Awareness

## Task Breakdown (7 tasks, 3 waves)

### TASK-001: Token estimation functions
**Component**: memory package
**Covers**: FR-001, FR-002
**Dependencies**: None
**Description**: Add `EstimateTokens(text string) int` and `TokenFooter(estimatedTokens int) string` to `internal/memory/detail_level.go` (co-located with existing response formatting utilities).
**Acceptance Criteria**:
- [ ] `EstimateTokens("")` returns 0
- [ ] `EstimateTokens("hello")` returns at least 1
- [ ] `EstimateTokens` uses `len(s)/4`, no iteration
- [ ] `TokenFooter(1234)` returns `"\n📏 ~1,234 tokens"`
- [ ] `TokenFooter(500)` returns `"\n📏 ~500 tokens"`

### TASK-002: Budget-capped mem_context
**Component**: memtools/context.go, memory/store.go
**Covers**: FR-003, FR-008, FR-013
**Dependencies**: TASK-001
**Description**: Add `max_tokens` parameter to `mem_context`. Modify `FormatContextDetailed` to accept a `MaxTokens` option. When set, build response incrementally and stop when budget is reached. Always append `TokenFooter` to response.
**Acceptance Criteria**:
- [ ] `max_tokens` param in tool definition
- [ ] `ContextFormatOptions.MaxTokens int` field
- [ ] Response capped at approximate token budget
- [ ] Budget footer when capped: `"⚡ Budget: ~N/M tokens used..."`
- [ ] Token footer always appended: `"📏 ~N tokens"`

### TASK-003: Budget-capped mem_search
**Component**: memtools/search.go
**Covers**: FR-004, FR-009, FR-013
**Dependencies**: TASK-001
**Description**: Add `max_tokens` parameter to `mem_search`. Build results incrementally, check budget after each result. Always append token footer.
**Acceptance Criteria**:
- [ ] `max_tokens` param in tool definition
- [ ] Response stops adding results when budget would be exceeded
- [ ] Budget footer when capped
- [ ] Token footer always appended

### TASK-004: Budget-capped mem_timeline + SDD tools
**Component**: memtools/timeline.go, tools/context.go, tools/bridge.go
**Covers**: FR-005, FR-006, FR-007, FR-010, FR-011, FR-012, FR-013
**Dependencies**: TASK-001
**Description**: Add `max_tokens` parameter to `mem_timeline`, `sdd_get_context`, and `sdd_context_check`. Apply budget-capping and token footer to all three.
**Acceptance Criteria**:
- [ ] `max_tokens` param on all 3 tool definitions
- [ ] Budget capping works on timeline entries
- [ ] Budget capping works on SDD pipeline artifact content
- [ ] Budget capping works on context-check report sections
- [ ] Token footer appended to all 3 tools

### TASK-005: Tests — token estimation + budget capping
**Component**: memory/detail_level_test.go (new), memtools/memtools_test.go
**Covers**: NFR-001
**Dependencies**: TASK-001, TASK-002, TASK-003
**Description**: Write unit tests for `EstimateTokens` and `TokenFooter`. Write handler tests for budget-capped responses on `mem_context` and `mem_search`.
**Acceptance Criteria**:
- [ ] EstimateTokens edge cases tested (empty, short, long)
- [ ] TokenFooter formatting verified
- [ ] mem_context with max_tokens returns capped response
- [ ] mem_search with max_tokens returns capped response
- [ ] All existing tests still pass

### TASK-006: Documentation and server instructions
**Component**: server.go, docs/
**Covers**: FR-014, FR-015, FR-016
**Dependencies**: TASK-002, TASK-003, TASK-004
**Description**: Add "Context Budget Awareness" section to server instructions. Update research-foundations.md and tool-reference.md.
**Acceptance Criteria**:
- [ ] Server instructions explain max_tokens workflow
- [ ] Research foundations links F6 to Anthropic articles
- [ ] Tool reference shows max_tokens on 5 tools

### TASK-007: Final verification
**Component**: All
**Covers**: NFR-001, NFR-002, NFR-003, NFR-004, NFR-005
**Dependencies**: TASK-005, TASK-006
**Description**: Run full test suite, vet, verify tool count stays at 34.
**Acceptance Criteria**:
- [ ] `go test ./... -race -count=1` all green
- [ ] `go vet ./...` zero issues
- [ ] Tool count = 34 (no new tools)

## Execution Waves

**Wave 1** (no dependencies):
- TASK-001: Token estimation functions

**Wave 2** (parallel — depends on Wave 1):
- TASK-002: Budget-capped mem_context
- TASK-003: Budget-capped mem_search
- TASK-004: Budget-capped mem_timeline + SDD tools

**Wave 3** (depends on Wave 2):
- TASK-005: Tests
- TASK-006: Documentation
- TASK-007: Final verification (after 005 + 006)

## Estimated Effort
~2-3 hours for a single developer. Most work is in TASK-002/003/004 (incremental response building).