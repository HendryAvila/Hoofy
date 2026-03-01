package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// StageGuidePrompt serves on-demand stage-by-stage workflow instructions.
// This is the "cold" counterpart to the hot pipeline overview in serverInstructions().
// Content moved here: Pre-Pipeline Exploration + Stage-by-Stage Workflow (stages 1-7).
type StageGuidePrompt struct{}

// NewStageGuidePrompt creates a StageGuidePrompt.
func NewStageGuidePrompt() *StageGuidePrompt {
	return &StageGuidePrompt{}
}

// Definition returns the MCP prompt definition for registration.
func (p *StageGuidePrompt) Definition() mcp.Prompt {
	return mcp.NewPrompt("sdd-stage-guide",
		mcp.WithPromptDescription(
			"Detailed stage-by-stage workflow for the SDD pipeline. "+
				"Includes research references (IEEE 29148, BRG, EARS, SOLID, Fowler) "+
				"and instructions for each stage from Propose through Validate.",
		),
		mcp.WithArgument("stage",
			mcp.ArgumentDescription(
				"Optional: focus on a specific stage (propose, specify, business-rules, clarify, design, tasks, validate, explore)",
			),
		),
	)
}

// Handle returns the stage-by-stage workflow instructions.
func (p *StageGuidePrompt) Handle(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	focusHint := ""
	if args := req.Params.Arguments; args != nil {
		if stage, ok := args["stage"]; ok && stage != "" {
			focusHint = fmt.Sprintf("\n\n> **Focus on stage: %s** — prioritize the instructions for this stage.\n", stage)
		}
	}

	return &mcp.GetPromptResult{
		Description: "SDD Stage-by-Stage Workflow Guide",
		Messages: []mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.NewTextContent(focusHint + stageGuideContent),
			},
		},
	}, nil
}

