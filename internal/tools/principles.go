package tools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/Hoofy/internal/config"
	"github.com/HendryAvila/Hoofy/internal/pipeline"
	"github.com/HendryAvila/Hoofy/internal/templates"
	"github.com/mark3labs/mcp-go/mcp"
)

// PrinciplesTool handles the sdd_create_principles MCP tool.
// It saves a principles document with golden invariants, coding standards,
// and domain truths provided by the AI.
type PrinciplesTool struct {
	store    config.Store
	renderer templates.Renderer
	bridge   StageObserver
}

// NewPrinciplesTool creates a PrinciplesTool with its dependencies.
func NewPrinciplesTool(store config.Store, renderer templates.Renderer) *PrinciplesTool {
	return &PrinciplesTool{store: store, renderer: renderer}
}

// SetBridge injects an optional StageObserver that gets notified
// when the principles stage completes. Nil is safe (disables bridge).
func (t *PrinciplesTool) SetBridge(obs StageObserver) { t.bridge = obs }

// Definition returns the MCP tool definition for registration.
func (t *PrinciplesTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_create_principles",
		mcp.WithDescription(
			"Save project principles — golden invariants that must NEVER be violated. "+
				"This is Stage 1 of the SDD pipeline. "+
				"IMPORTANT: Before calling this tool, the AI MUST discuss with the user what rules are sacred "+
				"in their project. Ask: 'What should NEVER be broken, no matter what?' "+
				"Pass the ACTUAL principles (not placeholders). "+
				"Requires: sdd_init_project must have been run first.",
		),
		mcp.WithString("principles",
			mcp.Required(),
			mcp.Description("Golden invariants — rules that must NEVER be broken. "+
				"These are the project's non-negotiable beliefs. Use markdown list format. "+
				"Example: '- Never store passwords in plain text\\n"+
				"- All API responses must include correlation IDs\\n"+
				"- No business logic in controllers — domain layer only'"),
		),
		mcp.WithString("coding_standards",
			mcp.Description("Coding conventions and standards the project follows. Optional but recommended. "+
				"Use markdown list format. "+
				"Example: '- Use conventional commits (feat:, fix:, refactor:)\\n"+
				"- All functions must have JSDoc/godoc comments\\n"+
				"- No magic numbers — use named constants'"),
		),
		mcp.WithString("domain_truths",
			mcp.Description("Domain-specific truths that are always true in this project's context. Optional. "+
				"Use markdown list format. "+
				"Example: '- A user can have at most one active subscription\\n"+
				"- Prices are always in cents (integer), never floats\\n"+
				"- All timestamps are UTC'"),
		),
	)
}

// Handle processes the sdd_create_principles tool call.
func (t *PrinciplesTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	principles := req.GetString("principles", "")
	codingStandards := req.GetString("coding_standards", "")
	domainTruths := req.GetString("domain_truths", "")

	// Validate required fields.
	if principles == "" {
		return mcp.NewToolResultError("'principles' is required — what rules must NEVER be broken in this project?"), nil
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	cfg, err := t.store.Load(projectRoot)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate we're at the right stage.
	if err := pipeline.RequireStage(cfg, config.StagePrinciples); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pipeline.MarkInProgress(cfg)

	// Build principles with REAL content from the AI.
	data := templates.PrinciplesData{
		Name:            cfg.Name,
		Principles:      principles,
		CodingStandards: codingStandards,
		DomainTruths:    domainTruths,
	}

	content, err := t.renderer.Render(templates.Principles, data)
	if err != nil {
		return nil, fmt.Errorf("rendering principles: %w", err)
	}

	// Write the principles file.
	principlesPath := config.StagePath(projectRoot, config.StagePrinciples)
	if err := writeStageFile(principlesPath, content); err != nil {
		return nil, fmt.Errorf("writing principles: %w", err)
	}

	// Advance pipeline to next stage.
	if err := pipeline.Advance(cfg); err != nil {
		return nil, fmt.Errorf("advancing pipeline: %w", err)
	}

	if err := t.store.Save(projectRoot, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	notifyObserver(t.bridge, cfg.Name, config.StagePrinciples, content)

	response := fmt.Sprintf(
		"# Principles Established\n\n"+
			"Saved to `%s/principles.md`\n\n"+
			"## Content\n\n%s\n\n"+
			"---\n\n"+
			"## Next Step\n\n"+
			"Pipeline advanced to **Stage 2: Charter**.\n\n"+
			"Now let's define the project charter — the scope, vision, stakeholders, and boundaries.\n\n"+
			"Call `sdd_create_charter` with the project's problem statement, target users, and proposed solution.",
		config.DocsDir, content,
	)

	return mcp.NewToolResultText(response), nil
}
