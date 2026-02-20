package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
	"github.com/HendryAvila/sdd-hoffy/internal/pipeline"
	"github.com/HendryAvila/sdd-hoffy/internal/templates"
	"github.com/mark3labs/mcp-go/mcp"
)

// ClarifyTool handles the sdd_clarify MCP tool.
// This is the CORE of SDD-Hoffy: the Clarity Gate that forces
// disambiguation before proceeding to implementation.
type ClarifyTool struct {
	store    config.Store
	renderer templates.Renderer
}

// NewClarifyTool creates a ClarifyTool with its dependencies.
func NewClarifyTool(store config.Store, renderer templates.Renderer) *ClarifyTool {
	return &ClarifyTool{store: store, renderer: renderer}
}

// Definition returns the MCP tool definition for registration.
func (t *ClarifyTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_clarify",
		mcp.WithDescription(
			"Run the Clarity Gate analysis on current requirements. "+
				"This is Stage 3 of the SDD pipeline — the MOST IMPORTANT stage. "+
				"It analyzes requirements for ambiguities across 8 dimensions "+
				"(target users, core functionality, data model, integrations, edge cases, "+
				"security, scale, scope boundaries) and generates clarifying questions. "+
				"Call without 'answers' to get questions. Call with 'answers' to submit responses. "+
				"The pipeline cannot advance until the clarity score meets the threshold. "+
				"Requires: sdd_generate_requirements must have been run first.",
		),
		mcp.WithString("answers",
			mcp.Description(
				"Answers to previously generated clarity questions. "+
					"Format: one answer per line, matching the order of questions asked. "+
					"Leave empty to generate new questions based on current requirements.",
			),
		),
		mcp.WithString("dimension_scores",
			mcp.Description(
				"AI-assessed scores for each clarity dimension, as a comma-separated list of "+
					"dimension_name:score pairs (score 0-100). "+
					"Example: 'target_users:80,core_functionality:90,edge_cases:40'. "+
					"The AI should evaluate how well the requirements + answers address each dimension.",
			),
		),
	)
}

// Handle processes the sdd_clarify tool call.
func (t *ClarifyTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	answers := req.GetString("answers", "")
	dimensionScores := req.GetString("dimension_scores", "")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	cfg, err := t.store.Load(projectRoot)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate pipeline stage.
	if err := pipeline.RequireStage(cfg, config.StageClarify); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Read requirements for analysis.
	reqPath := config.StagePath(projectRoot, config.StageSpecify)
	requirements, err := readStageFile(reqPath)
	if err != nil {
		return nil, fmt.Errorf("reading requirements: %w", err)
	}
	if requirements == "" {
		return mcp.NewToolResultError("requirements.md is empty — run sdd_generate_requirements first"), nil
	}

	pipeline.MarkInProgress(cfg)

	threshold := pipeline.ClarityThreshold(cfg.Mode)

	// Branch: generating questions vs processing answers.
	if answers == "" {
		return t.generateQuestions(cfg, requirements, projectRoot, threshold)
	}

	return t.processAnswers(cfg, requirements, answers, dimensionScores, projectRoot, threshold)
}

