# Clarification

## Questions & Answers

### Q1: Where should the site files live in the repo?
**Decision**: `/site/index.html` — a dedicated `site/` directory at repo root. GitHub Pages can be configured to serve from this folder. Keeps docs separate from Go source code.

### Q2: Should nodes require reading the "Learn More" section before marking complete?
**Decision**: No — the user can click "Got it!" after reading just the bite-sized summary. The "Learn More" is optional for those who want depth. We want to encourage exploration, not force it. Lowering friction = more engagement.

### Q3: Linear path or branching?
**Decision**: Linear path with track headers. All 14 nodes in a single vertical path — no branching. Keeps the UX simple and the implementation clean. Tracks are just visual groupings (labels between nodes), not separate branches.

### Q4: What happens if a user wants to revisit a completed node?
**Decision**: Completed nodes remain clickable — they open the same card but with a "Completed ✓" badge instead of "Got it!" button. Users can re-read any time.

### Q5: How should the "Learn More" content be structured?
**Decision**: Each node's deep content has 3 parts:
1. **What it does** — detailed explanation (1-2 paragraphs)
2. **How it works** — concrete example of the tool/feature in action
3. **Why it matters** — link to the research/reasoning behind it

## Clarity Assessment
All functional and non-functional requirements are clear. No ambiguity in scope, UX, or technical approach. User preferences captured (localStorage, mix content depth, full interactivity). Ready for design.
