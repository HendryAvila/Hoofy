# Hoofy Installer for Windows
# One-liner: irm https://raw.githubusercontent.com/HendryAvila/Hoofy/main/install.ps1 | iex
#
# This script:
#   1. Detects your architecture (x86_64/arm64)
#   2. Downloads the latest hoofy binary from GitHub
#   3. Installs it to %LOCALAPPDATA%\Hoofy\bin
#   4. Adds the install directory to your PATH
#
# Requires: PowerShell 5.1+ (Windows 10/11 built-in)

#Requires -Version 5.1

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# --- Colors and formatting ---

function Write-Info {
    param([string]$Message)
    Write-Host "  i  " -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "  +  " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warn {
    param([string]$Message)
    Write-Host "  !  " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Err {
    param([string]$Message)
    Write-Host "  x  " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "  > $Message" -ForegroundColor Cyan
}

# --- Banner ---

function Show-Banner {
    Write-Host ""
    Write-Host "  ╔═══════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "  ║                                           ║" -ForegroundColor Cyan
    Write-Host "  ║   Hoofy Installer (Windows)               ║" -ForegroundColor Cyan
    Write-Host "  ║   Spec-Driven Development MCP Server      ║" -ForegroundColor Cyan
    Write-Host "  ║                                           ║" -ForegroundColor Cyan
    Write-Host "  ╚═══════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Think first, code second. Reduce AI hallucinations" -ForegroundColor DarkGray
    Write-Host "  by writing clear specs BEFORE generating code." -ForegroundColor DarkGray
    Write-Host ""
}

# --- Architecture detection ---

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE

    switch ($arch) {
        "AMD64"  { return "amd64" }
        "x86"    {
            # Check for WoW64 (32-bit PowerShell on 64-bit OS)
            if ($env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
                return "amd64"
            }
            Write-Err "32-bit Windows is not supported."
            Write-Err "Hoofy requires a 64-bit system (x86_64 or arm64)."
            exit 1
        }
        "ARM64"  { return "arm64" }
        default {
            Write-Err "Unsupported architecture: $arch"
            Write-Err "Hoofy supports x86_64 (amd64) and arm64 only."
            exit 1
        }
    }
}

# --- Version fetching ---

function Get-LatestVersion {
    try {
        $release = Invoke-RestMethod `
            -Uri "https://api.github.com/repos/HendryAvila/Hoofy/releases/latest" `
            -Headers @{ Accept = "application/vnd.github.v3+json" } `
            -UseBasicParsing

        $tag = $release.tag_name
        if (-not $tag) {
            throw "No tag_name in response"
        }

        # Strip leading 'v' if present
        return $tag -replace "^v", ""
    }
    catch {
        Write-Err "Could not determine the latest version."
        Write-Err "Check your internet connection or visit:"
        Write-Err "https://github.com/HendryAvila/Hoofy/releases"
        exit 1
    }
}

# --- Download and install ---

function Install-Hoofy {
    param(
        [string]$Version,
        [string]$Arch,
        [string]$InstallDir
    )

    $archiveName = "hoofy_${Version}_windows_${Arch}.zip"
    $url = "https://github.com/HendryAvila/Hoofy/releases/download/v${Version}/${archiveName}"

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "hoofy-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        Write-Info "Downloading hoofy v${Version} for windows/${Arch}..."
        Write-Host "    $url" -ForegroundColor DarkGray

        $archivePath = Join-Path $tmpDir $archiveName

        try {
            Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing
        }
        catch {
            Write-Err "Download failed!"
            Write-Err "The file might not exist for your platform (windows/${Arch})."
            Write-Err "Check available downloads: https://github.com/HendryAvila/Hoofy/releases/tag/v${Version}"
            exit 1
        }

        Write-Info "Extracting..."
        $extractDir = Join-Path $tmpDir "extracted"
        Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force

        # Find hoofy.exe in extracted contents
        $binaryPath = Get-ChildItem -Path $extractDir -Filter "hoofy.exe" -Recurse -File | Select-Object -First 1

        if (-not $binaryPath) {
            Write-Err "Could not find hoofy.exe in the downloaded archive."
            exit 1
        }

        # Ensure install directory exists
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        $destination = Join-Path $InstallDir "hoofy.exe"

        # If hoofy.exe is already running or locked, inform the user
        if (Test-Path $destination) {
            try {
                Remove-Item $destination -Force
            }
            catch {
                Write-Err "Cannot replace existing hoofy.exe — it may be in use."
                Write-Err "Close any running hoofy processes and try again."
                exit 1
            }
        }

        Copy-Item -Path $binaryPath.FullName -Destination $destination -Force
        Write-Success "Installed hoofy to ${destination}"
    }
    finally {
        Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --- PATH management ---

function Add-ToPath {
    param([string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")

    # Check if already in PATH
    $paths = $userPath -split ";" | Where-Object { $_ -ne "" }
    if ($paths -contains $Dir) {
        Write-Info "Install directory is already in your PATH."
        return $true
    }

    Write-Info "Adding ${Dir} to your user PATH..."

    try {
        $newPath = "$userPath;$Dir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")

        # Also update current session
        $env:Path = "$env:Path;$Dir"

        Write-Success "Added to PATH. New terminals will pick it up automatically."
        return $true
    }
    catch {
        Write-Warn "Could not update PATH automatically."
        Write-Warn "Manually add this directory to your PATH:"
        Write-Host ""
        Write-Host "    $Dir" -ForegroundColor Cyan
        Write-Host ""
        return $false
    }
}

# --- Verification ---

function Test-Installation {
    param([string]$InstallDir)

    $binary = Join-Path $InstallDir "hoofy.exe"

    if (-not (Test-Path $binary)) {
        Write-Err "Installation verification failed — binary not found."
        exit 1
    }

    try {
        $versionOutput = & $binary --version 2>&1
        if ($versionOutput -match "hoofy") {
            Write-Success "Verification passed: $versionOutput"
        }
        else {
            Write-Warn "Binary exists but version check returned unexpected output."
            Write-Warn "Output: $versionOutput"
        }
    }
    catch {
        Write-Warn "Binary exists but could not run version check."
    }
}

# --- Main ---

function Main {
    Show-Banner

    Write-Step "Detecting system"
    $arch = Get-Arch
    Write-Success "Detected: windows/${arch}"

    Write-Step "Fetching latest version"
    $version = Get-LatestVersion
    Write-Success "Latest version: v${version}"

    Write-Step "Installing"
    $installDir = Join-Path $env:LOCALAPPDATA "Hoofy" "bin"
    Install-Hoofy -Version $version -Arch $arch -InstallDir $installDir

    Write-Step "Configuring PATH"
    Add-ToPath -Dir $installDir | Out-Null

    Write-Step "Verifying installation"
    Test-Installation -InstallDir $installDir

    # Done!
    Write-Host ""
    Write-Host "  ╔═══════════════════════════════════════════╗" -ForegroundColor Green
    Write-Host "  ║                                           ║" -ForegroundColor Green
    Write-Host "  ║   Hoofy installed successfully!           ║" -ForegroundColor Green
    Write-Host "  ║                                           ║" -ForegroundColor Green
    Write-Host "  ╚═══════════════════════════════════════════╝" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Next: Add Hoofy to your AI tool's MCP config:" -ForegroundColor White
    Write-Host ""
    Write-Host "  Claude Code (.claude/settings.json), Cursor, VS Code Copilot (.vscode/mcp.json)," -ForegroundColor DarkGray
    Write-Host "  Gemini CLI:" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host '  {' -ForegroundColor Cyan
    Write-Host '    "mcpServers": {' -ForegroundColor Cyan
    Write-Host '      "hoofy": {' -ForegroundColor Cyan
    Write-Host '        "command": "hoofy",' -ForegroundColor Cyan
    Write-Host '        "args": ["serve"]' -ForegroundColor Cyan
    Write-Host '      }' -ForegroundColor Cyan
    Write-Host '    }' -ForegroundColor Cyan
    Write-Host '  }' -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  OpenCode (~/.config/opencode/opencode.json, inside the `"mcp`" key):" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host '  "hoofy": {' -ForegroundColor Cyan
    Write-Host '    "type": "local",' -ForegroundColor Cyan
    Write-Host '    "command": ["hoofy", "serve"],' -ForegroundColor Cyan
    Write-Host '    "enabled": true' -ForegroundColor Cyan
    Write-Host '  }' -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  What's next?" -ForegroundColor White
    Write-Host ""
    Write-Host "    1. Add the JSON snippet above to your AI tool's MCP config"
    Write-Host "    2. Use the " -NoNewline; Write-Host "/sdd-start" -ForegroundColor White -NoNewline; Write-Host " prompt to begin"
    Write-Host "    3. Describe your idea — Hoofy will guide you"
    Write-Host ""
    Write-Host "  Useful commands:" -ForegroundColor White
    Write-Host ""
    Write-Host "    hoofy serve  " -ForegroundColor Cyan -NoNewline; Write-Host "   Start the MCP server"
    Write-Host "    hoofy update " -ForegroundColor Cyan -NoNewline; Write-Host "   Update to the latest version"
    Write-Host "    hoofy --help " -ForegroundColor Cyan -NoNewline; Write-Host "   Show help"
    Write-Host ""
    Write-Host "  Docs: https://github.com/HendryAvila/Hoofy" -ForegroundColor DarkGray
    Write-Host "  Star if you find it useful!" -ForegroundColor DarkGray
    Write-Host ""
}

# Run
Main
