<p align="center">
  <img src="assets/logo.png" alt="Hoofy" width="280" />
</p>

<h1 align="center">Hoofy</h1>

<p align="center">
  <strong>Your AI development companion.</strong><br>
  An MCP server that gives your AI persistent memory, structured specifications,<br>
  and adaptive change management — so it builds what you actually want.
</p>

<p align="center">
  <a href="https://github.com/HendryAvila/Hoofy/actions/workflows/ci.yml"><img src="https://github.com/HendryAvila/Hoofy/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-Compatible-purple" alt="MCP"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://github.com/HendryAvila/Hoofy/releases"><img src="https://img.shields.io/github/v/release/HendryAvila/Hoofy?include_prereleases" alt="Release"></a>
</p>

---

## What Hoofy Does

Hoofy is three systems in one MCP server:

| System | What it does | Tools |
|---|---|---|
| **Memory** | Persistent context across sessions — decisions, bugs, patterns, discoveries. Your AI remembers what happened yesterday. | 14 `mem_*` tools |
| **Change Pipeline** | Adaptive workflow for ongoing dev. Picks the right stages based on change type × size (12 flow variants). | 4 `sdd_change*` + `sdd_adr` |
| **Project Pipeline** | Full greenfield specification — from vague idea to validated architecture with a Clarity Gate that blocks hallucinations. | 8 `sdd_*` tools |

One binary. Zero config. Works in **any** MCP-compatible AI tool.

---

## Quick Start

### 1. Install

```bash
curl -sSL https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.sh | bash
```

<details>
<summary>Other methods</summary>

```bash
# Go install (requires Go 1.25+)
go install github.com/HendryAvila/Hoofy/cmd/hoofy@latest

# Build from source
git clone https://github.com/HendryAvila/Hoofy.git
cd Hoofy
make build
```
</details>

### 2. Add to your AI tool

<details>
<summary><strong>Claude Code</strong> (recommended: use the plugin)</summary>

**Option A — Plugin via Marketplace (recommended)**

Inside Claude Code, run:

```
/plugin marketplace add HendryAvila/hoofy-plugins
/plugin install hoofy@hoofy-plugins
```

That's it. You get the Hoofy agent, 3 skills, lifecycle hooks, and MCP auto-configuration.

**Option B — Plugin from local directory**

```bash
git clone https://github.com/HendryAvila/hoofy-plugins.git
claude --plugin-dir ./hoofy-plugins/plugins/hoofy
```

**Option C — MCP only (manual)**

```json
{
  "mcpServers": {
    "hoofy": {
      "command": "hoofy",
      "args": ["serve"]
    }
  }
}
```
</details>

<details>
<summary><strong>VS Code Copilot</strong></summary>

```json
{
  "servers": {
    "hoofy": {
      "type": "stdio",
      "command": "hoofy",
      "args": ["serve"]
    }
  }
}
```
</details>

<details>
<summary><strong>Cursor</strong></summary>

```json
{
  "mcpServers": {
    "hoofy": {
      "command": "hoofy",
      "args": ["serve"]
    }
  }
}
```
</details>

<details>
<summary><strong>OpenCode</strong></summary>

```json
{
  "mcp": {
    "hoofy": {
      "type": "local",
      "command": ["hoofy", "serve"],
      "enabled": true
    }
  }
}
```
</details>

<details>
<summary><strong>Gemini CLI</strong></summary>

```json
{
  "mcpServers": {
    "hoofy": {
      "command": "hoofy",
      "args": ["serve"]
    }
  }
}
```
</details>

### 3. Use it

Just talk to your AI. Hoofy's built-in instructions tell the AI when and how to use each system.

### 4. Update

```bash
hoofy update
```

Auto-checks on startup, updates when you say so.

### 5. Reinforce the behavior (recommended)

Hoofy already includes built-in server instructions, but adding a short block to your agent's instructions file reinforces the habit — the AI will think about specs *before* it even sees the tools.

<details>
<summary><strong>Claude Code</strong> — <code>CLAUDE.md</code> (only needed for MCP-only setup)</summary>

> **Using the plugin?** Skip this — the plugin's hooks and agent already enforce SDD behavior automatically.

```markdown
## Hoofy — Spec-Driven Development

Before coding any new feature or significant change, use Hoofy to create specs first.
- New projects: use the SDD pipeline (sdd_init_project → sdd_validate)
- Ongoing work: use the change pipeline (sdd_change) — it adapts stages to the size of the change
- Memory: save decisions, bugs, and discoveries with mem_save so future sessions have context
Do NOT start coding without specs for any non-trivial change.
```
</details>

<details>
<summary><strong>Cursor</strong> — <code>.cursor/rules/hoofy.md</code></summary>

