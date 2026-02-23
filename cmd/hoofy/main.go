// Hoofy: Spec-Driven Development MCP Server
//
// A universal MCP server that integrates with any AI coding tool
// (Claude Code, OpenCode, Gemini CLI, Codex, Cursor, VS Code Copilot)
// to guide users from vague ideas to clear, actionable specifications.
//
// Usage:
//
//	hoofy serve    # Start MCP server (stdio transport)
//	hoofy update   # Update to the latest version
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sddserver "github.com/HendryAvila/Hoofy/internal/server"
	"github.com/HendryAvila/Hoofy/internal/updater"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		if err := run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "update":
		runUpdate()
	case "--help", "-h", "help":
		printUsage()
		os.Exit(0)
	case "--version", "-v", "version":
		fmt.Printf("hoofy v%s\n", sddserver.Version)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func run() error {
	s, cleanup, err := sddserver.New()
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	defer cleanup()

	// Background version check — prints to stderr so it doesn't
	// interfere with MCP's stdio transport on stdout.
	go checkForUpdates()

	// Graceful shutdown on interrupt.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	_ = ctx // stdio server manages its own lifecycle

	return server.ServeStdio(s)
}

// checkForUpdates runs a non-blocking version check and prints a notice
// to stderr if an update is available. This runs in a goroutine during
// "serve" and is best-effort — network failures are silently ignored.
func checkForUpdates() {
	result := updater.CheckVersion(sddserver.Version)
	if result.UpdateAvailable {
		fmt.Fprintf(os.Stderr,
			"\n  📦 Update available: v%s → v%s\n"+
				"     Run: hoofy update\n"+
				"     Release: %s\n\n",
			result.CurrentVersion, result.LatestVersion, result.ReleaseURL,
		)
	}
}

// runUpdate performs a self-update to the latest version.
func runUpdate() {
	fmt.Fprintf(os.Stderr, "🔍 Checking for updates...\n")

	result := updater.CheckVersion(sddserver.Version)
	if !result.UpdateAvailable {
		fmt.Fprintf(os.Stderr, "✅ Already at the latest version (v%s)\n", result.CurrentVersion)
		return
	}

	fmt.Fprintf(os.Stderr, "📦 New version available: v%s → v%s\n", result.CurrentVersion, result.LatestVersion)
	fmt.Fprintf(os.Stderr, "⬇️  Downloading...\n")

	if err := updater.SelfUpdate(sddserver.Version); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Update failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n   You can download manually from:\n   %s\n", result.ReleaseURL)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✅ Updated to v%s!\n", result.LatestVersion)
	fmt.Fprintf(os.Stderr, "   Restart hoofy to use the new version.\n")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Hoofy v%s — Spec-Driven Development MCP Server

Usage:
  hoofy serve    Start the MCP server (stdio transport)
  hoofy update   Update to the latest version

Configuration:
  Add to your AI tool's MCP config:

  {
    "mcpServers": {
      "hoofy": {
        "command": "hoofy",
        "args": ["serve"]
      }
    }
  }

Learn more: https://github.com/HendryAvila/Hoofy
`, sddserver.Version)
}
