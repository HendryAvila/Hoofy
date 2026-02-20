// Package server wires all MCP components and creates the server instance.
//
// This is the composition root (DIP): it creates concrete implementations
// and injects them into the tools/prompts/resources that depend on abstractions.
// No business logic lives here — only wiring.
package server

import (
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
	"github.com/HendryAvila/sdd-hoffy/internal/prompts"
	"github.com/HendryAvila/sdd-hoffy/internal/resources"
	"github.com/HendryAvila/sdd-hoffy/internal/templates"
	"github.com/HendryAvila/sdd-hoffy/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set at build time via ldflags.
var Version = "dev"

// New creates and configures the MCP server with all tools, prompts,
// and resources registered. This is the single place where all
// dependencies are resolved.
func New() (*server.MCPServer, error) {
	// --- Create shared dependencies ---

	store := config.NewFileStore()

	renderer, err := templates.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("creating template renderer: %w", err)
	}

	// --- Create the MCP server ---

	s := server.NewMCPServer(
		"sdd-hoffy",
		Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
		server.WithInstructions(serverInstructions()),
	)

	// --- Register tools ---

	initTool := tools.NewInitTool(store)
	s.AddTool(initTool.Definition(), initTool.Handle)

	proposeTool := tools.NewProposeTool(store, renderer)
	s.AddTool(proposeTool.Definition(), proposeTool.Handle)

	specifyTool := tools.NewSpecifyTool(store, renderer)
	s.AddTool(specifyTool.Definition(), specifyTool.Handle)

	clarifyTool := tools.NewClarifyTool(store, renderer)
	s.AddTool(clarifyTool.Definition(), clarifyTool.Handle)

	contextTool := tools.NewContextTool(store)
	s.AddTool(contextTool.Definition(), contextTool.Handle)

	// --- Register prompts ---

	startPrompt := prompts.NewStartPrompt()
	s.AddPrompt(startPrompt.Definition(), startPrompt.Handle)

	statusPrompt := prompts.NewStatusPrompt()
	s.AddPrompt(statusPrompt.Definition(), statusPrompt.Handle)

	// --- Register resources ---

	resourceHandler := resources.NewHandler(store)
	s.AddResource(resourceHandler.StatusResource(), resourceHandler.HandleStatus)

	return s, nil
}

// serverInstructions returns the system instructions that tell the AI
// how to use SDD-Hoffy effectively.
func serverInstructions() string {
	return `You have access to SDD-Hoffy, a Spec-Driven Development tool.

## What is SDD?
Spec-Driven Development reduces AI hallucinations by forcing clear specifications 
BEFORE writing code. Ambiguous requirements are the #1 cause of bad AI-generated code.

## Pipeline
SDD follows a sequential pipeline:
1. INIT → Set up the project
2. PROPOSE → Transform a vague idea into a structured proposal
3. SPECIFY → Extract formal requirements with MoSCoW prioritization
4. CLARIFY → The Clarity Gate: resolve ALL ambiguities before proceeding
5. DESIGN → Technical architecture (coming soon)
6. TASKS → Atomic task breakdown (coming soon)
7. VALIDATE → Cross-artifact consistency check (coming soon)

## How to Use
1. Start with sdd_init_project or the /sdd-start prompt
2. Follow the pipeline in order — each tool tells you what to do next
3. The Clarity Gate (Stage 3) is the most important: it BLOCKS progress 
   until requirements are unambiguous
4. Use sdd_get_context anytime to check project status

## Modes
- Guided: More questions, examples, step-by-step. For non-technical users.
- Expert: Streamlined, direct. For experienced developers.

## Important Rules
- NEVER skip the Clarity Gate
- ALWAYS follow the pipeline order
- Each requirement must have a unique ID (FR-001, NFR-001)
- Be specific — "users" is not a valid target audience`
}
