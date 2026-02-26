# Technical Design: Interactive Documentation Site

## Architecture Overview

Single HTML file with three inline sections: `<style>` for CSS, semantic HTML for structure, `<script>` for interactivity. No build step, no framework, no external dependencies. GitHub Pages serves the file directly.

### Component Structure (logical, not framework components)

```
┌─────────────────────────────────────────┐
│  Hero Section                           │
│  (Title, tagline, progress bar)         │
├─────────────────────────────────────────┤
│  Skill Tree Container                   │
│  ┌─────────────┐                        │
│  │ Track Label  │ "🧠 Foundation"       │
│  ├─────────────┤                        │
│  │  Node ●──●  │ zigzag path with       │
│  │  ●──●──●   │ SVG connectors         │
│  │  Track Label│                        │
│  │  ●──●──●   │                        │
│  └─────────────┘                        │
├─────────────────────────────────────────┤
│  Modal Overlay (hidden by default)      │
│  ┌─────────────────────────────────┐    │
│  │ Icon + Title                    │    │
│  │ Bite-sized summary              │    │
│  │ [Learn More ▼] expandable       │    │
│  │ [Got it! ✓] / [Completed ✓]    │    │
│  └─────────────────────────────────┘    │
├─────────────────────────────────────────┤
│  Hoofy Mascot (floating, milestone)     │
│  Confetti Canvas (completion)           │
└─────────────────────────────────────────┘
```

## Data Model

All content stored as a JS array of node objects — no external data files:

```js
const NODES = [
  {
    id: 1,
    track: "foundation",
    trackLabel: "🧠 Foundation",
    trackStart: true,          // show track label before this node
    icon: "🐴",
    title: "What is Hoofy?",
    summary: "...",            // 2-3 sentences
    details: {
      what: "...",             // 1-2 paragraphs
      how: "...",              // concrete example
      why: "..."               // research link
    }
  },
  // ... 13 more nodes
];
```

Progress stored in localStorage:
```js
// Key: "hoofy-docs-progress"
// Value: JSON array of completed node IDs
// e.g., [1, 2, 3]
```

## CSS Architecture

### Theme Variables
```css
:root {
  --bg-primary: #0d1117;       /* GitHub dark */
  --bg-secondary: #161b22;     /* card backgrounds */
  --bg-tertiary: #21262d;      /* hover states */
  --accent-blue: #58a6ff;      /* primary accent */
  --accent-purple: #bc8cff;    /* secondary accent */
  --accent-green: #3fb950;     /* completion */
  --accent-gold: #d29922;      /* milestone */
  --text-primary: #f0f6fc;
  --text-secondary: #8b949e;
  --text-muted: #484f58;
  --border: #30363d;
  --glow-blue: 0 0 20px rgba(88, 166, 255, 0.3);
  --glow-purple: 0 0 20px rgba(188, 140, 255, 0.3);
  --glow-green: 0 0 20px rgba(63, 185, 80, 0.3);
}
```

### Animations
- **Node pulse**: `@keyframes pulse` — scale 1→1.08→1, loops on unlocked-current nodes
- **Node unlock**: `@keyframes unlock` — scale 0→1.2→1 with opacity, plays once when node becomes unlocked
- **Card entrance**: `@keyframes slideUp` — translateY(30px)→0 with opacity
- **Confetti**: Canvas-based particle system — 100 particles, gravity, fade out over 3 seconds
- **Path draw**: SVG stroke-dashoffset animation — connector lines "draw" as user scrolls
- **Sparkle**: CSS pseudo-element particles around completed node

### Layout
- Desktop: max-width 600px centered, nodes zigzag left-right
- Mobile (<768px): max-width 100%, nodes centered, no zigzag
- Path connectors: CSS `::before` pseudo-elements or inline SVG lines

## JS Architecture

### State Management
```
┌──────────────────┐
│  State (in-memory)│
│  - completedIds  │ ← loaded from localStorage on init
│  - currentNodeId │ ← derived: first non-completed
│  - modalOpen     │
│  - modalNodeId   │
└──────────────────┘
        │
        ▼
┌──────────────────┐
│  render()        │ ← re-renders all node states + progress bar
│  - updateNodes() │
│  - updateProgress│
│  - saveToStorage │
└──────────────────┘
```

### Functions
- `init()` — load progress from localStorage, render initial state
- `render()` — update all node classes, progress bar, track visibility
- `openNode(id)` — open modal for node, populate content
- `closeModal()` — close modal with animation
- `completeNode(id)` — mark complete, save to localStorage, unlock next, render, scroll
- `toggleDetails(id)` — expand/collapse "Learn More" section
- `resetProgress()` — clear localStorage, re-render
- `showMascot(message)` — show Hoofy emoji with message at milestones
- `showConfetti()` — fire confetti canvas animation on final completion
- `getNodeState(id)` — returns 'locked' | 'unlocked' | 'current' | 'completed'

### Event Handling
- Click on node circle → `openNode(id)` (only if unlocked/current/completed)
- Click "Got it!" → `completeNode(id)`
- Click "Learn More" → `toggleDetails(id)`
- Click modal backdrop / X → `closeModal()`
- Click "Reset Progress" → confirm dialog → `resetProgress()`
- `DOMContentLoaded` → `init()`

## Deployment
- File location: `/site/index.html`
- GitHub Pages config: serve from `/site` directory on `main` branch
- No build step — push and it's live
