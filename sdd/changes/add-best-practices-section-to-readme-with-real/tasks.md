# Tasks

## TASK-001: Create `docs/tool-reference.md`
**Covers**: FR-008
**Dependencies**: None
**Description**: Move the full "Available Tools" tables (Memory 14 tools, Change 5 tools, Project 8 tools, Prompts) from README.md into a new `docs/tool-reference.md`. Add a header and brief intro.
**Acceptance Criteria**:
- [ ] File exists at `docs/tool-reference.md`
- [ ] Contains all 3 tool tables + prompts table from current README
- [ ] Has a descriptive header and one-line intro per section

## TASK-002: Create `docs/workflow-guide.md` 
**Covers**: FR-003, FR-004, FR-005, FR-009
**Dependencies**: None (parallel with TASK-001)
**Description**: Create the main workflow documentation with:
1. **Greenfield Project Workflow** — Mermaid flowchart of the 7-stage pipeline + step-by-step walkthrough showing what the AI does at each stage, with example tool calls
2. **Existing Project Change Workflow** — Mermaid diagram of the change pipeline + examples for: large feature, medium fix, small refactor
3. **Memory Best Practices** — Session lifecycle (start → work → save → summary), what to save and when, search patterns
4. **Quick Decision Guide** — "I want to..." → which system to use

**Acceptance Criteria**:
- [ ] Greenfield walkthrough covers all 7 stages with example flow
- [ ] Change walkthrough shows at least 2 type×size combinations  
- [ ] Mermaid diagrams render on GitHub
- [ ] Memory section covers session lifecycle and observation types

## TASK-003: Restructure README.md
**Covers**: FR-001, FR-002, FR-006, FR-007, FR-010, NFR-001
**Dependencies**: TASK-001, TASK-002 (needs links to created docs)
**Description**: Modify README.md:
1. Add a single Mermaid diagram after "What Hoofy Does" showing both pipelines
2. Add "Best Practices" section with 5-7 concise rules
3. Add "Which system do I use?" quick decision guide
4. Condense "Memory System" to 3-4 lines + link to workflow guide
5. Replace "Available Tools" with summary count + link to `docs/tool-reference.md`
6. Condense "Change Pipeline" and "Project Pipeline" to briefer overviews + links
7. Remove "Hoofy vs Plan Mode" section (move concept into Best Practices)
8. Target: 350-400 lines total

**Acceptance Criteria**:
- [ ] README is 350-400 lines
- [ ] Mermaid diagram renders on GitHub
- [ ] All links to docs/ are valid
- [ ] Best Practices section has 5-7 rules
- [ ] No information is lost (everything moved, not deleted)

## TASK-004: Verify all links and rendering
**Covers**: NFR-002, NFR-003
**Dependencies**: TASK-003
**Description**: Manually verify:
1. All internal links between README and docs/ resolve correctly
2. Mermaid diagrams render on GitHub (push and check)
3. No broken anchors or missing sections

**Acceptance Criteria**:
- [ ] All `docs/` links work
- [ ] Mermaid renders in light mode
- [ ] No dead links

## Dependency Graph

```
TASK-001 ──┐
            ├──→ TASK-003 ──→ TASK-004
TASK-002 ──┘
```

## Estimated Effort

3-4 hours for a single session. TASK-001 and TASK-002 can run in parallel.