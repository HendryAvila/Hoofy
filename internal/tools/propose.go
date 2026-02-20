package tools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
	"github.com/HendryAvila/sdd-hoffy/internal/pipeline"
	"github.com/HendryAvila/sdd-hoffy/internal/templates"
	"github.com/mark3labs/mcp-go/mcp"
)

// ProposeTool handles the sdd_create_proposal MCP tool.
// It transforms a raw idea into a structured proposal document.
type ProposeTool struct {
	store    config.Store
	renderer templates.Renderer
}

// NewProposeTool creates a ProposeTool with its dependencies.
func NewProposeTool(store config.Store, renderer templates.Renderer) *ProposeTool {
	return &ProposeTool{store: store, renderer: renderer}
}

// Definition returns the MCP tool definition for registration.
func (t *ProposeTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_create_proposal",
		mcp.WithDescription(
			"Transform a vague project idea into a structured proposal document. "+
				"This is Stage 1 of the SDD pipeline. The AI will analyze your idea and "+
				"produce a proposal with: Problem Statement, Target Users, Proposed Solution, "+
				"Out of Scope, Success Criteria, and Open Questions. "+
				"Requires: sdd_init_project must have been run first.",
		),
		mcp.WithString("idea",
			mcp.Required(),
			mcp.Description("Your project idea — can be as vague or detailed as you want. "+
				"Example: 'I want to build a task manager app' or 'An API that processes invoices and sends notifications'"),
		),
	)
}

// Handle processes the sdd_create_proposal tool call.
func (t *ProposeTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idea := req.GetString("idea", "")
	if idea == "" {
		return mcp.NewToolResultError("'idea' is required — tell me what you want to build"), nil
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
	if err := pipeline.RequireStage(cfg, config.StagePropose); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pipeline.MarkInProgress(cfg)

	// The LLM will fill in the sections — we provide the structure.
	// In guided mode, we add more context and examples to help non-technical users.
	data := t.buildProposalData(cfg, idea)

	content, err := t.renderer.Render(templates.Proposal, data)
	if err != nil {
		return nil, fmt.Errorf("rendering proposal: %w", err)
	}

	// Write the proposal file.
	proposalPath := config.StagePath(projectRoot, config.StagePropose)
	if err := writeStageFile(proposalPath, content); err != nil {
		return nil, fmt.Errorf("writing proposal: %w", err)
	}

	// Advance pipeline to next stage.
	if err := pipeline.Advance(cfg); err != nil {
		return nil, fmt.Errorf("advancing pipeline: %w", err)
	}

	if err := t.store.Save(projectRoot, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	response := fmt.Sprintf(
		"# Proposal Created\n\n"+
			"Saved to `sdd/proposal.md`\n\n"+
			"## Content\n\n%s\n\n"+
			"---\n\n"+
			"## Next Step\n\n"+
			"Pipeline advanced to **Stage 2: Specify**.\n\n"+
			"Review the proposal above. When you're satisfied, use `sdd_generate_requirements` "+
			"to extract formal requirements from this proposal.\n\n"+
			"**Important:** The AI should now analyze this proposal and generate structured "+
			"requirements with MoSCoW prioritization (Must/Should/Could/Won't).",
		content,
	)

	return mcp.NewToolResultText(response), nil
}

// buildProposalData creates the template data, with mode-specific content.
func (t *ProposeTool) buildProposalData(cfg *config.ProjectConfig, idea string) templates.ProposalData {
	data := templates.ProposalData{
		Name: cfg.Name,
	}

	if cfg.Mode == config.ModeGuided {
		// Guided mode: structure the idea with helpful prompts.
		data.ProblemStatement = fmt.Sprintf(
			"Based on the following idea:\n\n> %s\n\n"+
				"_Describe the core problem in 2-3 sentences. "+
				"Think: What pain point does this solve? Who is frustrated and why?_",
			idea,
		)
		data.TargetUsers = "_List 2-3 specific user types. For each: who are they, what do they need, why do they care?\n\n" +
			"Example:\n- **Freelance designers** who need to track project hours but hate complex tools\n- **Small agency owners** who need team visibility without enterprise overhead_"
		data.ProposedSolution = fmt.Sprintf(
			"_Based on the idea: \"%s\"\n\n"+
				"Describe what you're building at a HIGH level. No tech stack, no databases — "+
				"just what it does for the user.\n\n"+
				"Example: 'A simple web app where freelancers log hours per project and see weekly reports'_",
			idea,
		)
		data.OutOfScope = "_This is CRUCIAL. List 3-5 things this project will NOT do.\n\n" +
			"Example:\n- Will NOT handle invoicing or payments\n- Will NOT support offline mode in v1\n- Will NOT integrate with accounting software_"
		data.SuccessCriteria = "_How will you know this project succeeded? List 2-4 measurable outcomes.\n\n" +
			"Example:\n- Users can log time in under 10 seconds\n- Weekly report generation takes < 2 seconds\n- 80% of test users complete onboarding without help_"
		data.OpenQuestions = "_Things you're still unsure about. It's OK! List them.\n\n" +
			"Example:\n- Should we support mobile from day one?\n- Do we need user authentication or is it single-user?\n- What's the deployment target (cloud, self-hosted, both)?_"
	} else {
		// Expert mode: lean template, just the raw idea.
		data.ProblemStatement = fmt.Sprintf("Idea: %s\n\n_Define the problem statement._", idea)
		data.TargetUsers = "_Define target user personas._"
		data.ProposedSolution = "_High-level solution description._"
		data.OutOfScope = "_Explicit exclusions._"
		data.SuccessCriteria = "_Measurable success criteria._"
		data.OpenQuestions = "_Open questions and unknowns._"
	}

	return data
}
