# Propose — F6: Context Budget Awareness

## Problem

Hoofy's read tools (`mem_context`, `mem_search`, `mem_timeline`, `sdd_get_context`, `sdd_context_check`) return variable-length responses without awareness of how much context window they're consuming. The AI has no way to ask "give me memories, but keep it under ~2000 tokens" — it must either guess with `limit` or use `detail_level` which controls verbosity but not total response size.

This leads to:
- Bloated context when many observations exist (each at 300-char standard snippets × 20 results = large response)
- The AI cannot make informed budget decisions — it doesn't know how many tokens a response consumed
- No way to "fill up to X tokens" with the highest-value content

## Research Source

- **Anthropic "Effective Context Engineering"**: "Context is a finite resource with diminishing marginal returns" — tools should be budget-aware
- **Anthropic "Multi-Agent Research System"**: "Token usage explains 80% of performance variance"
- **Anthropic "Writing Effective Tools"**: "Truncate tool responses, but always include total counts"

## Proposed Solution

Add **token estimation** and **budget-capping** to read-heavy tool responses.

### 1. Token Estimation Function
A simple, zero-dependency `EstimateTokens(text string) int` function that approximates token count using the `chars / 4` heuristic (standard for English text with GPT/Claude tokenizers). No external dependencies — just math.

### 2. `max_tokens` Parameter on Read Tools
Add an optional `max_tokens` parameter to the 5 read-heavy tools that already have `detail_level`:
- `mem_context`
- `mem_search`
- `mem_timeline`
- `sdd_get_context`
- `sdd_context_check`

When `max_tokens` is set:
- The tool builds its response incrementally
- After each item (observation, search result, etc.), it checks estimated token count
- If adding the next item would exceed the budget, it STOPS and appends a budget footer
- The footer reports: estimated tokens used, items shown vs total, and a hint

### 3. Token Count in Response Footer
ALL read-heavy tool responses (even without `max_tokens`) include an estimated token count in the response. This gives the AI visibility into how much context each tool call consumed.

Format: `📏 ~1,234 tokens` appended to the existing navigation hints.

## Scope

### In Scope
- `EstimateTokens()` function in `memory` package
- `max_tokens` parameter on 5 read tools
- Token count footer on all read-heavy responses
- Update server instructions with context budget workflow
- Update research-foundations.md with F6 entry
- Update tool-reference.md with max_tokens parameter

### Out of Scope
- Actual tokenizer integration (tiktoken, etc.) — estimation is sufficient
- Auto-adjusting detail_level based on budget — the AI decides
- Token counting for write tools — not useful
- Cross-tool budget coordination — each call is independent

## Success Criteria
- AI can request `mem_context(max_tokens=2000)` and get a response that fits within ~2000 tokens
- Every read-heavy response includes `📏 ~N tokens` footer
- All existing tests pass without modification
- Tool count stays at 34 (no new tools)
- Zero external dependencies added