package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// StatusPrompt handles the sdd-status MCP prompt.
// It instructs the AI to read and present the current SDD project state.
type StatusPrompt struct{}

// NewStatusPrompt creates a StatusPrompt.
func NewStatusPrompt() *StatusPrompt {
	return &StatusPrompt{}
}

// Definition returns the MCP prompt definition for registration.
func (p *StatusPrompt) Definition() mcp.Prompt {
	return mcp.NewPrompt("sdd-status",
		mcp.WithPromptDescription(
			"Check the current status of your SDD project. "+
				"Shows pipeline progress, current stage, clarity score, "+
				"and what to do next.",
		),
	)
}

// Handle processes the sdd-status prompt request.
func (p *StatusPrompt) Handle(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "SDD Project Status",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.NewTextContent(
					"Please run `sdd_get_context` to check my SDD project status.\n\n" +
						"Then:\n" +
						"1. Show me the current pipeline state in a clear, visual format\n" +
						"2. Highlight any blockers or issues (especially clarity score if in clarify stage)\n" +
						"3. Tell me exactly what I should do next\n" +
						"4. If there are completed artifacts, give me a brief summary of each",
				),
			},
		},
	}, nil
}
