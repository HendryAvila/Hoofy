## Verification

### TASK-001: README Quick Start rewrite ✅

**Install section**:
- [x] macOS Homebrew shown first (open by default): `brew install HendryAvila/hoofy/hoofy`
- [x] macOS/Linux script: `curl | bash`
- [x] Windows PowerShell: `irm | iex`
- [x] Go/Source in collapsed details

**MCP vs Plugin clarity**:
- [x] Blockquote callout explaining the difference between MCP Server and Plugin
- [x] MCP = the binary, works everywhere, 27 tools
- [x] Plugin = Claude Code only, adds agent + skills + hooks, optional

**Claude Code section**:
- [x] `claude mcp add --scope user hoofy hoofy serve` shown prominently as the primary MCP setup
- [x] Plugin shown as optional secondary option

**Other tools**:
- [x] Cursor, VS Code Copilot, OpenCode, Gemini CLI all preserved with their configs
- [x] Each now has a brief intro line ("Add to your MCP config", "Add to `.vscode/mcp.json`")
