# Context Check: sdd_reverse_engineer

## Artifacts Scanned

| Artifact | Status | Relevance |
|----------|--------|-----------|
| `sdd/proposal.md` | ✅ Found (3KB) | Medium — describes the adaptive change pipeline, confirms tools-are-scanners philosophy |
| `sdd/requirements.md` | ✅ Found (4.5KB) | High — FR-001 through FR-014 define change pipeline; NFRs define performance expectations |
| `sdd/design.md` | ✅ Found (21KB) | High — parallel pipeline architecture, SRP file structure, composition root wiring pattern |
| `sdd/business-rules.md` | ❌ Not found | N/A — ironically, Hoofy's own project doesn't have business rules documented |

## Prior Changes Analyzed

7 completed changes found. Most relevant:
- **add-sdd-explore-tool**: Established pattern for hybrid SDD+Memory tools (explore writes to memory, not to sdd/ files)
- **add-response-format-parameter**: Established `detail_level` parameter pattern across read-heavy tools
- **add-mem-progress-tool**: Shows pattern for tools that read/write independently of pipeline state

## Conflict Analysis

### No Conflicts Detected

1. **Pipeline independence**: The reverse-engineer tool operates OUTSIDE both pipelines (project and change). It writes artifacts to `sdd/` but doesn't advance any pipeline state machine. This is consistent with how `sdd_explore` works — it writes to memory without touching pipeline state.

2. **Existing tool compatibility**: The 3 artifacts it generates (`business-rules.md`, `design.md`, `requirements.md`) are the SAME files that `sdd_create_business_rules`, `sdd_create_design`, and `sdd_generate_requirements` write. `context-check` already knows how to read them.

3. **No FR/NFR violations**: No existing requirement prohibits writing artifacts outside the pipeline flow. The pipeline stage guards exist on the PROJECT pipeline tools (they check `pipeline.RequireStage`), but the reverse-engineer tool would write directly via `writeStageFile` — bypassing stage validation intentionally.

## Ambiguity Smells Detected (IEEE 29148)

1. **"Key files"** — The proposal says the scanner reads "key files" from the project. Which files are "key"? This needs an explicit, bounded list per language ecosystem. → **Resolved in proposal**: explicit list provided (directory structure, manifests, configs, entry points, ADRs, schemas, API specs).

2. **"Structured report"** — What format? What sections? The AI needs a predictable structure to analyze. → **Needs spec**: Define the exact report sections and format.

3. **"Auto-trigger in sdd_change"** — The proposal describes this but the mechanism is unclear. Does `sdd_change` check for artifacts and return a different response? Or does it call the scanner internally? → **Needs design decision**.

## Impact Classification

- **Non-breaking**: Adds new tool and new behavior to `sdd_change` (early return with guidance when no artifacts exist). No existing behavior changes. Existing tests unaffected.

## Recommendations

1. Proceed with spec stage — resolve the ambiguities about report format and auto-trigger mechanism
2. The scanner should follow the `detail_level` pattern established by other read-heavy tools
3. Consider adding `max_tokens` support for large project scans
4. The auto-trigger in `sdd_change` should be a SIGNAL (message in response), not an automatic action — let the AI decide whether to run the scanner
