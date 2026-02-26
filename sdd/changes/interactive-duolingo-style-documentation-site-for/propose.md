# Proposal: Interactive Duolingo-Style Documentation Site

## Problem Statement
Hoofy has 34 MCP tools across 3 systems (Memory, Change Pipeline, Project Pipeline) with 6 research-backed features — but the only documentation is markdown files in the repo. New users have no engaging way to learn what Hoofy does, why each feature exists, or how to use it. Traditional docs are passive walls of text that people skim and forget.

## Target Users
- **AI developers** evaluating Hoofy for their workflow — need to quickly understand what it does and why it matters
- **Current Hoofy users** wanting to learn features they haven't explored yet — need progressive discovery
- **Curious devs** who heard about SDD or MCP tools — need an engaging intro that hooks them

## Proposed Solution
A single-page interactive documentation site hosted on GitHub Pages. Duolingo-style skill tree where concepts are presented as circular nodes connected by a path. Users click nodes to learn each feature through bite-sized cards that expand for deeper dives. Progress is saved in localStorage — completed nodes stay unlocked across sessions. The page feels alive with CSS animations, smooth transitions, and visual feedback.

### Skill Tree Structure (Draft)

**Track 1 — Foundation** (top of path):
1. What is Hoofy? (intro, MCP server, what problem it solves)
2. Persistent Memory (SQLite, FTS5, observations, sessions)
3. Knowledge Graph (relations, traversal, connected decisions)

**Track 2 — Spec-Driven Development**:
4. SDD Pipeline (propose → specify → clarify → design → tasks → validate)
5. Clarity Gate (8-dimension ambiguity analysis, threshold blocking)
6. Business Rules (BRG taxonomy, ubiquitous language)

**Track 3 — Adaptive Changes**:
7. Change Pipeline (4 types × 3 sizes = 12 flows)
8. Context Check (conflict scanning, IEEE 29148 smells)
9. ADRs (Architecture Decision Records)

**Track 4 — Context Engineering** (research-backed features):
10. Detail Level (summary/standard/full verbosity control)
11. Budget Awareness (max_tokens, token estimation)
12. Navigation & Compaction (hints, mem_compact, mem_progress)
13. Sub-Agent Scoping (namespace isolation)

**Final node — Boss level**:
14. Research Foundations (why every feature is grounded in published research)

### UX Details
- Nodes are circles on a winding vertical path (like Duolingo's lesson tree)
- Locked nodes are grayed out with a lock icon
- Completed nodes glow/pulse with a checkmark
- Current (unlocked, not completed) nodes bounce subtly to invite clicking
- Clicking a node opens a card overlay with:
  - Title + icon
  - 2-3 sentence bite-sized explanation
  - "Learn More" expands to deeper content with usage examples
  - "Got it!" button marks as complete and unlocks next node
- Confetti or sparkle animation on completion
- Progress bar at top showing X/14 completed
- Dark theme with Hoofy's circuit-vest color palette (dark bg, electric blue/purple accents)

## Out of Scope
- No backend, no API, no database
- No user accounts or server-side progress
- No interactive code playground (future v2 maybe)
- No search functionality (14 nodes is small enough)
- No multi-language support (English only for v1)

## Success Criteria
- Site loads in under 2 seconds (single HTML file, no external deps)
- All 14 nodes are navigable and contain accurate Hoofy documentation
- Progress persists across browser sessions via localStorage
- Works on mobile (responsive)
- Feels engaging — animations, transitions, visual feedback on every interaction
