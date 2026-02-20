package tools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
	"github.com/HendryAvila/sdd-hoffy/internal/pipeline"
	"github.com/HendryAvila/sdd-hoffy/internal/templates"
	"github.com/mark3labs/mcp-go/mcp"
)

// SpecifyTool handles the sdd_generate_requirements MCP tool.
// It extracts formal requirements from the proposal document.
type SpecifyTool struct {
	store    config.Store
	renderer templates.Renderer
}

// NewSpecifyTool creates a SpecifyTool with its dependencies.
func NewSpecifyTool(store config.Store, renderer templates.Renderer) *SpecifyTool {
	return &SpecifyTool{store: store, renderer: renderer}
}

// Definition returns the MCP tool definition for registration.
func (t *SpecifyTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_generate_requirements",
		mcp.WithDescription(
			"Extract formal requirements from the proposal document. "+
				"This is Stage 2 of the SDD pipeline. The AI reads the proposal (sdd/proposal.md) "+
				"and produces structured requirements using MoSCoW prioritization "+
				"(Must Have, Should Have, Could Have, Won't Have). "+
				"Each requirement gets a unique ID (FR-001, NFR-001) for traceability. "+
				"Requires: sdd_create_proposal must have been run first.",
		),
		mcp.WithString("additional_context",
			mcp.Description("Optional extra context, constraints, or preferences to consider when generating requirements"),
		),
	)
}

// Handle processes the sdd_generate_requirements tool call.
func (t *SpecifyTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	additionalContext := req.GetString("additional_context", "")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	cfg, err := t.store.Load(projectRoot)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate pipeline stage.
	if err := pipeline.RequireStage(cfg, config.StageSpecify); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Read the proposal to base requirements on.
	proposalPath := config.StagePath(projectRoot, config.StagePropose)
	proposal, err := readStageFile(proposalPath)
	if err != nil {
		return nil, fmt.Errorf("reading proposal: %w", err)
	}
	if proposal == "" {
		return mcp.NewToolResultError("proposal.md is empty — run sdd_create_proposal first"), nil
	}

	pipeline.MarkInProgress(cfg)

	// Build requirements template data.
	data := t.buildRequirementsData(cfg, proposal, additionalContext)

	content, err := t.renderer.Render(templates.Requirements, data)
	if err != nil {
		return nil, fmt.Errorf("rendering requirements: %w", err)
	}

	// Write requirements file.
	reqPath := config.StagePath(projectRoot, config.StageSpecify)
	if err := writeStageFile(reqPath, content); err != nil {
		return nil, fmt.Errorf("writing requirements: %w", err)
	}

	// Advance pipeline.
	if err := pipeline.Advance(cfg); err != nil {
		return nil, fmt.Errorf("advancing pipeline: %w", err)
	}

	if err := t.store.Save(projectRoot, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	response := fmt.Sprintf(
		"# Requirements Generated\n\n"+
			"Saved to `sdd/requirements.md`\n\n"+
			"## Content\n\n%s\n\n"+
			"---\n\n"+
			"## Next Step\n\n"+
			"Pipeline advanced to **Stage 3: Clarify (Clarity Gate)**.\n\n"+
			"This is the MOST IMPORTANT stage. The AI will now analyze these requirements "+
			"for ambiguities and generate clarifying questions.\n\n"+
			"Use `sdd_clarify` to start the Clarity Gate process. The pipeline cannot proceed "+
			"until the clarity score reaches the threshold (%d for %s mode).\n\n"+
			"**Why this matters:** Ambiguous requirements are the #1 cause of AI hallucinations. "+
			"The clearer your specs, the better AI-generated code will be.",
		content, pipeline.ClarityThreshold(cfg.Mode), cfg.Mode,
	)

	return mcp.NewToolResultText(response), nil
}

// buildRequirementsData creates template data with mode-specific guidance.
func (t *SpecifyTool) buildRequirementsData(cfg *config.ProjectConfig, proposal, additionalContext string) templates.RequirementsData {
	data := templates.RequirementsData{
		Name: cfg.Name,
	}

	contextNote := ""
	if additionalContext != "" {
		contextNote = fmt.Sprintf("\n\n**Additional context:** %s", additionalContext)
	}

	if cfg.Mode == config.ModeGuided {
		data.MustHave = fmt.Sprintf(
			"_Based on the proposal below, list the absolute minimum features needed for this to work.%s_\n\n"+
				"_Format each requirement as:_\n"+
				"- **FR-001**: _[Clear, testable requirement]_\n\n"+
				"_Example:_\n"+
				"- **FR-001**: Users can create an account with email and password\n"+
				"- **FR-002**: Users can log time entries with project, duration, and description\n\n"+
				"---\n\n"+
				"**Source proposal:**\n\n%s",
			contextNote, proposal,
		)
		data.ShouldHave = "_Nice-to-have features that add significant value but aren't blocking launch._\n\n" +
			"- **FR-0XX**: ..."
		data.CouldHave = "_Features that would be great but can wait for a future version._\n\n" +
			"- **FR-0XX**: ..."
		data.WontHave = "_Features explicitly excluded from THIS version. Being explicit prevents scope creep._\n\n" +
			"- **FR-0XX**: ..."
		data.NonFunctional = "_Performance, security, usability requirements._\n\n" +
			"_Format:_\n" +
			"- **NFR-001**: _[Measurable constraint]_\n\n" +
			"_Example:_\n" +
			"- **NFR-001**: Page load time must be under 2 seconds on 3G\n" +
			"- **NFR-002**: All user data must be encrypted at rest"
		data.Constraints = "_Technical or business limitations._\n\n" +
			"_Example:_\n" +
			"- Must run on Node.js 20+\n" +
			"- Budget limited to free-tier cloud services"
		data.Assumptions = "_What we assume to be true. Flag these — if they change, so do requirements._"
		data.Dependencies = "_External systems, APIs, or services we need._"
	} else {
		data.MustHave = fmt.Sprintf(
			"_Extract must-have requirements from proposal. Use FR-XXX IDs.%s_\n\n"+
				"**Source:**\n\n%s",
			contextNote, proposal,
		)
		data.ShouldHave = "_Should-have requirements (FR-XXX)._"
		data.CouldHave = "_Could-have requirements (FR-XXX)._"
		data.WontHave = "_Won't-have this version (FR-XXX)._"
		data.NonFunctional = "_Non-functional requirements (NFR-XXX)._"
		data.Constraints = "_Technical and business constraints._"
		data.Assumptions = "_Assumptions._"
		data.Dependencies = "_Dependencies._"
	}

	return data
}