// generateQuestions analyzes requirements and produces clarifying questions.
func (t *ClarifyTool) generateQuestions(
	cfg *config.ProjectConfig,
	requirements string,
	projectRoot string,
	threshold int,
) (*mcp.CallToolResult, error) {
	dimensions := pipeline.DefaultDimensions()

	// Build the analysis prompt for the AI.
	var sb strings.Builder
	sb.WriteString("# Clarity Gate Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Mode:** %s | **Threshold:** %d/100\n\n", cfg.Mode, threshold))
	sb.WriteString("## Requirements Under Analysis\n\n")
	sb.WriteString(requirements)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("## Dimensions to Evaluate\n\n")
	sb.WriteString("Analyze the requirements above for ambiguity across these dimensions. ")
	sb.WriteString("For each dimension that has gaps, generate 1-2 specific questions.\n\n")

	for _, d := range dimensions {
		sb.WriteString(fmt.Sprintf("### %s (weight: %d/10)\n", d.Name, d.Weight))
		sb.WriteString(fmt.Sprintf("_%s_\n\n", d.Description))
	}

	sb.WriteString("---\n\n")
	sb.WriteString("## Instructions for the AI\n\n")
	sb.WriteString("1. Read the requirements carefully\n")
	sb.WriteString("2. For each dimension, assess whether the requirements clearly address it\n")
	sb.WriteString("3. Generate up to 5 total questions targeting the MOST ambiguous areas\n")
	sb.WriteString("4. Questions should be specific and answerable (not open-ended philosophy)\n")
	sb.WriteString("5. Present questions to the user and ask them to answer\n")
	sb.WriteString("6. After receiving answers, call `sdd_clarify` again with the answers and your dimension scores\n\n")

	if cfg.Mode == config.ModeGuided {
		sb.WriteString("**Guided mode:** Ask questions in simple, non-technical language. ")
		sb.WriteString("Provide examples of good answers. Be encouraging.\n")
	} else {
		sb.WriteString("**Expert mode:** Be direct. Technical language is fine. ")
		sb.WriteString("Focus on the most critical gaps.\n")
	}

	// Read existing clarifications to show history.
	clarifyPath := config.StagePath(projectRoot, config.StageClarify)
	existing, _ := readStageFile(clarifyPath)
	if existing != "" {
		sb.WriteString("\n---\n\n## Previous Clarification Rounds\n\n")
		sb.WriteString(existing)
		sb.WriteString("\n\n_Build on previous rounds. Don't re-ask answered questions._\n")
	}

	if err := t.store.Save(projectRoot, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// processAnswers records answers, updates clarity score, and checks the gate.
func (t *ClarifyTool) processAnswers(
	cfg *config.ProjectConfig,
	requirements, answers, dimensionScores string,
	projectRoot string,
	threshold int,
) (*mcp.CallToolResult, error) {
	// Parse dimension scores if provided.
	dimensions := pipeline.DefaultDimensions()
	if dimensionScores != "" {
		parseDimensionScores(dimensionScores, dimensions)
	}

	// Calculate new clarity score.
	newScore := pipeline.CalculateScore(dimensions)
	cfg.ClarityScore = newScore

	// Read existing clarifications and append this round.
	clarifyPath := config.StagePath(projectRoot, config.StageClarify)
	existing, _ := readStageFile(clarifyPath)

	iteration := cfg.StageStatus[config.StageClarify].Iterations
	roundContent := fmt.Sprintf(
		"\n### Round %d\n\n**Answers:**\n\n%s\n\n**Clarity Score after this round:** %d/100\n",
		iteration, answers, newScore,
	)

	updatedContent := existing + roundContent
	if err := writeStageFile(clarifyPath, updatedContent); err != nil {
		return nil, fmt.Errorf("writing clarifications: %w", err)
	}

	// Render the full clarifications document.
	status := "IN PROGRESS"
	if newScore >= threshold {
		status = "PASSED"
	}

	fullDoc, err := t.renderer.Render(templates.Clarifications, templates.ClarificationsData{
		Name:         cfg.Name,
		ClarityScore: newScore,
		Mode:         string(cfg.Mode),
		Threshold:    threshold,
		Status:       status,
		Rounds:       updatedContent,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering clarifications: %w", err)
	}

	if err := writeStageFile(clarifyPath, fullDoc); err != nil {
		return nil, fmt.Errorf("writing clarifications: %w", err)
	}

	// Check if we passed the gate.
	var response string
	if newScore >= threshold {
		// Gate passed! Advance pipeline.
		if err := pipeline.Advance(cfg); err != nil {
			return nil, fmt.Errorf("advancing pipeline: %w", err)
		}

		response = fmt.Sprintf(
			"# Clarity Gate PASSED\n\n"+
				"**Score:** %d/100 (threshold: %d)\n\n"+
				"Your requirements are now clear enough to proceed.\n\n"+
				"## Next Step\n\n"+
				"Pipeline advanced to **Stage 4: Design**.\n\n"+
				"The AI can now create a technical design based on these well-defined requirements. "+
				"Use `sdd_get_context` to review all artifacts before proceeding.",
			newScore, threshold,
		)
	} else {
		// Need more clarification.
		uncovered := pipeline.UncoveredDimensions(dimensions)
		var uncoveredNames []string
		for _, d := range uncovered {
			uncoveredNames = append(uncoveredNames, d.Name)
		}

		response = fmt.Sprintf(
			"# Clarity Gate: More Clarification Needed\n\n"+
				"**Score:** %d/100 (need %d to pass)\n\n"+
				"## Weak Areas\n\n"+
				"These dimensions still need attention: %s\n\n"+
				"## What to Do\n\n"+
				"Call `sdd_clarify` again (without answers) to get the next round of questions "+
				"targeting these weak areas.",
			newScore, threshold, strings.Join(uncoveredNames, ", "),
		)
	}

	if err := t.store.Save(projectRoot, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	return mcp.NewToolResultText(response), nil
}

// parseDimensionScores parses "name:score,name:score" format into dimensions.
func parseDimensionScores(input string, dimensions []pipeline.ClarityDimension) {
	pairs := strings.Split(input, ",")
	scoreMap := make(map[string]int)

	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		var score int
		if _, err := fmt.Sscanf(parts[1], "%d", &score); err == nil {
			if score < 0 {
				score = 0
			}
			if score > 100 {
				score = 100
			}
			scoreMap[name] = score
		}
	}

	for i := range dimensions {
		if score, ok := scoreMap[dimensions[i].Name]; ok {
			dimensions[i].Score = score
			dimensions[i].Covered = score > 30 // Consider "covered" if score > 30
		}
	}
}
