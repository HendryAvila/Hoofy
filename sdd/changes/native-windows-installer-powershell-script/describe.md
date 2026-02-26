## What

Create an `install.ps1` PowerShell script for native Windows support. Currently, only `install.sh` (Bash) exists, which silently downloads a Linux binary when run via PowerShell + WSL — confusing Windows users.

## Why

- Users running `curl ... | bash` from PowerShell with WSL installed get a Linux binary that doesn't work on Windows natively
- Windows is a first-class platform — it deserves a native installer
- GoReleaser already builds `windows/amd64` and `windows/arm64` binaries (`.zip` format)

## Scope

- New `install.ps1` script mirroring `install.sh` behavior but for PowerShell
- Detects Windows architecture (x86_64/arm64)
- Downloads `.zip` archive from GitHub releases (GoReleaser produces `.zip` for Windows)
- Installs to a user-writable directory (e.g., `$env:LOCALAPPDATA\Hoofy\bin`)
- Adds install directory to user PATH if not present
- Verifies installation
- Update README one-liner for Windows users

## Out of Scope

- Chocolatey/Scoop/WinGet package managers (future enhancement)
- MSI installer
- Modifying the existing `install.sh`
