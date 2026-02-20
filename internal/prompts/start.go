// Package prompts implements MCP prompt handlers for the SDD pipeline.
//
// MCP prompts are user-triggered workflows (like slash commands) that
// instruct the AI to execute a specific sequence. Unlike tools (which
// the AI calls), prompts are initiated by the user.
package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// StartPrompt handles the sdd-start MCP prompt.
// It guides the AI to initialize a new SDD project and begin the pipeline.
type StartPrompt struct{}

// NewStartPrompt creates a StartPrompt.
func NewStartPrompt() *StartPrompt {
	return &StartPrompt{}
}

// Definition returns the MCP prompt definition for registration.
func (p *StartPrompt) Definition() mcp.Prompt {
	return mcp.NewPrompt("sdd-start",
		mcp.WithPromptDescription(
			"Start a new Spec-Driven Development project. "+
				"This will guide you through initializing the SDD pipeline, "+
				"from setting up the project to creating your first proposal.",
		),
		mcp.WithArgument("project_name",
			mcp.ArgumentDescription("Name of your project"),
		),
		mcp.WithArgument("mode",
			mcp.ArgumentDescription(
				"Interaction mode: 'guided' (step-by-step for beginners) or 'expert' (streamlined for developers). Default: guided",
			),
		),
	)
}

// Handle processes the sdd-start prompt request.
func (p *StartPrompt) Handle(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	projectName := "my-project"
	if args := req.Params.Arguments; args != nil {
		if name, ok := args["project_name"]; ok && name != "" {
			projectName = name
		}
	}

	mode := "guided"
	if args := req.Params.Arguments; args != nil {
		if m, ok := args["mode"]; ok && m != "" {
			mode = m
		}
	}

	modeExplanation := ""
	if mode == "guided" {
		modeExplanation = "You're in **Guided mode** — I'll walk you through each step with examples and explanations. Perfect if you're new to this or not super technical."
	} else {
		modeExplanation = "You're in **Expert mode** — streamlined flow with less hand-holding. I'll get straight to the point."
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Start SDD project: %s", projectName),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.NewTextContent(fmt.Sprintf(
					"I want to start a new Spec-Driven Development project called '%s' in %s mode.\n\n"+
						"Please:\n"+
						"1. Run `sdd_init_project` with name='%s', description (ask me for a brief description), and mode='%s'\n"+
						"2. After init, ask me to describe my project idea\n"+
						"3. Once I describe my idea, run `sdd_create_proposal` with my idea\n"+
						"4. Guide me through the rest of the SDD pipeline step by step\n\n"+
						"%s",
					projectName, mode, projectName, mode, modeExplanation,
				)),
			},
		},
	}, nil
}
