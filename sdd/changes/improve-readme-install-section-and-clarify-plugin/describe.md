## What

Restructure README Quick Start section to:
1. Show all install methods (macOS/Homebrew, Linux, Windows, Go, source)
2. Clearly separate "Plugin" vs "MCP Server" — explain what each is
3. Add `claude mcp add` command as the easiest MCP setup for Claude Code
4. Make the flow: Install binary → Connect to your AI tool (MCP or Plugin)

## Why

- Windows installer and Homebrew tap were added but README still only shows `curl | bash`
- Users are confused about what's the plugin vs what's the MCP server
- `claude mcp add` is the simplest way to add an MCP server in Claude Code but it's not documented
