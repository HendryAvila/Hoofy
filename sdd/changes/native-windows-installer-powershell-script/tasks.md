## Tasks

### TASK-001: Create install.ps1 PowerShell script
**Description**: Create the native Windows installer script mirroring install.sh functionality
**Acceptance Criteria**:
- [ ] Detects Windows architecture (AMD64/ARM64) via `$env:PROCESSOR_ARCHITECTURE`
- [ ] Fetches latest version from GitHub API (`Invoke-RestMethod`)
- [ ] Downloads `.zip` archive: `hoofy_<version>_windows_<arch>.zip`
- [ ] Extracts `hoofy.exe` to `$env:LOCALAPPDATA\Hoofy\bin\`
- [ ] Adds install dir to user PATH if not present (via `[Environment]::SetEnvironmentVariable`)
- [ ] Verifies installation by running `hoofy.exe --version`
- [ ] Has colored output and banner matching install.sh style
- [ ] Handles errors gracefully (no release found, download failed, etc.)
- [ ] One-liner: `irm https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.ps1 | iex`
- [ ] Documents in post-install output how to add to MCP config (same JSON snippet as install.sh)
