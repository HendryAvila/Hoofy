# Tasks: Interactive Documentation Site

## Total Tasks: 5
## Estimated Effort: 1 session

### TASK-001: Create site structure with HTML skeleton, CSS theme, and responsive layout
**Covers**: FR-001, FR-011, FR-012, FR-013, NFR-001, NFR-002, NFR-004, NFR-005, NFR-006
**Dependencies**: None
**Description**: Create `/site/index.html` with the full HTML skeleton — hero section, skill tree container, modal overlay, CSS custom properties (dark theme), responsive media queries, base animations (pulse, slideUp, unlock). No JS yet, no content yet — just the visual shell.
**Acceptance Criteria**:
- [ ] File exists at `/site/index.html`
- [ ] Dark theme with Hoofy color palette renders correctly
- [ ] Hero section with title, tagline, progress bar placeholder
- [ ] Skill tree container with zigzag layout
- [ ] Modal structure hidden by default
- [ ] Responsive breakpoint at 768px
- [ ] All CSS is inline in `<style>` tag
- [ ] File under 50KB at this stage

### TASK-002: Implement JS state management, localStorage, and node interaction logic
**Covers**: FR-004, FR-005, FR-007, FR-008, FR-009, FR-010, FR-015, FR-020
**Dependencies**: TASK-001
**Description**: Add the `<script>` section with: NODES data array (empty content for now), state management (completedIds, currentNodeId), localStorage read/write, render() function that updates node classes, openNode/closeModal/completeNode functions, progress bar logic, reset progress, scroll-to-next behavior. Keyboard nav if time permits.
**Acceptance Criteria**:
- [ ] Nodes render with correct states (locked/unlocked/current/completed)
- [ ] Clicking unlocked/current node opens modal
- [ ] Clicking "Got it!" marks complete and unlocks next
- [ ] Progress persists across page reload (localStorage)
- [ ] Progress bar updates on completion
- [ ] Reset button clears all progress
- [ ] Locked nodes are not clickable

### TASK-003: Write all 14 node content (summary + deep dive)
**Covers**: FR-002, FR-006, FR-018
**Dependencies**: TASK-002
**Description**: Write the actual content for all 14 nodes — each with icon, title, summary (2-3 sentences), and details object (what/how/why). Content sourced from `docs/research-foundations.md`, `docs/tool-reference.md`, `AGENTS.md`, and server instructions. This is the documentation meat.
**Acceptance Criteria**:
- [ ] All 14 nodes have accurate, complete content
- [ ] Summaries are bite-sized (2-3 sentences max)
- [ ] "Learn More" sections have what/how/why structure
- [ ] Track labels appear between groups
- [ ] Content matches current Hoofy capabilities (34 tools, 6 features)

### TASK-004: Add animations, confetti, mascot milestones, and polish
**Covers**: FR-014, FR-016, FR-017, FR-019, NFR-003
**Dependencies**: TASK-003
**Description**: Add the "alive" factor — confetti canvas on all-complete, sparkle on node completion, Hoofy mascot messages at 25/50/75/100%, smooth transitions on all interactions, SVG path connectors with draw animation. Cross-browser testing.
**Acceptance Criteria**:
- [ ] Confetti fires on completing all 14 nodes
- [ ] Sparkle/particle effect on individual node completion
- [ ] Hoofy mascot appears at milestone percentages
- [ ] Smooth scroll to next node after completion
- [ ] Path connectors are visible between nodes
- [ ] Animations work on Chrome, Firefox, Safari, Edge

### TASK-005: Verify, commit, push, configure GitHub Pages
**Covers**: NFR-001, NFR-002, NFR-006
**Dependencies**: TASK-004
**Description**: Final verification — file size check (<200KB), load time, mobile test, localStorage persistence, all 14 nodes navigable. Commit and push. Note about GitHub Pages configuration (user configures repo settings to serve from /site on main).
**Acceptance Criteria**:
- [ ] File size under 200KB
- [ ] All 14 nodes work end-to-end
- [ ] Mobile responsive verified
- [ ] Committed and pushed to main
- [ ] README or comment about GitHub Pages setup

## Execution Waves

**Wave 1**: TASK-001 (HTML/CSS skeleton)
**Wave 2**: TASK-002 (JS logic) — depends on Wave 1
**Wave 3**: TASK-003 (content) — depends on Wave 2
**Wave 4**: TASK-004 (polish) — depends on Wave 3
**Wave 5**: TASK-005 (verify + ship) — depends on Wave 4

All sequential — each builds on the previous.
