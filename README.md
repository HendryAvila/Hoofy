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
| **Memory** | Persistent context across sessions — decisions, bugs, patterns, discoveries. Knowledge graph relations connect observations into a navigable web. Your AI remembers what happened yesterday. | 17 `mem_*` tools |
| **Change Pipeline** | Adaptive workflow for ongoing dev. Picks the right stages based on change type × size (12 flow variants). | 4 `sdd_change*` + `sdd_adr` |
| **Project Pipeline** | Full greenfield specification — from vague idea to validated architecture with a Clarity Gate that blocks hallucinations. | 8 `sdd_*` tools |

One binary. Zero config. Works in **any** MCP-compatible AI tool. **30 tools total.**

### How it flows

```mermaid
flowchart TB
    subgraph project ["New Project (greenfield)"]
        direction LR
        P1[Init] --> P2[Propose] --> P3[Requirements] --> P4{Clarity Gate}
        P4 -->|Ambiguous| P3
        P4 -->|Clear| P5[Design] --> P6[Tasks] --> P7[Validate]
    end

    subgraph change ["Existing Project (changes)"]
        direction LR
        C1["sdd_change\n(type × size)"] --> C2["Opening Stage\n(describe/propose/scope)"]
        C2 --> C3["Spec + Design\n(if needed)"]
        C3 --> C4[Tasks] --> C5[Verify]
    end

    subgraph memory ["Memory (always active)"]
        direction LR
        M1[Session Start] --> M2["Work + Save Discoveries"] --> M3[Session Summary]
    end

    style P4 fill:#f59e0b,stroke:#d97706,color:#000
    style P7 fill:#10b981,stroke:#059669,color:#fff
    style C5 fill:#10b981,stroke:#059669,color:#fff
```

> **[Full workflow guide with step-by-step examples](docs/workflow-guide.md)** · **[Complete tool reference (30 tools)](docs/tool-reference.md)**

---

## Quick Start

### 1. Install the binary

<details open>
<summary><strong>macOS</strong> (Homebrew)</summary>

```bash
brew install HendryAvila/hoofy/hoofy
```
</details>

<details>
<summary><strong>macOS / Linux</strong> (script)</summary>

```bash
curl -sSL https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.sh | bash
```
</details>

<details>
<summary><strong>Windows</strong> (PowerShell)</summary>

```powershell
irm https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.ps1 | iex
```
</details>

<details>
<summary><strong>Go / Source</strong></summary>

```bash
# Go install (requires Go 1.25+)
go install github.com/HendryAvila/Hoofy/cmd/hoofy@latest

# Or build from source
git clone https://github.com/HendryAvila/Hoofy.git
cd Hoofy
make build
```
</details>

### 2. Connect to your AI tool

> **MCP Server vs Plugin — what's the difference?**
>
> The **MCP server** is Hoofy itself — the binary you just installed. It provides 30 tools (memory, change pipeline, project pipeline) and works with **any** MCP-compatible AI tool.
>
> The **Plugin** is a Claude Code-only enhancement that adds an agent personality, skills, lifecycle hooks, and auto-configures the MCP server for you. It's optional — you get full Hoofy functionality with just the MCP server.

<details open>
<summary><strong>Claude Code</strong></summary>

**MCP Server** — one command, done:

```bash
claude mcp add --scope user hoofy hoofy serve
```

**Plugin** (optional, Claude Code only) — adds agent + skills + hooks on top of the MCP server:

```
/plugin marketplace add HendryAvila/hoofy-plugins
/plugin install hoofy@hoofy-plugins
```
</details>

<details>
<summary><strong>Cursor</strong></summary>

Add to your MCP config:

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

Add to `.vscode/mcp.json`:

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
<summary><strong>OpenCode</strong></summary>

Add to `~/.config/opencode/opencode.json` inside the `"mcp"` key:

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

Add to your MCP config:

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

## Best Practices

### 1. Specs before code — always

The AI will try to jump straight to coding. Don't let it. For any non-trivial work:
- **New project?** → `sdd_init_project` and walk through the full pipeline
- **New feature?** → `sdd_change(type: "feature", size: "medium")` at minimum
- **Bug fix?** → Even `sdd_change(type: "fix", size: "small")` gives you describe → tasks → verify

The three cheapest stages (describe + tasks + verify) take under 2 minutes and save hours of debugging hallucinated code.

### 2. Right-size your changes

Don't use a large pipeline for a one-line fix. Don't use a small pipeline for a new authentication system.

| If the change... | It's probably... |
|---|---|
| Touches 1-2 files, clear fix | **small** (3 stages) |
| Needs requirements or design thought | **medium** (4 stages) |
| Affects architecture, multiple systems | **large** (5-6 stages) |

### 3. Let memory work for you

You don't need to tell the AI to use memory — Hoofy's built-in instructions handle it. But you'll get better results if you:
- **Start sessions by greeting the AI** — it triggers `mem_context` to load recent history
- **Mention past decisions** — "remember when we chose SQLite?" triggers `mem_search`
- **Confirm session summaries** — the AI writes them at session end, review them for accuracy

### 4. Connect knowledge with relations

Hoofy's knowledge graph lets you connect related observations with typed, directional edges — turning flat memories into a navigable web.

```
Decision: "Switched to JWT"  →(caused_by)→  Discovery: "Session storage doesn't scale"
    ↑(implements)                               ↑(relates_to)
Bugfix: "Fixed token expiry"              Pattern: "Retry with backoff"
```

The AI creates relations automatically when it recognizes connections. You can also ask it to relate observations manually. Use `mem_build_context` to explore the full graph around any observation.

### 5. Use topic keys for evolving knowledge

When a decision might change (database schema, API design, architecture), use `topic_key` in `mem_save`. This **updates** the existing observation instead of creating duplicates. One observation per topic, always current.

### 6. One change at a time

Hoofy enforces one active change at a time. This isn't a limitation — it's a feature. Scope creep happens when you try to do three things at once. Finish one change, verify it, then start the next.

### 7. Trust the Clarity Gate

When the Clarity Gate asks questions, don't rush past them. Every question it asks represents an ambiguity that would have become a bug, a hallucination, or a "that's not what I meant" moment. Two minutes answering questions saves two hours debugging wrong implementations.

### 8. Hoofy is the architect, Plan mode is the contractor

If your AI tool has a plan/implementation mode, use it **after** Hoofy specs are done. Hoofy answers WHO and WHAT. Plan mode answers HOW.

```
Hoofy (Requirements Layer)  →  "WHAT are we building? For WHO?"
Plan Mode (Implementation)  →  "HOW do we build it? Which files?"
```

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
