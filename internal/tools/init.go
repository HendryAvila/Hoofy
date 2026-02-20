package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// InitTool handles the sdd_init_project MCP tool.
// It creates the sdd/ directory structure and initial configuration.
type InitTool struct {
	store config.Store
}

// NewInitTool creates an InitTool with the given config store.
func NewInitTool(store config.Store) *InitTool {
	return &InitTool{store: store}
}

// Definition returns the MCP tool definition for registration.
func (t *InitTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_init_project",
		mcp.WithDescription(
			"Initialize a new SDD (Spec-Driven Development) project. "+
				"Creates the sdd/ directory with configuration and empty templates. "+
				"This is always the first step in the SDD pipeline.",
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Project name"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Brief description of what the project does"),
		),
		mcp.WithString("mode",
			mcp.Description("Interaction mode: 'guided' (step-by-step for non-technical users) or 'expert' (streamlined for developers). Defaults to 'guided'."),
			mcp.DefaultString("guided"),
			mcp.Enum("guided", "expert"),
		),
	)
}

// Handle processes the sdd_init_project tool call.
func (t *InitTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	description := req.GetString("description", "")
	modeStr := req.GetString("mode", "guided")

	if name == "" {
		return mcp.NewToolResultError("'name' is required"), nil
	}
	if description == "" {
		return mcp.NewToolResultError("'description' is required"), nil
	}

	mode := config.Mode(modeStr)
	if mode != config.ModeGuided && mode != config.ModeExpert {
		return mcp.NewToolResultError("'mode' must be 'guided' or 'expert'"), nil
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	// Guard: don't overwrite an existing project.
	if config.Exists(projectRoot) {
		return mcp.NewToolResultError(
			"SDD project already exists in this directory. Use sdd_get_context to see current state.",
		), nil
	}

	// Create directory structure.
	sddDir := config.SDDPath(projectRoot)
	dirs := []string{
		sddDir,
		filepath.Join(sddDir, "history"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Write initial config.
	cfg := config.NewProjectConfig(name, description, mode)
	if err := t.store.Save(projectRoot, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	// Build response based on mode.
	modeLabel := "Guided"
	modeHint := "I'll walk you through each step with examples and explanations."
	if mode == config.ModeExpert {
		modeLabel = "Expert"
		modeHint = "Streamlined flow — I'll ask fewer questions and accept technical input directly."
	}

	response := fmt.Sprintf(
		"# SDD Project Initialized\n\n"+
			"**Project:** %s\n"+
			"**Mode:** %s\n"+
			"**Location:** `%s/`\n\n"+
			"## What was created\n\n"+
			"```\nsdd/\n├── sdd.json          # Project configuration\n└── history/          # For completed changes\n```\n\n"+
			"## Next Step\n\n"+
			"The pipeline is now at **Stage 1: Propose**.\n\n"+
			"%s\n\n"+
			"Use `sdd_create_proposal` with your idea to generate a structured proposal.\n\n"+
			"**Tell me about your project idea** — what are you trying to build?",
		name, modeLabel, config.SDDDir, modeHint,
	)

	return mcp.NewToolResultText(response), nil
}
