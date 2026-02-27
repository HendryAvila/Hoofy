# Clarify: sdd_reverse_engineer

## Questions Asked & Answers Received

### Q1: Auto-trigger behavior — Block vs. Warning?
**Answer**: **Option B** — Block only medium/large changes when no SDD artifacts exist; let small changes through. BUT include a warning in the response for ALL change sizes when artifacts don't exist.

**Updated FR-013**: When `sdd_change` is called and no SDD artifacts exist:
- **Small changes**: Create the change normally, but include a prominent warning: "⚠️ No SDD artifacts found. Consider running `sdd_reverse_engineer` to scan your project and generate baseline specs. Context-check will have limited information."
- **Medium/Large changes**: Block the change creation and return: "❌ No SDD artifacts found. Medium/large changes require project context. Run `sdd_reverse_engineer` first to scan your project and generate baseline SDD artifacts (`business-rules.md`, `design.md`, `requirements.md`)."

### Q2: Who writes the artifacts after the scan?
**Answer**: **Option A** (use existing pipeline tools) — but with a critical architectural refinement discovered during verification:

**Problem found**: The 3 pipeline tools (`sdd_generate_requirements`, `sdd_create_business_rules`, `sdd_create_design`) have stage guards (`pipeline.RequireStage`) that reject calls when no pipeline is active. They also depend on `config.Store.Load()` which fails without `sdd.json`.

**Resolution**: **Option A2** — Extract the rendering/writing logic from the 3 tools into shared functions. Both the pipeline tools and the reverse engineer tool call the same shared functions. Pipeline tools continue to enforce stage guards before calling the shared function. The reverse engineer flow calls the shared functions directly (no stage validation needed).

**New requirement (FR-022)**: Extract artifact rendering+writing from `sdd_generate_requirements`, `sdd_create_business_rules`, and `sdd_create_design` into shared functions (e.g., `RenderAndWriteRequirements`, `RenderAndWriteBusinessRules`, `RenderAndWriteDesign`) in a shared package or in `helpers.go`. These functions take content parameters + project root, render via templates, and write to `sdd/`. The existing pipeline tools become thin wrappers: validate stage → call shared function → advance pipeline.

### Q3: Existing SDD artifacts — Overwrite or skip?
**Answer**: **Only regenerate the missing ones**. If a project already has `design.md` and `requirements.md` but is missing `business-rules.md`, only generate `business-rules.md`. Existing artifacts are respected — they may have been human-curated or refined.

**Updated FR-016**: After the AI analyzes the scan report, it MUST check which of the 3 artifacts (`business-rules.md`, `design.md`, `requirements.md`) already exist in `sdd/`. Only generate the missing ones. The scan report itself always includes a section listing which artifacts already exist vs. which need generation.

## Ambiguities Resolved

| Ambiguity | Resolution | Impact |
|-----------|------------|--------|
| Auto-trigger behavior | Block medium/large, warn small | Updated FR-013 |
| Who writes artifacts | Shared rendering functions (A2 pattern) | New FR-022 |
| Existing artifact handling | Only regenerate missing | Updated FR-016 |
| Stage guard bypass | Extraction over bypass — no conditional logic in pipeline tools | Architecture decision |

## New Requirements Added

- **FR-022**: Extract artifact rendering+writing into shared functions callable by both pipeline tools and reverse engineer flow. Pipeline tools become thin wrappers (validate → render → advance).

## Clarity Assessment

All 3 ambiguities from the context-check are now resolved. The spec is clear enough to proceed to design:
- Scanner scope: bounded, explicit file lists per category ✅
- Report format: structured markdown with 9 sections ✅
- Auto-trigger: block medium/large, warn small ✅
- Artifact writing: shared rendering functions ✅
- Artifact overwrite policy: only missing ones ✅
- Token budgeting: detail_level + max_tokens ✅
