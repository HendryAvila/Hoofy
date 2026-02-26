# Proposal: Claude Code Plugin Integration Package

## Problem Statement

Hoofy is a powerful MCP server with 26 tools for spec-driven development and persistent memory, but it currently only has first-class integration with OpenCode. Claude Code — the most popular AI coding tool (69.3k★) — supports MCP servers natively but requires manual setup and has no structured guidance on how to use Hoofy's tools effectively. Users who install Hoofy in Claude Code get raw tools with no workflow enforcement, no personality, and no guardrails.

## Target Users

- **Claude Code developers** who want structured development workflows instead of ad-hoc coding
- **Teams using Claude Code** who want consistent spec-driven practices across projects
- **Hoofy users on OpenCode** who also use Claude Code and want the same experience

## Proposed Solution

Create a **Hoofy Plugin for Claude Code** that bundles:

1. **MCP Server auto-configuration** (`.mcp.json`) — Hoofy MCP server starts automatically when the plugin is enabled, no manual `claude mcp add` needed
2. **Hoofy Agent** (`agents/hoofy.md`) — The sarcastic horse-architect personality as a Claude Code subagent, enforcing SDD workflows and teaching through humor
3. **Skills** — Packaged SDD workflows as invokable `/hoofy:*` skills:
   - `/hoofy:init` — Initialize a new SDD project
   - `/hoofy:change` — Start an adaptive change pipeline
   - `/hoofy:review` — Review current SDD pipeline status
4. **Hooks** (`hooks/hooks.json`) — Lifecycle automation:
   - `SessionStart` hook to load memory context automatically
   - `PostToolUse` hook on Hoofy MCP tools to provide guidance after pipeline operations
5. **README.md** — Installation and usage documentation

The plugin lives at `plugins/claude-code/` in the Hoofy repo and can be installed via `claude --plugin-dir` or through a marketplace.

## Out of Scope

- Hoofy Cloud / remote memory sync
- Skills marketplace / registry
- Custom LSP server integration
- Changes to the Hoofy Go codebase (this is purely a plugin package)
- Plugin marketplace distribution (we'll do `--plugin-dir` and manual install first)
- Agent Teams integration (subagents can't spawn subagents in Claude Code)

## Success Criteria

- Plugin loads cleanly with `claude --plugin-dir ./plugins/claude-code/` — no errors
- `/hoofy:init`, `/hoofy:change`, and `/hoofy:review` skills are discoverable and functional
- Hoofy agent appears in `/agents` and can be invoked manually or automatically
- SessionStart hook successfully loads memory context
- Plugin structure follows Claude Code's official plugin specification exactly
- README provides clear installation instructions for both plugin-dir and manual methods

## Open Questions

- Should the Hoofy agent be set as the default agent via `settings.json` `agent` field? (Probably yes for full Hoofy experience, but might be intrusive)
- Should we include a `PreToolUse` hook to intercept code-writing attempts and enforce specs? (Powerful but potentially annoying — needs user testing)
- What model should the Hoofy agent use? `inherit` (user's choice) vs `sonnet` (consistent personality)?