```markdown
## Hoofy — Spec-Driven Development

Before coding any new feature or significant change, use Hoofy to create specs first.
- New projects: use the SDD pipeline (sdd_init_project → sdd_validate)
- Ongoing work: use the change pipeline (sdd_change) — it adapts stages to the size of the change
- Memory: save decisions, bugs, and discoveries with mem_save so future sessions have context
Do NOT start coding without specs for any non-trivial change.
```
</details>

<details>
<summary><strong>OpenCode</strong> — <code>AGENTS.md</code></summary>

```markdown
## Hoofy — Spec-Driven Development

Before coding any new feature or significant change, use Hoofy to create specs first.
- New projects: use the SDD pipeline (sdd_init_project → sdd_validate)
- Ongoing work: use the change pipeline (sdd_change) — it adapts stages to the size of the change
- Memory: save decisions, bugs, and discoveries with mem_save so future sessions have context
Do NOT start coding without specs for any non-trivial change.
```
</details>

<details>
<summary><strong>VS Code Copilot</strong> — <code>.github/copilot-instructions.md</code></summary>

```markdown
## Hoofy — Spec-Driven Development

Before coding any new feature or significant change, use Hoofy to create specs first.
- New projects: use the SDD pipeline (sdd_init_project → sdd_validate)
- Ongoing work: use the change pipeline (sdd_change) — it adapts stages to the size of the change
- Memory: save decisions, bugs, and discoveries with mem_save so future sessions have context
Do NOT start coding without specs for any non-trivial change.
```
</details>

<details>
<summary><strong>Gemini CLI</strong> — <code>GEMINI.md</code></summary>

```markdown
## Hoofy — Spec-Driven Development

Before coding any new feature or significant change, use Hoofy to create specs first.
- New projects: use the SDD pipeline (sdd_init_project → sdd_validate)
- Ongoing work: use the change pipeline (sdd_change) — it adapts stages to the size of the change
- Memory: save decisions, bugs, and discoveries with mem_save so future sessions have context
Do NOT start coding without specs for any non-trivial change.
```
</details>

---

## Memory System

Hoofy gives your AI **persistent memory** across sessions using SQLite + FTS5 full-text search. No more re-explaining context every time you start a conversation.

### What gets remembered

- **Decisions** — Why you chose PostgreSQL over MongoDB
- **Bug fixes** — What was wrong, why, how you fixed it
- **Patterns** — Conventions established in the codebase
- **Discoveries** — Gotchas, edge cases, non-obvious learnings
- **Session summaries** — What happened in each coding session

### How it works

Memory is structured around **observations** with types (`decision`, `architecture`, `bugfix`, `pattern`, `discovery`, `config`), **topic keys** for upserts (evolving knowledge overwrites, not duplicates), and **sessions** for temporal context.

```
mem_save → Store an observation with structured content
mem_search → Full-text search across all sessions
mem_context → Recent context for session startup
mem_timeline → Chronological drill-down around a specific event
mem_session_start/end/summary → Session lifecycle management
```

The AI uses these proactively — saving important findings after completing work, searching past context when starting new tasks, and building session summaries when wrapping up.

---

## Change Pipeline

For ongoing development, Hoofy adapts the number of stages to what you're actually doing. A small bug fix doesn't need the same ceremony as a large feature.

### Adaptive flows

```
                    Small           Medium              Large
                    ─────           ──────              ─────
Fix             describe→tasks    describe→spec       describe→spec→design
                  →verify          →tasks→verify        →tasks→verify

Feature         describe→tasks    propose→spec        propose→spec→clarify
                  →verify          →tasks→verify        →design→tasks→verify

Refactor        scope→tasks       scope→design        scope→spec→design
                  →verify          →tasks→verify        →tasks→verify

Enhancement     describe→tasks    propose→spec        propose→spec→clarify
                  →verify          →tasks→verify        →design→tasks→verify
```

**12 flow variants**, all deterministic. One active change at a time to prevent scope creep.

### ADRs (Architecture Decision Records)

First-class concept. Record decisions at any point with full lifecycle management (`proposed` → `accepted` → `deprecated` / `rejected`):

```
sdd_adr(title: "PostgreSQL over MongoDB", status: "accepted",
        context: "Need ACID for financial data",
        decision: "Use PostgreSQL 16",
        consequences: "Need migration tooling")
```

### Change artifacts

```
sdd/changes/
└── fix-login-timeout/
    ├── manifest.json       # Metadata (type, size, status, stages)
    ├── describe.md         # What's the problem/feature?
    ├── spec.md             # Requirements (medium+ changes)
    ├── design.md           # Technical approach (large changes)
    ├── tasks.md            # Implementation breakdown
    └── verify.md           # Verification checklist
```

---

## Project Pipeline

For greenfield projects, Hoofy runs a full 7-stage specification process:

```
Vague Idea → Proposal → Requirements → Clarity Gate → Architecture → Tasks → Validation
```