const stageGuideContent = `# SDD Stage-by-Stage Workflow Guide

## PRE-PIPELINE EXPLORATION

Before starting any pipeline (project or change), use sdd_explore to capture
the user's context, goals, and constraints. This ensures every subsequent
stage is informed by structured pre-work rather than ad-hoc conversation.

### When to Use sdd_explore
- Before sdd_init_project: Capture project vision, user constraints, tech preferences
- Before sdd_change: Capture change context, help determine type and size
- During any open-ended discussion about features, architecture, or direction
- When the user is "thinking out loud" and you want to preserve their reasoning

### How to Use sdd_explore
1. Discuss the idea with the user — ask clarifying questions
2. Call sdd_explore with structured categories:
   - goals: What they want to achieve
   - constraints: Limitations (technical, business, time)
   - preferences: Architecture, tech stack, patterns they prefer
   - unknowns: Things they're unsure about
   - decisions: Choices already made
   - context: Any additional context
3. The tool saves to memory with topic_key upsert — call it again as context evolves
4. When ready, start the pipeline — retrieve explore context with mem_search(type=explore)
   to inform your proposal/spec/design content
5. The response includes type/size suggestions based on keywords — use these as hints

### Important
- sdd_explore is OPTIONAL — it never blocks pipeline advancement
- It uses memory, not the pipeline state machine — no stage gates
- Call it multiple times as the conversation evolves — it upserts, not duplicates
- The type/size suggestion is a HINT — the user decides

## Stage 1: Propose (Research: IREB Elicitation Techniques)

1. Ask the user about their project idea
2. Use IREB elicitation techniques — ask about CONTEXT, not just features:
   - Instead of "What features do you want?" ask "What are the top 3 tasks
     your primary user performs daily that this tool should improve?"
   - Instead of "Who are the users?" ask "Describe someone who would use this
     in their first week — what's their role, frustration, and goal?"
3. Generate content for ALL sections:
   - problem_statement: The core problem (2-3 sentences)
   - target_users: 2-3 specific user personas with needs
   - proposed_solution: High-level description (NO tech details)
   - out_of_scope: 3-5 explicit exclusions
   - success_criteria: 2-4 measurable outcomes
   - open_questions: Remaining unknowns
4. Call sdd_create_proposal with all sections filled in

## Stage 2: Specify (Research: IEEE 29148 Quality Attributes)

1. Read the proposal from sdd/proposal.md (use sdd_get_context if needed)
2. Extract formal requirements using MoSCoW prioritization
3. Each requirement gets a unique ID (FR-001 for functional, NFR-001 for non-functional)
4. Apply IEEE 29148 quality attributes — each requirement MUST be:
   - Necessary: traceable to a user need from the proposal
   - Unambiguous: one interpretation only (no "etc.", "and/or", "appropriate")
   - Verifiable: testable with a concrete condition
   - Consistent: no contradictions with other requirements
5. Call sdd_generate_requirements with real requirements content

## Stage 3: Business Rules (Research: BRG Taxonomy, Business Rules Manifesto, DDD)

1. Read the requirements (use sdd_get_context stage=requirements)
2. For each requirement, ask: "Is there an implicit business rule here?"
3. Extract rules into four categories (BRG taxonomy — Business Rules Group):
   - Definitions: What do domain terms MEAN? Build a Ubiquitous Language
     (DDD, Eric Evans — every term must have ONE precise meaning)
   - Facts: What relationships between terms are ALWAYS true?
   - Constraints: What behavior is NOT allowed? Use declarative format:
     "When <condition> Then <imposition> [Otherwise <consequence>]"
     (Business Rules Manifesto, Ronald Ross v2.0)
   - Derivations: What knowledge is COMPUTED from other rules?
4. Present the extracted rules to the user for validation
5. Call sdd_create_business_rules with the validated content
   Required params: definitions, facts, constraints
   Optional params: derivations, glossary

## Stage 4: Clarify — Clarity Gate (Research: EARS, Femmer et al. 2017)

1. Call sdd_clarify WITHOUT answers to get the analysis framework
2. Analyze the requirements AND business rules across all 8 dimensions
3. Use EARS syntax patterns (Rolls-Royce) to test requirement completeness:
   - Ubiquitous: "The <system> shall <action>" — always active
   - State-driven: "While <state>, the <system> shall <action>"
   - Event-driven: "When <trigger>, the <system> shall <action>"
   - Optional: "Where <feature>, the <system> shall <action>"
   - Unwanted: "If <condition>, then the <system> shall <action>"
   If a requirement doesn't fit ANY pattern, it's likely ambiguous.
4. Generate 3-5 specific questions targeting the weakest areas
5. Present questions to the user and collect answers
6. Call sdd_clarify WITH answers and your dimension_scores assessment
7. If score < threshold, repeat from step 1

## Stage 5: Design (Research: ADR format — Michael Nygard, SOLID — Robert C. Martin, Refactoring — Martin Fowler)

1. Read ALL previous artifacts (use sdd_get_context for proposal, requirements,
   business-rules, clarifications)
2. Design the technical architecture addressing ALL requirements AND business rules
3. Choose tech stack with rationale, define components, data model, API contracts
4. Document key architectural decisions as ADRs with: Context, Decision, Rationale,
   Alternatives Rejected (Michael Nygard format)
5. Perform a Structural Quality Analysis of the proposed design:

   **SOLID Compliance** (Robert C. Martin — Clean Architecture):
   For each component, evaluate:
   - SRP: Does this component have exactly ONE reason to change?
     Ask: "If requirement X changes, which components are affected?"
     If the answer is more than 2 → Shotgun Surgery risk.
   - OCP: Can this component be extended without modifying its source?
     Look for: hardcoded conditionals, switch statements on types.
   - LSP: Are abstractions truly substitutable?
     Look for: type checks, casting, "special case" handling.
   - ISP: Are interfaces specific to their consumers?
     Look for: interfaces with 5+ methods, consumers using only 1-2 methods.
   - DIP: Do components depend on abstractions or concretions?
     Look for: direct struct instantiation vs interface injection.

   **Code Smell Detection** (Martin Fowler — Refactoring, 2nd ed.):
   Scan the component design for these structural smells:
   - Shotgun Surgery: A single logical change requires modifications in many
     components. Ask: "If I change the data model for X, how many files change?"
   - Feature Envy: A component uses more data/methods from another component
     than from itself. Symptom: excessive cross-component method calls.
   - God Class: A component with too many responsibilities (covers 4+ requirements
     OR has 5+ dependencies). Split into focused subcomponents.
   - Divergent Change: A single component changes for multiple unrelated reasons.
     Symptom: "we change this file for both auth AND billing changes."
   - Inappropriate Intimacy: Two components know too much about each other's
     internals. Symptom: accessing private fields, circular dependencies.

   **Coupling & Cohesion**:
   - Afferent coupling (Ca): How many components DEPEND ON this one?
     High Ca = high impact on changes (be careful modifying it).
   - Efferent coupling (Ce): How many components does this one DEPEND ON?
     High Ce = fragile, breaks when dependencies change.
   - Instability (I = Ce / (Ca + Ce)): 0 = maximally stable, 1 = maximally unstable.
     Stable components should be abstract. Unstable ones can be concrete.
   - Cohesion: Do all elements within a component serve its single responsibility?

   **Mitigations**: For each detected smell or SOLID violation, document:
   - What pattern or architectural choice prevents it
   - If the smell is accepted as a trade-off, explain WHY

6. Call sdd_create_design with the complete architecture document, including
   the quality_analysis parameter

## Stage 6: Tasks

1. Read the design document (use sdd_get_context stage=design)
2. Break the design into atomic, AI-ready implementation tasks
3. Each task must have: unique ID (TASK-001), clear scope, requirements covered,
   component affected, dependencies, and acceptance criteria
4. Define the dependency graph (what can be parallelized)
5. Assign execution waves: group tasks into parallel waves based on dependencies.
   Algorithm: tasks with no dependencies = Wave 1, tasks depending only on
   Wave 1 = Wave 2, etc. Tasks within the same wave can execute in parallel.
6. Call sdd_create_tasks with the complete task breakdown, including wave_assignments

## Stage 7: Validate

1. Read ALL artifacts (proposal, requirements, business-rules, clarifications,
   design, tasks)
2. Cross-reference every requirement against tasks (coverage analysis)
3. Cross-reference every component against tasks (component coverage)
4. Check for inconsistencies between artifacts
5. Verify business rules are reflected in design and tasks
6. Assess risks and provide recommendations
7. Verify structural design quality against the task breakdown:
   - For each requirement (FR-XXX), count how many components and tasks it touches.
     If a single requirement change would require modifying 3+ tasks across
     different components → flag as Shotgun Surgery risk.
   - For each task, verify it maintains the SRP established in the design.
     If a task modifies 3+ components → flag as potential coupling issue.
   - Check if the tasks introduce dependencies not documented in the design's
     coupling analysis. New dependencies = new risk.
   - Verify that mitigations documented in the design's Quality Analysis section
     are preserved in the task breakdown (smells are not re-introduced by tasks).
8. Call sdd_validate with the full analysis, design_quality assessment, and
   verdict (PASS/PASS_WITH_WARNINGS/FAIL)
`
