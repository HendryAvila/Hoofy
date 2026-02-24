// Package server wires all MCP components and creates the server instance.
//
// This is the composition root (DIP): it creates concrete implementations
// and injects them into the tools/prompts/resources that depend on abstractions.
// No business logic lives here — only wiring.
package server

import (
	"fmt"
	"log"

	"github.com/HendryAvila/Hoofy/internal/changes"
	"github.com/HendryAvila/Hoofy/internal/config"
	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/HendryAvila/Hoofy/internal/memtools"
	"github.com/HendryAvila/Hoofy/internal/prompts"
	"github.com/HendryAvila/Hoofy/internal/resources"
	"github.com/HendryAvila/Hoofy/internal/templates"
	"github.com/HendryAvila/Hoofy/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set at build time via ldflags.
var Version = "dev"

// New creates and configures the MCP server with all tools, prompts,
// and resources registered. This is the single place where all
// dependencies are resolved.
//
// The returned cleanup function closes the memory store's database
// connection and must be called on shutdown (typically via defer).
// It is always non-nil and safe to call even if memory init failed.
func New() (*server.MCPServer, func(), error) {
	// --- Create shared dependencies ---

	store := config.NewFileStore()

	renderer, err := templates.NewRenderer()
	if err != nil {
		return nil, noop, fmt.Errorf("creating template renderer: %w", err)
	}

	// --- Create the MCP server ---

	s := server.NewMCPServer(
		"hoofy",
		Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
		server.WithInstructions(serverInstructions()),
	)

	// --- Register SDD tools ---

	initTool := tools.NewInitTool(store)
	s.AddTool(initTool.Definition(), initTool.Handle)

	proposeTool := tools.NewProposeTool(store, renderer)
	s.AddTool(proposeTool.Definition(), proposeTool.Handle)

	specifyTool := tools.NewSpecifyTool(store, renderer)
	s.AddTool(specifyTool.Definition(), specifyTool.Handle)

	clarifyTool := tools.NewClarifyTool(store, renderer)
	s.AddTool(clarifyTool.Definition(), clarifyTool.Handle)

	designTool := tools.NewDesignTool(store, renderer)
	s.AddTool(designTool.Definition(), designTool.Handle)

	tasksTool := tools.NewTasksTool(store, renderer)
	s.AddTool(tasksTool.Definition(), tasksTool.Handle)

	validateTool := tools.NewValidateTool(store)
	s.AddTool(validateTool.Definition(), validateTool.Handle)

	contextTool := tools.NewContextTool(store)
	s.AddTool(contextTool.Definition(), contextTool.Handle)

	// --- Register change pipeline tools ---
	//
	// The change pipeline is independent from the project pipeline —
	// it works without sdd.json. It uses its own FileStore for
	// persistence under sdd/changes/.

	changeStore := changes.NewFileStore()

	changeTool := tools.NewChangeTool(changeStore)
	s.AddTool(changeTool.Definition(), changeTool.Handle)

	changeAdvanceTool := tools.NewChangeAdvanceTool(changeStore)
	s.AddTool(changeAdvanceTool.Definition(), changeAdvanceTool.Handle)

	changeStatusTool := tools.NewChangeStatusTool(changeStore)
	s.AddTool(changeStatusTool.Definition(), changeStatusTool.Handle)

	adrTool := tools.NewADRTool(changeStore)
	s.AddTool(adrTool.Definition(), adrTool.Handle)

	// --- Register memory tools ---
	//
	// Memory is an independent subsystem: if it fails to initialize,
	// SDD tools continue working. We log a warning and skip memory
	// tool registration — the server is still fully functional for
	// spec-driven development.

	cleanup := noop
	memStore, memErr := memory.New(memory.DefaultConfig())
	if memErr != nil {
		log.Printf("WARNING: memory subsystem disabled: %v", memErr)
	} else {
		cleanup = func() {
			if err := memStore.Close(); err != nil {
				log.Printf("WARNING: memory store close: %v", err)
			}
		}
		registerMemoryTools(s, memStore)

		// --- Wire SDD-Memory bridge ---
		//
		// When memory is available, SDD stage completions are automatically
		// saved as memory observations with topic_key upserts. This enables
		// cross-session awareness of pipeline state. The bridge is nil-safe:
		// if memory init failed, tools work normally without it.
		bridge := tools.NewMemoryBridge(memStore)
		proposeTool.SetBridge(bridge)
		specifyTool.SetBridge(bridge)
		clarifyTool.SetBridge(bridge)
		designTool.SetBridge(bridge)
		tasksTool.SetBridge(bridge)
		validateTool.SetBridge(bridge)

		// Wire change pipeline bridge — saves stage completions and ADRs
		// to memory for cross-session awareness.
		changeAdvanceTool.SetBridge(bridge)
		adrTool.SetBridge(bridge)

		// --- Register explore tool (SDD + Memory hybrid) ---
		//
		// sdd_explore is a standalone tool that captures pre-pipeline context.
		// It depends only on memory.Store, not on config or change stores.
		// Registered here because it requires memory to be available.
		exploreTool := tools.NewExploreTool(memStore)
		s.AddTool(exploreTool.Definition(), exploreTool.Handle)
	}

	// --- Register prompts ---

	startPrompt := prompts.NewStartPrompt()
	s.AddPrompt(startPrompt.Definition(), startPrompt.Handle)

	statusPrompt := prompts.NewStatusPrompt()
	s.AddPrompt(statusPrompt.Definition(), statusPrompt.Handle)

	// --- Register resources ---

	resourceHandler := resources.NewHandler(store)
	s.AddResource(resourceHandler.StatusResource(), resourceHandler.HandleStatus)

	return s, cleanup, nil
}

// noop is a no-op cleanup function used as the default when memory
// is disabled or hasn't been initialized.
func noop() {}

// registerMemoryTools registers all 17 memory MCP tools with the server.
func registerMemoryTools(s *server.MCPServer, ms *memory.Store) {
	// --- Session lifecycle ---
	sessionStart := memtools.NewSessionStartTool(ms)
	s.AddTool(sessionStart.Definition(), sessionStart.Handle)

	sessionEnd := memtools.NewSessionEndTool(ms)
	s.AddTool(sessionEnd.Definition(), sessionEnd.Handle)

	sessionSummary := memtools.NewSessionSummaryTool(ms)
	s.AddTool(sessionSummary.Definition(), sessionSummary.Handle)

	// --- Save & capture ---
	saveTool := memtools.NewSaveTool(ms)
	s.AddTool(saveTool.Definition(), saveTool.Handle)

	savePrompt := memtools.NewSavePromptTool(ms)
	s.AddTool(savePrompt.Definition(), savePrompt.Handle)

	passiveCapture := memtools.NewPassiveCaptureTool(ms)
	s.AddTool(passiveCapture.Definition(), passiveCapture.Handle)

	// --- Query & retrieval ---
	searchTool := memtools.NewSearchTool(ms)
	s.AddTool(searchTool.Definition(), searchTool.Handle)

	memContext := memtools.NewContextTool(ms)
	s.AddTool(memContext.Definition(), memContext.Handle)

	timelineTool := memtools.NewTimelineTool(ms)
	s.AddTool(timelineTool.Definition(), timelineTool.Handle)

	getObs := memtools.NewGetObservationTool(ms)
	s.AddTool(getObs.Definition(), getObs.Handle)

	// --- Management ---
	deleteTool := memtools.NewDeleteTool(ms)
	s.AddTool(deleteTool.Definition(), deleteTool.Handle)

	updateTool := memtools.NewUpdateTool(ms)
	s.AddTool(updateTool.Definition(), updateTool.Handle)

	suggestKey := memtools.NewSuggestTopicKeyTool()
	s.AddTool(suggestKey.Definition(), suggestKey.Handle)

	// --- Statistics ---
	statsTool := memtools.NewStatsTool(ms)
	s.AddTool(statsTool.Definition(), statsTool.Handle)

	// --- Knowledge graph (relations) ---
	relateTool := memtools.NewRelateTool(ms)
	s.AddTool(relateTool.Definition(), relateTool.Handle)

	unrelateTool := memtools.NewUnrelateTool(ms)
	s.AddTool(unrelateTool.Definition(), unrelateTool.Handle)

	buildCtx := memtools.NewBuildContextTool(ms)
	s.AddTool(buildCtx.Definition(), buildCtx.Handle)
}

// serverInstructions returns the system instructions that tell the AI
// how to use Hoofy effectively.
func serverInstructions() string {
	return `You have access to Hoofy, a Spec-Driven Development MCP server.

## WHEN TO ACTIVATE Hoofy

You MUST proactively suggest using Hoofy when the user:
- Asks to build a new project, app, or system
- Asks to add a new feature or major enhancement
- Describes a vague idea and wants to start coding
- Says things like "I want to build...", "let's create...", "add a feature for..."
- Asks you to plan, architect, or design something

When you detect any of these, say something like:
"Before we start coding, let's use Hoofy to define clear specs.
This prevents hallucinations and ensures we build exactly what you need.
Should I start the SDD pipeline?"

You do NOT need to activate Hoofy for:
- Bug fixes or small patches
- Refactoring existing code without changing behavior
- Questions, explanations, or documentation
- One-liner changes or config tweaks

For bug fixes, refactors, enhancements, and small features, use the
ADAPTIVE CHANGE PIPELINE instead (see below).

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

## What is SDD?
Spec-Driven Development reduces AI hallucinations by forcing clear specifications 
BEFORE writing code. Ambiguous requirements are the #1 cause of bad AI-generated code.

## CRITICAL: How Tools Work
Hoofy tools are STORAGE tools, not AI tools. They save content YOU generate.
The workflow for each stage is:

1. TALK to the user → understand their idea, ask questions
2. GENERATE the content yourself (proposals, requirements, etc.)
3. CALL the tool with the ACTUAL content as parameters
4. The tool saves it to disk and advances the pipeline

NEVER call a tool with placeholder text like "TBD" or "to be defined".
ALWAYS generate real, substantive content based on your conversation with the user.

## Pipeline
SDD follows a sequential 7-stage pipeline:
1. INIT → Set up the project (call sdd_init_project)
2. PROPOSE → Create a structured proposal (YOU write it, tool saves it)
3. SPECIFY → Extract formal requirements (YOU write them, tool saves them)
4. CLARIFY → The Clarity Gate: resolve ambiguities (interactive Q&A)
5. DESIGN → Technical architecture document (YOU design it, tool saves it)
6. TASKS → Atomic implementation task breakdown (YOU break it down, tool saves it)
7. VALIDATE → Cross-artifact consistency check (YOU analyze, tool saves report)

## Stage-by-Stage Workflow

### Stage 1: Propose
1. Ask the user about their project idea
2. Ask follow-up questions to understand the problem, users, and goals
3. Based on the conversation, generate content for ALL sections:
   - problem_statement: The core problem (2-3 sentences)
   - target_users: 2-3 specific user personas with needs
   - proposed_solution: High-level description (NO tech details)
   - out_of_scope: 3-5 explicit exclusions
   - success_criteria: 2-4 measurable outcomes
   - open_questions: Remaining unknowns
4. Call sdd_create_proposal with all sections filled in

### Stage 2: Specify
1. Read the proposal from sdd/proposal.md (use sdd_get_context if needed)
2. Extract formal requirements using MoSCoW prioritization
3. Each requirement gets a unique ID (FR-001 for functional, NFR-001 for non-functional)
4. Call sdd_generate_requirements with real requirements content

### Stage 3: Clarify (Clarity Gate)
1. Call sdd_clarify WITHOUT answers to get the analysis framework
2. Analyze the requirements across all 8 dimensions
3. Generate 3-5 specific questions targeting the weakest areas
4. Present questions to the user and collect answers
5. Call sdd_clarify WITH answers and your dimension_scores assessment
6. If score < threshold, repeat from step 1

### Stage 4: Design
1. Read ALL previous artifacts (use sdd_get_context for proposal, requirements, clarifications)
2. Design the technical architecture addressing ALL requirements
3. Choose tech stack with rationale, define components, data model, API contracts
4. Document key architectural decisions (ADRs) with alternatives considered
5. Call sdd_create_design with the complete architecture document

### Stage 5: Tasks
1. Read the design document (use sdd_get_context stage=design)
2. Break the design into atomic, AI-ready implementation tasks
3. Each task must have: unique ID (TASK-001), clear scope, requirements covered,
   component affected, dependencies, and acceptance criteria
4. Define the dependency graph (what can be parallelized)
5. Call sdd_create_tasks with the complete task breakdown

### Stage 6: Validate
1. Read ALL artifacts (proposal, requirements, clarifications, design, tasks)
2. Cross-reference every requirement against tasks (coverage analysis)
3. Cross-reference every component against tasks (component coverage)
4. Check for inconsistencies between artifacts
5. Assess risks and provide recommendations
6. Call sdd_validate with the full analysis and verdict (PASS/PASS_WITH_WARNINGS/FAIL)

## Modes
- Guided: More questions, examples, encouragement. For non-technical users.
  Clarity threshold: 70/100.
- Expert: Direct, concise, technical. For experienced developers.
  Clarity threshold: 50/100.

## Important Rules
- NEVER skip the Clarity Gate
- ALWAYS follow the pipeline order
- NEVER pass placeholder text to tools — generate REAL content
- Each requirement must have a unique ID (FR-001, NFR-001)
- Each task must have a unique ID (TASK-001) and trace to requirements
- Be specific — "users" is not a valid target audience
- In Guided mode: use simple language, give examples, be encouraging
- In Expert mode: be direct, technical language is fine
- After validation, the user's SDD specs are ready for implementation with /plan mode

## PERSISTENT MEMORY

Hoofy includes a persistent memory system for cross-session awareness.
Memory survives between conversations — use it to build project knowledge over time.

### When to Save (call mem_save PROACTIVELY after each of these)
- Architectural decisions or tradeoffs made
- Bug fixes: what was wrong, why, how it was fixed
- New patterns or conventions established
- Configuration changes or environment setup
- Important discoveries, gotchas, or edge cases
- File structure changes or significant refactoring

### Content Format (use this structured format for mem_save content)
**What**: [concise description of what was done]
**Why**: [the reasoning, user request, or problem that drove it]
**Where**: [files/paths affected, e.g. src/auth/middleware.ts]
**Learned**: [gotchas, edge cases, or decisions — omit if none]

### Title Guidelines
Short and searchable: "JWT auth middleware", "Fixed N+1 in user list", "Switched from REST to gRPC"

### Type Categories
Use the type parameter: decision, architecture, bugfix, pattern, config, discovery, learning

### When to Search (call mem_search)
- At the start of a new session to recover context
- Before making architectural decisions (check if prior decisions exist)
- When encountering familiar errors or patterns
- When the user references something from a previous session

### Session Lifecycle
1. Call mem_session_start at the beginning of each coding session
2. Save observations throughout the session (decisions, fixes, discoveries)
3. Call mem_session_summary with a structured summary (Goal/Instructions/Discoveries/Accomplished/Files)
4. Call mem_session_end to close the session

### Topic Keys for Evolving Observations
Use topic_key when an observation should UPDATE over time (not create duplicates):
- Architecture decisions: "architecture/auth-model", "architecture/data-layer"
- Project configuration: "config/deployment", "config/ci-cd"
Use mem_suggest_topic_key to generate a normalized key from a title.

### User Prompts
Call mem_save_prompt to record what the user asked — their intent and goals.
This helps future sessions understand context without the user repeating themselves.

### Progressive Disclosure Pattern
1. Start with mem_context for recent observations
2. Use mem_search for specific topics
3. Use mem_timeline to see chronological context around a search result
4. Use mem_get_observation to read the full, untruncated content

### Knowledge Graph (Relations)

Observations can be connected with typed, directional relations to form a knowledge graph.
This transforms flat memories into a navigable web of connected decisions, patterns, and discoveries.

**Creating relations** — use mem_relate after saving related observations:
- mem_relate(from_id, to_id, relation_type) — creates a directional edge
- Common types: relates_to, implements, depends_on, caused_by, supersedes, part_of
- Use bidirectional=true when the relationship goes both ways
- Add a note to explain WHY the observations are related

**Traversing the graph** — use mem_build_context to explore connections:
- mem_build_context(observation_id) — shows connected observations up to depth 2
- mem_build_context(observation_id, depth=3) — goes deeper for more context
- Use this when exploring a topic to understand its full web of related decisions

**Removing relations** — use mem_unrelate(id) with the relation ID

**When to create relations**:
- After a bug fix, relate it to the decision that caused it (caused_by)
- After implementing a feature, relate tasks to their requirements (implements)
- When a new decision supersedes an old one (supersedes)
- When observations are about the same topic (relates_to)
- When one pattern depends on another (depends_on)

## ADAPTIVE CHANGE PIPELINE

For ongoing development (features, fixes, refactors, enhancements), use the
adaptive change pipeline instead of the full 7-stage SDD pipeline.

### When to Use Changes vs Full Pipeline
- **Full pipeline** (sdd_init_project): Brand new projects from scratch
- **Change pipeline** (sdd_change): Any modification to an existing codebase

### How It Works
Each change has a TYPE and SIZE that determine the pipeline stages:

**Types**: feature, fix, refactor, enhancement
**Sizes**: small (3 stages), medium (4 stages), large (5-6 stages)

### Stage Flows by Type and Size

**Fix**: 
- small: describe → tasks → verify
- medium: describe → spec → tasks → verify
- large: describe → spec → design → tasks → verify

**Feature**:
- small: describe → tasks → verify
- medium: propose → spec → tasks → verify
- large: propose → spec → clarify → design → tasks → verify

**Refactor**:
- small: scope → tasks → verify
- medium: scope → design → tasks → verify
- large: scope → spec → design → tasks → verify

**Enhancement**:
- small: describe → tasks → verify
- medium: propose → spec → tasks → verify
- large: propose → spec → clarify → design → tasks → verify

### Change Pipeline Workflow

1. **Create a change**: Call sdd_change with type, size, and description
   - Only ONE active change at a time
   - The tool creates a directory at sdd/changes/<slug>/

2. **Work through stages**: For each stage, generate content and call
   sdd_change_advance with the content
   - The tool writes the content as <stage>.md in the change directory
   - It advances the state machine to the next stage
   - When the final stage (verify) is completed, the change is marked done

3. **Check progress**: Call sdd_change_status to see the current state,
   stage progress, artifact sizes, and ADRs

4. **Capture decisions**: Call sdd_adr at any time to record an
   Architecture Decision Record
   - With active change: saves ADR file + updates change record
   - Without active change: saves to memory only (standalone ADR)

### Important Rules
- Only ONE active change at a time
- Complete or archive a change before starting a new one
- Generate REAL content for each stage — no placeholders
- All flows end with verify — use it to validate the change
- ADRs can be captured at any time during a change`
}
