# Spec: Interactive Documentation Site

## Functional Requirements

### Must Have

- **FR-001**: Single HTML file (`site/index.html`) with inline CSS and JS — zero external dependencies, zero build step
- **FR-002**: Skill tree with 14 nodes organized in 5 tracks (Foundation, SDD Pipeline, Adaptive Changes, Context Engineering, Boss Level)
- **FR-003**: Each node renders as a circle on a vertical winding path with SVG/CSS connectors between nodes
- **FR-004**: Node states — locked (gray, lock icon), unlocked (glowing border, subtle bounce animation), completed (checkmark, solid glow), current (pulsing to invite click)
- **FR-005**: Clicking an unlocked node opens a modal/overlay card with: icon, title, bite-sized explanation (2-3 sentences), "Learn More" expandable section, "Got it!" completion button
- **FR-006**: "Learn More" section expands inline with deeper content: feature explanation, real usage example, and when/why to use it
- **FR-007**: Completing a node (clicking "Got it!") unlocks the next node(s) in the path with an unlock animation
- **FR-008**: Progress persisted in localStorage — completed nodes survive browser close/reopen
- **FR-009**: Progress bar at top showing "X / 14 completed" with visual fill
- **FR-010**: "Reset Progress" button to start over (clears localStorage)
- **FR-011**: Dark theme with Hoofy branding — dark background (#0d1117), electric blue (#58a6ff) and purple (#bc8cff) accents
- **FR-012**: Responsive design — works on desktop (path centered) and mobile (path full-width, cards stack)
- **FR-013**: Hero section at top with Hoofy title, tagline, and brief intro before the skill tree
- **FR-014**: Completion celebration when all 14 nodes are done — confetti/particle animation + congratulations message

### Should Have

- **FR-015**: Smooth scroll to the next unlocked node after completing one
- **FR-016**: Subtle particle/sparkle effect on node completion
- **FR-017**: Nodes alternate left-right on the path (zigzag like Duolingo)
- **FR-018**: Track headers/labels between node groups (e.g., "🧠 Foundation", "📋 SDD Pipeline")
- **FR-019**: Floating Hoofy mascot emoji (🐴) that appears with encouraging messages at milestones (25%, 50%, 75%, 100%)

### Could Have

- **FR-020**: Keyboard navigation (arrow keys to move between nodes, Enter to open)
- **FR-021**: Sound effects on completion (subtle, optional, off by default)

### Won't Have

- No backend or API
- No user accounts
- No search (14 nodes is navigable)
- No multi-language
- No code playground
- No tool count or version display that would need updating

## Non-Functional Requirements

- **NFR-001**: Page load under 2 seconds (single file, no external requests)
- **NFR-002**: Total file size under 200KB (HTML + inline CSS + inline JS)
- **NFR-003**: Works in modern browsers (Chrome, Firefox, Safari, Edge — last 2 versions)
- **NFR-004**: No external CDN dependencies — everything inline or CSS-only
- **NFR-005**: Accessible — semantic HTML, ARIA labels on interactive elements, sufficient color contrast
- **NFR-006**: GitHub Pages compatible — static files only, no server-side rendering

## Content Outline (14 Nodes)

### Track 1 — 🧠 Foundation
1. **What is Hoofy?** — MCP server, AI development companion, the problem it solves
2. **Persistent Memory** — SQLite + FTS5, observations, sessions, search
3. **Knowledge Graph** — Relations, traversal, connected decisions

### Track 2 — 📋 SDD Pipeline
4. **Spec-Driven Development** — The 8-stage pipeline, why specs before code
5. **Clarity Gate** — 8-dimension ambiguity analysis, the core innovation
6. **Business Rules** — BRG taxonomy, ubiquitous language, declarative rules

### Track 3 — 🔄 Adaptive Changes
7. **Change Pipeline** — 4 types × 3 sizes, adaptive stage selection
8. **Context Check** — Conflict scanning, requirements smells, impact classification
9. **Architecture Decision Records** — Capturing decisions with context and rationale

### Track 4 — ⚡ Context Engineering
10. **Detail Level** — summary/standard/full verbosity control
11. **Budget Awareness** — max_tokens, token estimation, budget capping
12. **Navigation & Compaction** — Hints, mem_compact, mem_progress
13. **Sub-Agent Scoping** — Namespace isolation for parallel agents

### Track 5 — 🏆 Boss Level
14. **Research Foundations** — Every feature grounded in published research
