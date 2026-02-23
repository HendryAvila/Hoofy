---
description: >
  Start an adaptive change pipeline for a feature, fix, refactor, or enhancement.
  Use when making any code modification that deserves structured planning.
  Guides you through the appropriate pipeline stages based on change type and size.
---

# Start Adaptive Change

You are starting an adaptive change pipeline using Hoofy. This is for ongoing development — not new project bootstrapping (use `/hoofy:init` for that).

## What to do

1. Parse the user's input from `$ARGUMENTS` to understand what they want to change. Determine:
   - **Type**: `feature` (new capability), `fix` (bug fix), `refactor` (restructure), or `enhancement` (improve existing)
   - **Size**: `small` (quick, 3 stages), `medium` (moderate, 4 stages), or `large` (complex, 5-6 stages)
   - **Description**: brief description of the change

   If any of these are unclear, ask the user.

2. Call `mcp__hoofy__sdd_change` with `type`, `size`, and `description`.

3. The tool returns the pipeline stages for this change. Guide the user through the **current stage**:
   - **propose/describe/scope**: Discuss the change intent, scope, and approach
   - **spec**: Define requirements (functional and non-functional)
   - **clarify**: Answer ambiguity questions (Clarity Gate)
   - **design**: Technical architecture decisions
   - **tasks**: Break down into implementation tasks
   - **verify**: Cross-artifact validation

4. For each stage, generate the content through discussion with the user, then call `mcp__hoofy__sdd_change_advance` with the content.

5. Only ONE active change at a time. Check `mcp__hoofy__sdd_change_status` if unsure.

## Pipeline Variants

| Type × Size | Stages |
|-------------|--------|
| fix/small | describe → tasks → verify |
| fix/medium | describe → spec → tasks → verify |
| feature/medium | propose → spec → tasks → verify |
| feature/large | propose → spec → clarify → design → tasks → verify |
| refactor/small | scope → tasks → verify |

## Important

- Each stage produces a markdown artifact saved in `sdd/changes/<slug>/`
- The AI generates content, the tools SAVE it — tools are storage, not generators
- ADRs can be captured at any point with `mcp__hoofy__sdd_adr`
