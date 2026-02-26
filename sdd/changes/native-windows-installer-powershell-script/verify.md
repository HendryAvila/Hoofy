## Verification

### TASK-001: install.ps1 — Checklist

- [x] **Architecture detection**: Uses `$env:PROCESSOR_ARCHITECTURE` — handles `AMD64`, `ARM64`, and WoW64 (32-bit PS on 64-bit OS via `$env:PROCESSOR_ARCHITEW6432`)
- [x] **Version fetching**: `Invoke-RestMethod` to GitHub API, strips `v` prefix from tag
- [x] **Download**: `Invoke-WebRequest` for `.zip` archive — `hoofy_<version>_windows_<arch>.zip`
- [x] **Extraction**: `Expand-Archive` (native PS 5.1+), searches recursively for `hoofy.exe`
- [x] **Install path**: `$env:LOCALAPPDATA\Hoofy\bin` — no admin required
- [x] **PATH management**: `[Environment]::SetEnvironmentVariable("Path", ..., "User")` — persistent, user-level, updates current session too
- [x] **Locked file handling**: Catches error if `hoofy.exe` is in use and tells user to close it
- [x] **Verification**: Runs `hoofy.exe --version` and checks output
- [x] **Banner and output**: Colored output matching `install.sh` style
- [x] **Post-install instructions**: MCP config JSON for Claude Code, Cursor, VS Code Copilot, OpenCode, Gemini CLI
- [x] **One-liner**: `irm https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.ps1 | iex`
- [x] **Error handling**: `Set-StrictMode`, `$ErrorActionPreference = "Stop"`, graceful exits with informative messages
- [x] **PowerShell version guard**: `#Requires -Version 5.1`

### Consistency with install.sh
- Same banner structure (box drawing characters)
- Same post-install MCP config snippets
- Same "What's next?" section
- Same useful commands section
- Archive naming follows GoReleaser convention: `hoofy_<version>_windows_<arch>.zip`

### Not tested (requires Windows)
- Actual download and extraction on Windows
- PATH modification persistence across terminal sessions
- WoW64 detection on real 32-bit PowerShell
