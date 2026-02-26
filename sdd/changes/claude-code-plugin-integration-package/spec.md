# Specification: Claude Code Plugin Integration Package

## Functional Requirements

### Plugin Structure (FR-001 through FR-004)

- **FR-001**: Plugin directory at `plugins/claude-code/` follows Claude Code's official plugin structure exactly:
  - `.claude-plugin/plugin.json` — manifest with name, version, description, author, repository, keywords
  - `agents/` — subagent definitions
  - `skills/` — Agent Skills with SKILL.md
  - `hooks/hooks.json` — lifecycle hooks
  - `.mcp.json` — MCP server configuration
  - `README.md` — documentation

- **FR-002**: `.claude-plugin/plugin.json` manifest contains:
  - `name`: `"hoofy"` (skills appear as `/hoofy:*`, agent as `hoofy:*`)
  - `version`: matches current Hoofy release (`0.4.0`)
  - `description`: concise plugin purpose
  - `author`: with name, url
  - `repository`: GitHub URL
  - `keywords`: discovery tags
  - `license`: MIT

- **FR-003**: `.mcp.json` configures Hoofy MCP server:
  - Transport: stdio
  - Command: `hoofy`
  - Args: `["serve"]`
  - Server name: `hoofy`

- **FR-004**: `README.md` documents:
  - Prerequisites (Hoofy binary installed)
  - Installation via `claude --plugin-dir`
  - Installation via manual copy to `~/.claude/plugins/`
  - Available skills, agent, and hooks
  - How to verify the plugin is working

### Agent (FR-005)

- **FR-005**: `agents/hoofy.md` defines the Hoofy subagent:
  - `name`: `hoofy`
  - `description`: Clear trigger description for Claude's automatic delegation — should mention SDD, architecture, specs, memory
  - `model`: `inherit` (user controls model choice)
  - `tools`: All tools (needs MCP tools for Hoofy operations + Read/Write/Edit/Bash for code work)
  - `mcpServers`: references `hoofy` from the plugin's `.mcp.json`
  - System prompt: Full Hoofy personality (sarcastic horse-architect, never gives code directly, Socratic method, concepts > code, bilingual ES/EN)
  - Memory operations guidance: always check `mem_context` first, save decisions, end with summaries
  - SDD pipeline guidance: refuse to code without specs, use `sdd_change` for modifications, enforce Clarity Gate

### Skills (FR-006 through FR-008)

- **FR-006**: `skills/init/SKILL.md` — Initialize SDD project:
  - `description`: triggers on new project setup, bootstrapping, "start SDD"
  - Instructions: call `sdd_init_project`, guide user through proposal
  - Accepts `$ARGUMENTS` for project name/description

- **FR-007**: `skills/change/SKILL.md` — Start adaptive change:
  - `description`: triggers on new feature, fix, refactor, enhancement
  - Instructions: call `sdd_change`, guide through pipeline stages
  - Accepts `$ARGUMENTS` for change description

- **FR-008**: `skills/review/SKILL.md` — Review SDD status:
  - `description`: triggers on checking pipeline status, progress review
  - Instructions: call `sdd_get_context` and `sdd_change_status`, present formatted status
  - `disable-model-invocation`: false (Claude can auto-invoke)

### Hooks (FR-009 through FR-010)

- **FR-009**: `SessionStart` hook:
  - Runs on every session start
  - Type: `prompt` — uses LLM to generate a context-loading instruction
  - Prompt instructs Claude to call `mcp__hoofy__mem_context` to load previous session memory
  - Fallback: if Hoofy MCP is not available, hook should not crash (exit 0)

- **FR-010**: `PostToolUse` hook on SDD change operations:
  - Matcher: `mcp__hoofy__sdd_change_advance`
  - Type: `prompt` — uses LLM to interpret the stage result and guide the user to the next step
  - Provides contextual guidance after each pipeline stage advancement

## Non-Functional Requirements

- **NFR-001**: Plugin must load without errors using `claude --plugin-dir ./plugins/claude-code/`
- **NFR-002**: No runtime dependencies beyond Hoofy binary being in `$PATH`
- **NFR-003**: All file paths within the plugin use `${CLAUDE_PLUGIN_ROOT}` for portability
- **NFR-004**: Plugin follows Claude Code's plugin specification as of v1.0.33+
- **NFR-005**: Hoofy agent system prompt must be under ~4000 tokens to avoid context bloat
- **NFR-006**: Hook scripts must be idempotent — safe to run multiple times
- **NFR-007**: Plugin must gracefully degrade if Hoofy MCP server is not installed (hooks exit 0, agent still provides guidance)

## Constraints

- No changes to the Hoofy Go codebase
- Plugin is static files only (JSON, Markdown) — no compiled code
- Must work with Claude Code v1.0.33+
- Hoofy binary must be pre-installed and in PATH

## Assumptions

- Users have Hoofy installed via `install.sh` or `go install`
- Users have Claude Code v1.0.33+ installed
- Users understand basic Claude Code plugin concepts (or can follow README)
- The `hoofy serve` command works correctly via stdio transport
