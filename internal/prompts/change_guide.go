package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// ChangeGuidePrompt serves on-demand adaptive change pipeline documentation.
// This is the "cold" counterpart to the change pipeline essentials in serverInstructions().
// Content moved here: context-check heuristics, good vs bad questions,
// structural quality in changes, wave execution multi-agent orchestration.
type ChangeGuidePrompt struct{}

// NewChangeGuidePrompt creates a ChangeGuidePrompt.
func NewChangeGuidePrompt() *ChangeGuidePrompt {
	return &ChangeGuidePrompt{}
}

// Definition returns the MCP prompt definition for registration.
func (p *ChangeGuidePrompt) Definition() mcp.Prompt {
	return mcp.NewPrompt("sdd-change-guide",
		mcp.WithPromptDescription(
			"Detailed adaptive change pipeline documentation. "+
				"Covers context-check heuristics (Requirements Smells, conflict detection, impact classification), "+
				"good vs bad questions, structural quality analysis, and wave execution orchestration.",
		),
	)
}

// Handle returns the change pipeline detail documentation.
func (p *ChangeGuidePrompt) Handle(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "SDD Change Pipeline Guide",
		Messages: []mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.NewTextContent(changeGuideContent),
			},
		},
	}, nil
}

const changeGuideContent = `# SDD Change Pipeline Guide

## Context-Check Stage (Research: IEEE 29148, Femmer et al. 2017, Bohner & Arnold)

The context-check stage is a MANDATORY gate in every change flow. It prevents
conflicts with existing specs, detects ambiguity early, and classifies impact.
Even a small change can break a business rule — context-check catches that.

When context-check is the current stage:

1. Call sdd_context_check with the change description and optional project_name
   - The tool SCANS filesystem and memory, returning a structured report
   - It does NOT analyze — YOU analyze the report using the heuristics below

2. Read the returned report — it contains:
   - Existing SDD artifacts (charters, requirements, business rules, design)
   - Keyword-matched completed changes (max 10, ranked by relevance)
   - Explore observations from memory (if available)
   - Convention files (if no SDD artifacts exist — CLAUDE.md, AGENTS.md, etc.)

3. Analyze for ambiguity using Requirements Smells (Femmer et al. 2017, IEEE 29148):
   - Subjective language: "user-friendly", "fast", "easy", "intuitive", "simple"
   - Ambiguous adverbs: "often", "sometimes", "usually", "typically", "mostly"
   - Non-verifiable terms: "high quality", "good performance", "secure enough"
   - Superlatives: "best", "fastest", "most efficient"
   - Negative statements hiding requirements: "the system shall not..."
   - Comparatives without baseline: "faster than", "better than", "more reliable"
   - Totality terms: "all", "every", "always", "never" (are these truly universal?)

4. Check for conflicts with existing specs and business rules:
   - Does this change contradict any existing constraint or business rule?
   - Does it modify behavior covered by existing requirements (FR-XXX)?
   - Does it introduce terms not in the Ubiquitous Language glossary?
   - Does it reference components or requirement IDs that don't exist?

5. Classify impact (SemVer model — Bohner & Arnold, Software Change Impact Analysis):
   - **Breaking**: changes existing behavior (existing tests would fail)
   - **Non-breaking**: adds new behavior without affecting existing
   - **Patch**: internal change, no behavior modification

6. Generate the context-check.md content and call sdd_change_advance

**If critical issues are found**:
- Present them to the user with specific questions
- Wait for answers before generating context-check.md
- Include both questions and answers in the artifact

**If no issues found**:
- Generate a brief "all clear" documenting what was checked
- Proceed to the next stage

## Good vs Bad Questions (Research: IREB Elicitation Techniques)

Bad (vague, answerable with yes/no):
- "Is this change safe?" — too vague, no actionable answer
- "Will this break anything?" — invites unverified "no"
- "Are there any edge cases?" — too broad, produces hand-waving

Good (specific, evidence-based, probing):
- "FR-012 requires email notifications on status change. Your change modifies
  the status enum — which notification templates need updating?"
- "The business rule says 'orders over $500 require manager approval'. Your
  change removes the approval step — is this rule being deprecated?"
- "The existing design uses JWT with 15-min expiry. Your change adds a
  'remember me' feature — what should the extended token lifetime be?"

## Structural Quality in Changes (ALL sizes — Robert C. Martin, Martin Fowler)

Quality analysis is required for ALL change sizes, not just large ones.
Code smells like Shotgun Surgery often emerge from small, seemingly harmless changes.

**When the current stage is "design"** (medium/large changes):
Include a Structural Quality Analysis section in the design content.
Use the same SOLID + Code Smell + Coupling & Cohesion framework from Stage 5
of the project pipeline. Even in a change context, analyze:
- Which existing components does this change touch? If 3+ → Shotgun Surgery risk.
- Does this change add a new dependency between components? Document it.
- Does this change violate SRP of any existing component? Propose a split.
- Does this change introduce Feature Envy (new code uses more data from
  another component than its own)?

**When the current stage is "verify"** (ALL sizes):
Include a Design Quality section in the verification content.
For small changes (no design stage), perform the quality analysis HERE:
- Map the change description to affected components
- Count how many components/files the change touches
- For each task, ask: "If this requirement changes again, how many places break?"
- Flag any detected smells with specific mitigation recommendations
- If a smell is accepted as a trade-off, document WHY explicitly

For medium/large changes (design stage exists), cross-check:
- Verify the mitigations from the design's Quality Analysis are preserved in tasks
- Check if the task breakdown introduces new coupling not documented in the design
- Verify no God Class patterns emerge from combining multiple tasks into one component

## Wave Assignments in Tasks Stage

When writing content for the **tasks** stage (both project pipeline and change pipeline),
include execution wave assignments to enable parallel execution:
- Group tasks into waves based on dependencies
- Wave 1: tasks with no dependencies (can all run in parallel)
- Wave 2: tasks that depend only on Wave 1 tasks (can run in parallel with each other)
- Continue for Wave 3, 4, etc.
- Format as a clear section in the tasks content (e.g., "## Execution Waves")
- For the project pipeline, use the wave_assignments parameter in sdd_create_tasks

## Wave Execution — Multi-Agent Orchestration

When the user asks you to IMPLEMENT tasks that have wave assignments, use this strategy
to execute them efficiently. Adapt based on your available capabilities:

**Step 1 — Detect your orchestration tier:**
- **Tier 1 (Agent Teams)**: You can create a team of independent agents with a shared task list,
  inter-agent communication, and dependency-aware task claiming. Agents coordinate among themselves.
- **Tier 2 (Sub-Agents)**: You can launch parallel worker agents that execute independently and
  report results back to you (the caller). Workers cannot communicate with each other.
- **Tier 3 (Sequential)**: You have no parallelization capability. You execute tasks one by one.

If you are unsure which tier you support, ASK THE USER before proceeding.

**Step 2 — Execute by tier:**

Tier 1 (Agent Teams):
1. Create a team with enough agents to cover the largest wave
2. Map each TASK from the wave assignments to a task in the shared task list
3. Set task dependencies matching the dependency graph — blocked tasks auto-unblock when dependencies complete
4. Let agents self-claim and execute tasks. Wave boundaries are enforced by the dependency graph itself.
5. Each agent should use the namespace parameter on memory tools (e.g., namespace="agent/<task-id>")
   to isolate its observations. The orchestrator reads without namespace to see all progress.
6. Use mem_progress to track overall wave completion state

Tier 2 (Sub-Agents):
1. Start with Wave 1 — launch one sub-agent per task in the wave, all in parallel
2. Wait for ALL sub-agents in the wave to complete before starting the next wave
3. NEVER start Wave N+1 until every task in Wave N has succeeded
4. Each sub-agent should use namespace="subagent/<task-id>" on memory tools for isolation
5. After each wave, check results and update mem_progress before launching the next wave
6. If a sub-agent fails, stop and report — do not continue to the next wave

Tier 3 (Sequential):
1. Execute tasks in dependency graph order (not wave order — follow the actual dependencies)
2. Complete each task fully before starting the next
3. Use mem_progress to checkpoint after each task completion
4. If a task fails, stop and report

**Step 3 — Prevent file conflicts:**
Tasks within the same wave MUST NOT modify the same files. If the task breakdown has
overlapping file ownership in the same wave, flag this to the user before executing.
This applies to Tier 1 and Tier 2 only (Tier 3 is sequential, so no conflicts).

**Step 4 — Report completion:**
After all waves complete, provide a summary: which tasks succeeded, which failed,
total time if available, and any issues encountered during execution.
`
