# Spec: Best Practices & Workflow Guide

## Requirements

### Must Have

- **FR-001**: README must include a visual Mermaid diagram showing both pipelines (project + change) in a single view
- **FR-002**: README must include a "Best Practices" section with concise, actionable rules (max 7 rules)
- **FR-003**: Create `docs/workflow-guide.md` with a complete greenfield project walkthrough (step-by-step with example AI ↔ Hoofy dialogue)
- **FR-004**: Create `docs/workflow-guide.md` with a complete existing project change walkthrough (feature + fix examples)
- **FR-005**: `docs/workflow-guide.md` must include Mermaid diagrams for each flow (project pipeline, change pipeline type×size matrix)
- **FR-006**: README "Memory System" section must be condensed to a brief overview linking to detailed docs
- **FR-007**: README "Available Tools" section must be condensed to a summary count + link to detailed reference
- **FR-008**: Create `docs/tool-reference.md` with the full tool tables currently in README

### Should Have

- **FR-009**: `docs/workflow-guide.md` should include a memory best practices section (session lifecycle, what to save, search patterns)
- **FR-010**: README should include a "decision tree" or quick guide: "I want to do X → use Y"
- **FR-011**: Mermaid diagrams should work in both GitHub light and dark mode

### Non-Functional

- **NFR-001**: README total length must be 350-400 lines (currently 477)
- **NFR-002**: All Mermaid diagrams must render on GitHub web (verified manually)
- **NFR-003**: No broken internal links between README and docs/
- **NFR-004**: Existing Quick Start, installation, and research sections remain untouched

## Files affected

| File | Action | Description |
|------|--------|-------------|
| `README.md` | Modify | Restructure, add Best Practices + Mermaid, condense Memory/Tools sections |
| `docs/workflow-guide.md` | Create | Full workflow walkthrough with examples and diagrams |
| `docs/tool-reference.md` | Create | Complete tool tables moved from README |

## Constraints

- No new Hoofy features — documentation only
- Mermaid only (no external image dependencies)
- Must preserve all existing information (moved, not deleted)