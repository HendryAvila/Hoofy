# Proposal: Best Practices & Workflow Guide

## Problem

Hoofy's README explains WHAT the tools do (14 memory tools, 5 change tools, 8 project tools) but doesn't teach HOW to use them in practice. A user installs Hoofy and then... what? They see 27 tools and have no idea what a real workflow looks like.

The README is currently 477 lines with detailed tool tables and pipeline descriptions, but lacks:
- A real-world workflow showing the tools in action
- Best practices for getting maximum value
- Visual flow diagrams that make the pipeline intuitive
- Guidance on WHEN to use each system (memory vs change vs project)

## Research findings

Analysis of top developer tool READMEs (bubbletea, fzf, lazygit) shows:
- **Keep README under 400 lines** — it's a vitrine, not a manual
- **Link out to docs/** for detailed workflows and tutorials
- **Show, don't tell** — GIFs, diagrams, concise examples beat walls of text
- **Lead with the problem** — why should users care?

## Proposed solution

### 1. Restructure README (~350-400 lines)

**Cut from README** (move to docs/):
- Detailed Memory System section → `docs/memory-system.md`  
- Detailed Available Tools tables → `docs/tool-reference.md`
- Adaptive flows matrix → lives in `docs/workflow-guide.md`

**Add to README:**
- One Mermaid diagram showing the two main flows (project pipeline + change pipeline)
- "Best Practices" section with 5-7 concise rules
- Links to `docs/workflow-guide.md` for full walkthrough with examples

### 2. Create `docs/workflow-guide.md` (the main deliverable)

This is the "teaching" document with:
- **Greenfield project workflow** — step-by-step with Mermaid diagram and example dialogue
- **Existing project workflow** — feature, fix, refactor examples  
- **Memory workflow** — session lifecycle, what to save, when to search
- **Decision tree** — "I want to do X, which system do I use?"

### 3. Keep Mermaid (with restraint)

GitHub renders Mermaid natively. One clean diagram in the README, detailed ones in docs/. Simple flowcharts, not complex sequence diagrams.

## Out of scope

- GIF/terminal recordings (valuable but separate effort)
- Rewriting existing sections that work fine (Quick Start, installation)
- Translating to other languages
- Adding new Hoofy features

## Success criteria

- README drops from 477 to ~350-400 lines
- New user can understand the TWO main workflows in under 2 minutes from README
- `docs/workflow-guide.md` provides a complete real-world walkthrough
- Mermaid diagrams render correctly on GitHub (light + dark mode)