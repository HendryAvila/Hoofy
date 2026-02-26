# Tasks: Claude Code Plugin Integration Package

## Estimated Effort
~2-3 hours for a single developer. All static files (JSON, Markdown), no compilation.

## Task Breakdown

### TASK-001: Create plugin directory structure and manifest
**Covers**: FR-001, FR-002
**Dependencies**: None
**Description**: Create the `plugins/claude-code/` directory with `.claude-plugin/plugin.json` manifest following Claude Code's official plugin spec.
**Files**:
- `plugins/claude-code/.claude-plugin/plugin.json`
**Acceptance Criteria**:
- [ ] Directory structure matches Claude Code plugin layout
- [ ] `plugin.json` has name, version, description, author, repository, keywords, license
- [ ] Name is `hoofy` (namespace for skills/agents)

### TASK-002: Create MCP server configuration
**Covers**: FR-003
**Dependencies**: TASK-001
**Description**: Create `.mcp.json` at plugin root configuring Hoofy MCP server with stdio transport.
**Files**:
- `plugins/claude-code/.mcp.json`
**Acceptance Criteria**:
- [ ] Configures `hoofy` server with command `hoofy` and args `["serve"]`
- [ ] Uses stdio transport (default)
- [ ] Server name matches what agents/hooks reference

### TASK-003: Create Hoofy agent definition
**Covers**: FR-005
**Dependencies**: TASK-001
**Description**: Create `agents/hoofy.md` with full Hoofy personality as a Claude Code subagent. System prompt includes: personality traits (sarcastic horse-architect, bilingual, Socratic method), core rules (never give code directly, concepts > code, foundations first), MCP usage guidance (memory operations, SDD pipeline enforcement), and teaching methodology (The Hoofy Method™).
**Files**:
- `plugins/claude-code/agents/hoofy.md`
**Acceptance Criteria**:
- [ ] Valid YAML frontmatter with name, description, model, tools
- [ ] System prompt captures Hoofy personality faithfully
- [ ] Under ~4000 tokens (NFR-005)
- [ ] Includes memory and SDD workflow guidance
- [ ] model is `inherit`
- [ ] Description is clear enough for Claude's auto-delegation

### TASK-004: Create `/hoofy:init` skill
**Covers**: FR-006
**Dependencies**: TASK-001
**Description**: Create `skills/init/SKILL.md` that guides SDD project initialization. Instructs the AI to use `mcp__hoofy__sdd_init_project` and walk the user through proposal creation.
**Files**:
- `plugins/claude-code/skills/init/SKILL.md`
**Acceptance Criteria**:
- [ ] Valid YAML frontmatter with description
- [ ] References correct MCP tool names (`mcp__hoofy__sdd_init_project`)
- [ ] Accepts `$ARGUMENTS` for project name/description
- [ ] Provides clear step-by-step workflow

### TASK-005: Create `/hoofy:change` skill
**Covers**: FR-007
**Dependencies**: TASK-001
**Description**: Create `skills/change/SKILL.md` that starts an adaptive change pipeline. Instructs the AI to determine type/size from user input and call `mcp__hoofy__sdd_change`.
**Files**:
- `plugins/claude-code/skills/change/SKILL.md`
**Acceptance Criteria**:
- [ ] Valid YAML frontmatter with description
- [ ] References correct MCP tool names (`mcp__hoofy__sdd_change`, `mcp__hoofy__sdd_change_advance`)
- [ ] Accepts `$ARGUMENTS` for change description
- [ ] Explains the 4 types (feature/fix/refactor/enhancement) and 3 sizes (small/medium/large)

### TASK-006: Create `/hoofy:review` skill
**Covers**: FR-008
**Dependencies**: TASK-001
**Description**: Create `skills/review/SKILL.md` that checks current SDD pipeline and change status.
**Files**:
- `plugins/claude-code/skills/review/SKILL.md`
**Acceptance Criteria**:
- [ ] Valid YAML frontmatter with description
- [ ] References `mcp__hoofy__sdd_get_context` and `mcp__hoofy__sdd_change_status`
- [ ] Presents status in a clear, formatted way

### TASK-007: Create hooks configuration
**Covers**: FR-009, FR-010
**Dependencies**: TASK-001
**Description**: Create `hooks/hooks.json` with SessionStart hook (loads memory context) and PostToolUse hook on `sdd_change_advance` (guides next steps).
**Files**:
- `plugins/claude-code/hooks/hooks.json`
**Acceptance Criteria**:
- [ ] Valid JSON following Claude Code hooks schema
- [ ] SessionStart hook uses prompt type to instruct memory loading
- [ ] PostToolUse hook matches `mcp__hoofy__sdd_change_advance`
- [ ] Hooks are idempotent (NFR-006)
- [ ] Graceful degradation if Hoofy MCP unavailable (NFR-007)

### TASK-008: Create plugin README
**Covers**: FR-004
**Dependencies**: TASK-001 through TASK-007
**Description**: Create comprehensive README.md documenting installation, prerequisites, available features, and verification steps.
**Files**:
- `plugins/claude-code/README.md`
**Acceptance Criteria**:
- [ ] Documents prerequisites (Hoofy binary, Claude Code v1.0.33+)
- [ ] Installation via `claude --plugin-dir`
- [ ] Manual installation instructions
- [ ] Lists all skills, agent, and hooks
- [ ] Includes verification steps

### TASK-009: Verify — end-to-end structure validation
**Covers**: NFR-001, NFR-002, NFR-003, NFR-004
**Dependencies**: TASK-001 through TASK-008
**Description**: Verify all files exist, JSON is valid, Markdown frontmatter is correct, paths use `${CLAUDE_PLUGIN_ROOT}` where needed, and the overall structure matches Claude Code's plugin spec.
**Acceptance Criteria**:
- [ ] All JSON files parse without errors
- [ ] All YAML frontmatter in .md files is valid
- [ ] Directory structure matches Claude Code plugin layout
- [ ] No absolute paths (use `${CLAUDE_PLUGIN_ROOT}`)
- [ ] No references to non-existent files

## Dependency Graph

```
TASK-001 (structure + manifest)
├── TASK-002 (MCP config)
├── TASK-003 (agent)
├── TASK-004 (skill: init)
├── TASK-005 (skill: change)
├── TASK-006 (skill: review)
└── TASK-007 (hooks)
    └── TASK-008 (README) ← depends on all above
        └── TASK-009 (verify) ← depends on everything
```

TASK-002 through TASK-007 can run in parallel after TASK-001.