The **Clarity Gate** is the core innovation. It analyzes requirements across 8 dimensions (target users, core functionality, data model, integrations, edge cases, security, scale, scope boundaries) and **blocks progress** until ambiguities are resolved. The AI can't skip ahead to architecture until specs are clear enough.

- **Guided mode** (70/100 threshold) — Step-by-step with examples, for non-technical users
- **Expert mode** (50/100 threshold) — Streamlined for experienced developers

After the gate: technical design with ADRs → atomic task breakdown with dependency graphs → cross-artifact validation that catches inconsistencies before any code is written.

---

## Available Tools

### Memory (14 tools)

| Tool | Description |
|---|---|
| `mem_save` | Save an observation (decision, bugfix, pattern, etc.) |
| `mem_save_prompt` | Record user intent for future context |
| `mem_search` | Full-text search across all sessions |
| `mem_context` | Recent observations for session startup |
| `mem_timeline` | Chronological context around a specific event |
| `mem_get_observation` | Full content of a specific observation |
| `mem_session_start` | Register a new coding session |
| `mem_session_end` | Close a session with summary |
| `mem_session_summary` | Save comprehensive end-of-session summary |
| `mem_stats` | Memory system statistics |
| `mem_capture_passive` | Passive observation capture |
| `mem_delete` | Remove an observation |
| `mem_update` | Update an existing observation |
| `mem_suggest_topic_key` | Suggest stable key for upserts |

### Change Pipeline (5 tools)

| Tool | Description |
|---|---|
| `sdd_change` | Create a new change (feature, fix, refactor, enhancement) with size |
| `sdd_change_advance` | Save stage content and advance to next stage |
| `sdd_change_status` | View current change status and artifacts |
| `sdd_adr` | Create or update Architecture Decision Records |

### Project Pipeline (8 tools)

| Tool | Description |
|---|---|
| `sdd_init_project` | Initialize project structure |
| `sdd_create_proposal` | Save structured proposal |
| `sdd_generate_requirements` | Save formal requirements (MoSCoW) |
| `sdd_clarify` | Run the Clarity Gate |
| `sdd_create_design` | Save technical architecture |
| `sdd_create_tasks` | Save implementation task breakdown |
| `sdd_validate` | Cross-artifact consistency check |
| `sdd_get_context` | View project state and artifacts |

### Prompts

| Prompt | Description |
|---|---|
| `/sdd-start` | Start a new SDD project |
| `/sdd-status` | Check pipeline status |

---

## Hoofy vs Plan Mode

If you use Cursor, Claude Code, Codex, or similar tools, you've probably used plan mode. They're complementary:

```
┌─────────────────────────────────────────────┐
│  Hoofy (Requirements + Memory Layer)        │
│  WHO are the users? WHAT must the system    │
│  do? What happened in yesterday's session?  │
├─────────────────────────────────────────────┤
│  /plan mode (Implementation Layer)          │
│  What files? What functions? What tests?    │
└─────────────────────────────────────────────┘
```

**Hoofy is the architect. Plan mode is the contractor.** You wouldn't hire a contractor without blueprints — and you shouldn't use plan mode without specifications.

---

## The Research Behind SDD

Hoofy's specification pipeline isn't built on opinions. It's built on research:

- **METR 2025**: Experienced developers were [19% slower with AI](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/) despite feeling 20% faster — unstructured AI usage introduces debugging overhead and false confidence.

- **DORA 2025**: [7.2% delivery instability increase](https://dora.dev/research/2025/dora-report/) for every 25% AI adoption — without foundational systems and practices.

- **McKinsey 2025**: Top performers see [16-30% productivity gains](https://www.mckinsey.com/capabilities/mckinsey-digital/our-insights/superagency-in-the-workplace-empowering-people-to-unlock-ais-full-potential-at-work) only with structured specification and communication.

- **IEEE Requirements Engineering**: Fixing a requirement error in production costs [10-100x more](https://ieeexplore.ieee.org/document/720574) than fixing it during requirements. This multiplier is worse with AI-generated code.

**Structure beats speed.**

---

## Contributing

```bash
git clone https://github.com/HendryAvila/Hoofy.git
cd Hoofy
make build        # Build binary
make test         # Tests with race detector
make lint         # golangci-lint
./bin/hoofy serve # Run the MCP server
```

### Areas for contribution

- More clarity dimensions (mobile, API, data pipeline)
- More change types beyond fix/feature/refactor/enhancement
- Template improvements and customization
- Streamable HTTP transport for remote deployment
- Export to Jira, Linear, GitHub Issues
- i18n for non-English specs

---

## License

[MIT](LICENSE)

---

<p align="center">
  <strong>Stop prompting. Start specifying.</strong><br>
  Built with care by the Hoofy community.
</p>
