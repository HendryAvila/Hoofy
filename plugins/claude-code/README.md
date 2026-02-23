# Hoofy Plugin for Claude Code

> Think before you code. Hoofy is your AI development companion — a sarcastic horse-architect who enforces spec-driven development, manages persistent memory, and teaches through humor.

## Prerequisites

1. **Hoofy binary** installed and in your `$PATH`:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.sh | bash
   ```
   Or via Go:
   ```bash
   go install github.com/HendryAvila/Hoofy/cmd/hoofy@latest
   ```

2. **Claude Code v1.0.33+** installed:
   ```bash
   claude --version  # Should be >= 1.0.33
   ```

## Installation

### Option 1: Marketplace (recommended)

Inside Claude Code, run:

```
/plugin marketplace add HendryAvila/hoofy-plugins
/plugin install hoofy@hoofy-plugins
```

### Option 2: Plugin directory (for development/testing)

```bash
git clone https://github.com/HendryAvila/hoofy-plugins.git
claude --plugin-dir ./hoofy-plugins/plugins/hoofy
```

### Option 3: MCP only (no agent, skills, or hooks)

Add to your `.mcp.json`:

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

## What's Included

### Agent: Hoofy

A sarcastic horse-architect subagent that enforces structured development. Claude automatically delegates to Hoofy when discussing architecture, starting projects, or planning changes.

Invoke manually:
```
Use the hoofy agent to help me plan this feature
```

Or Hoofy will be invoked automatically when Claude detects architecture/planning tasks.

### Skills

| Skill | Description |
|-------|-------------|
| `/hoofy:init` | Initialize a new SDD project — creates structure, guides through proposal |
| `/hoofy:change` | Start an adaptive change pipeline for features, fixes, refactors |
| `/hoofy:review` | Check current SDD pipeline and change status |

Usage:
```
/hoofy:init My Awesome Project — a task management app for developers
/hoofy:change Add user authentication with OAuth
/hoofy:review
```

### Hooks

| Hook | Event | What it does |
|------|-------|-------------|
| Memory loader | `SessionStart` | Automatically loads memory context from previous sessions |
| Pipeline guide | `PostToolUse` | Provides guidance after each SDD pipeline stage advancement |

### MCP Server

The plugin auto-configures the Hoofy MCP server (stdio transport). No manual `claude mcp add` needed. All 26 Hoofy tools become available:

- **Memory tools** (14): `mem_save`, `mem_search`, `mem_context`, `mem_timeline`, `mem_session_start`, `mem_session_end`, `mem_session_summary`, and more
- **Change pipeline** (4): `sdd_change`, `sdd_change_advance`, `sdd_change_status`, `sdd_adr`
- **Project pipeline** (8): `sdd_init_project`, `sdd_create_proposal`, `sdd_generate_requirements`, `sdd_clarify`, `sdd_create_design`, `sdd_create_tasks`, `sdd_validate`, `sdd_get_context`

## Verification

After installing, verify everything works:

1. Start Claude Code with the plugin loaded
2. Run `/help` — you should see `/hoofy:init`, `/hoofy:change`, `/hoofy:review`
3. Run `/agents` — you should see the `hoofy` agent listed
4. Try `/hoofy:review` — should report SDD status (no project initialized yet is fine)

## How It Works

Hoofy's philosophy: **specs before code**. The SDD (Spec-Driven Development) pipeline ensures you think through requirements, clarify ambiguities, and design architecture BEFORE writing a single line of code.

```
New Project:  init → propose → specify → clarify → design → tasks → validate
Changes:      sdd_change → [stages based on type × size] → verify
```

The **Clarity Gate** is the core innovation — it blocks advancement until requirements are clear enough, preventing the #1 cause of AI hallucinations: vague specs.

**Persistent memory** means Hoofy remembers what you did across sessions — architecture decisions, bugs fixed, patterns established. No more repeating context.

## Plugin Structure

```
plugins/claude-code/
├── .claude-plugin/
│   └── plugin.json          # Plugin manifest
├── agents/
│   └── hoofy.md             # Hoofy subagent definition
├── skills/
│   ├── init/
│   │   └── SKILL.md         # /hoofy:init skill
│   ├── change/
│   │   └── SKILL.md         # /hoofy:change skill
│   └── review/
│       └── SKILL.md         # /hoofy:review skill
├── hooks/
│   └── hooks.json           # Lifecycle hooks
├── .mcp.json                # MCP server configuration
└── README.md                # This file
```

## License

MIT
