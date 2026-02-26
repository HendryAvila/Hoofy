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

	businessRulesTool := tools.NewBusinessRulesTool(store, renderer)
	s.AddTool(businessRulesTool.Definition(), businessRulesTool.Handle)

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

	// Context-check tool registered unconditionally — handles nil memStore
	// internally by skipping memory search (ADR-001: scanner, not analyzer).
	contextCheckTool := tools.NewContextCheckTool(changeStore, memStore)
	s.AddTool(contextCheckTool.Definition(), contextCheckTool.Handle)
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
		businessRulesTool.SetBridge(bridge)
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

// registerMemoryTools registers all 19 memory MCP tools with the server.
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

	// --- Progress tracking ---
	progressTool := memtools.NewProgressTool(ms)
	s.AddTool(progressTool.Definition(), progressTool.Handle)

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

	// --- Compaction ---
	compactTool := memtools.NewCompactTool(ms)
	s.AddTool(compactTool.Definition(), compactTool.Handle)

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
(Source: IEEE 29148 — "well-formed requirements" prevent defects downstream)

## CRITICAL: How Tools Work
Hoofy tools are STORAGE tools, not AI tools. They save content YOU generate.
The workflow for each stage is:

1. TALK to the user — understand their idea, ask questions
2. GENERATE the content yourself (proposals, requirements, etc.)
3. CALL the tool with the ACTUAL content as parameters
4. The tool saves it to disk and advances the pipeline

NEVER call a tool with placeholder text like "TBD" or "to be defined".
ALWAYS generate real, substantive content based on your conversation with the user.

## Pipeline
SDD follows a sequential 8-stage pipeline:
1. INIT — Set up the project (call sdd_init_project)
2. PROPOSE — Create a structured proposal (YOU write it, tool saves it)
3. SPECIFY — Extract formal requirements with IEEE 29148 quality attributes
4. BUSINESS RULES — Extract declarative business rules using BRG taxonomy
5. CLARIFY — The Clarity Gate: resolve ambiguities using EARS patterns
6. DESIGN — Technical architecture document with ADRs (Michael Nygard format)
7. TASKS — Atomic task breakdown with execution wave assignments
8. VALIDATE — Cross-artifact consistency check (YOU analyze, tool saves report)

## Stage-by-Stage Workflow

### Stage 1: Propose (Research: IREB Elicitation Techniques)
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

### Stage 2: Specify (Research: IEEE 29148 Quality Attributes)
1. Read the proposal from sdd/proposal.md (use sdd_get_context if needed)
2. Extract formal requirements using MoSCoW prioritization
3. Each requirement gets a unique ID (FR-001 for functional, NFR-001 for non-functional)
4. Apply IEEE 29148 quality attributes — each requirement MUST be:
   - Necessary: traceable to a user need from the proposal
   - Unambiguous: one interpretation only (no "etc.", "and/or", "appropriate")
   - Verifiable: testable with a concrete condition
   - Consistent: no contradictions with other requirements
5. Call sdd_generate_requirements with real requirements content

### Stage 3: Business Rules (Research: BRG Taxonomy, Business Rules Manifesto, DDD)
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

### Stage 4: Clarify — Clarity Gate (Research: EARS, Femmer et al. 2017)
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

### Stage 5: Design (Research: ADR format — Michael Nygard)
1. Read ALL previous artifacts (use sdd_get_context for proposal, requirements,
   business-rules, clarifications)
2. Design the technical architecture addressing ALL requirements AND business rules
3. Choose tech stack with rationale, define components, data model, API contracts
4. Document key architectural decisions as ADRs with: Context, Decision, Rationale,
   Alternatives Rejected (Michael Nygard format)
5. Call sdd_create_design with the complete architecture document

### Stage 6: Tasks
1. Read the design document (use sdd_get_context stage=design)
2. Break the design into atomic, AI-ready implementation tasks
3. Each task must have: unique ID (TASK-001), clear scope, requirements covered,
   component affected, dependencies, and acceptance criteria
4. Define the dependency graph (what can be parallelized)
5. Assign execution waves: group tasks into parallel waves based on dependencies.
   Algorithm: tasks with no dependencies = Wave 1, tasks depending only on
   Wave 1 = Wave 2, etc. Tasks within the same wave can execute in parallel.
6. Call sdd_create_tasks with the complete task breakdown, including wave_assignments

### Stage 7: Validate
1. Read ALL artifacts (proposal, requirements, business-rules, clarifications,
   design, tasks)
2. Cross-reference every requirement against tasks (coverage analysis)
3. Cross-reference every component against tasks (component coverage)
4. Check for inconsistencies between artifacts
5. Verify business rules are reflected in design and tasks
6. Assess risks and provide recommendations
7. Call sdd_validate with the full analysis and verdict (PASS/PASS_WITH_WARNINGS/FAIL)

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
- After validation, the user's SDD specs are ready for implementation

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

### Progress Tracking (mem_progress)
Use mem_progress to persist a structured work-in-progress document that survives context compaction.
Unlike session summaries (end-of-session), progress tracks WHERE YOU ARE mid-session.

**Dual behavior**:
- Read: mem_progress(project="X") — returns current progress (call at session start!)
- Write: mem_progress(project="X", content=JSON) — upserts the progress doc

**When to use**:
- At session start: read progress to check for prior WIP
- After completing significant work: update with current state
- Before context compaction: save progress so the next window can continue

**Content must be valid JSON.** Recommended structure:
{"goal": "...", "completed": ["..."], "next_steps": ["..."], "blockers": ["..."]}

One active progress per project — each write replaces the previous one.

### Memory Compaction (mem_compact)
Use mem_compact to identify and clean up stale observations that add noise to memory.
Over time, memory accumulates old session notes, outdated discoveries, and superseded
decisions. Compaction keeps memory lean and relevant.

**Dual behavior**:
- Identify: mem_compact(older_than_days=90) — lists stale candidates without deleting
- Execute: mem_compact(older_than_days=90, compact_ids="[1,2,3]") — batch soft-deletes

**Workflow** (two-step process):
1. Call mem_compact WITHOUT compact_ids to review candidates
2. Review the list — decide which observations are truly stale
3. Optionally write a summary to preserve key knowledge
4. Call mem_compact WITH compact_ids (and optional summary_title/summary_content)

**When to suggest compaction**:
- When mem_context returns many old, low-value observations
- When a user complains about memory noise or irrelevant results
- After a major milestone (v1 shipped, refactor complete) — clean up WIP notes
- When observation count exceeds 200+ for a project

**Summary observations**:
When compacting, create a summary to preserve the essence of what was deleted:
- summary_title: "Compacted 15 pre-v1 session notes"
- summary_content: Key decisions and patterns extracted from the deleted observations
- The summary is saved as type "compaction_summary" — searchable via mem_search

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

### Response Verbosity Control (detail_level parameter)
Several read-heavy tools support a detail_level parameter that controls response size.
Use this to manage context window budget — fetch the minimum detail needed first,
then drill deeper only when necessary (Anthropic: "context is a finite resource").

**Available levels**:
- summary: Minimal tokens — IDs, titles, metadata only. Use for orientation and triage.
- standard: Truncated content snippets. Good balance for most operations.
- full: Complete untruncated content. Use only when you need to analyze details.

**Default detail_level by tool**:
- sdd_get_context: defaults to summary (minimal pipeline overview)
- mem_context, mem_search, mem_timeline, sdd_context_check: default to standard

**Tools that support detail_level**:
- mem_context: Controls observation content in recent memory context
- mem_search: Controls search result content (summary = titles only, full = complete content)
- mem_timeline: Controls timeline entries (summary = titles only, full = all content untruncated)
- sdd_context_check: Controls artifact excerpts and memory results in change reports
- sdd_get_context: Controls pipeline artifact content (summary = stage status only, full = complete artifacts)

**Navigation hints**:
When results are capped by limit, tools append a "📊 Showing X of Y" footer.
This tells you whether you're seeing everything or need to adjust limits.
Tools with navigation hints: mem_search, mem_context, mem_timeline.

**Progressive disclosure with detail_level**:
1. Start with summary to scan what exists (minimal tokens)
2. If something looks relevant, use standard for that specific tool call
3. Only use full when you need the complete content for analysis

Summary-mode responses include a footer hint reminding about the option to use
standard or full for more detail.

### Sub-Agent Memory Scoping (namespace parameter)

When multiple AI sub-agents work in parallel (e.g., orchestrator spawns researcher, coder, reviewer),
use the namespace parameter to isolate each sub-agent's memory observations.

**What namespace does**:
- Tags observations with a namespace string (e.g., "subagent/task-123", "agent/researcher")
- Read tools filter by namespace when provided — each sub-agent sees only its own notes
- Omitting namespace = no filter — the orchestrator sees EVERYTHING (by design)

**Namespace vs scope**: These are orthogonal concepts:
- scope = WHO sees it (project vs personal) — visibility level
- namespace = WHICH AGENT owns it — isolation boundary

**Tools that support namespace**:
- Write: mem_save, mem_save_prompt, mem_session_summary, mem_progress
- Read: mem_search, mem_context, mem_compact

**Convention for namespace values**:
- Sub-agents by task: "subagent/task-123", "subagent/research-auth"
- Sub-agents by role: "agent/researcher", "agent/coder", "agent/reviewer"
- Orchestrator: omit namespace entirely (sees all namespaces)

**Typical multi-agent workflow**:
1. Orchestrator spawns sub-agent with a task ID
2. Sub-agent uses namespace="subagent/<task-id>" on all mem_save/mem_search calls
3. Sub-agent's observations are isolated — no cross-contamination with other sub-agents
4. Orchestrator reads without namespace to see all observations, then synthesizes
5. Orchestrator saves final synthesis without namespace (shared knowledge)

**mem_progress with namespace**: When namespace is provided, the topic_key becomes
"progress/<namespace>/<project>" instead of "progress/<project>", giving each
sub-agent its own progress document.

**mem_timeline does NOT support namespace**: Timeline is inherently ID-scoped
(centered on a specific observation_id), so namespace filtering is unnecessary.

### Context Budget Awareness (max_tokens parameter)

Five read-heavy tools accept an optional max_tokens integer parameter that caps response
size by estimated token count. Use this when context window space is limited or when
you need to fit a response within a specific token budget.

**How it works**:
- Token estimation uses len(text)/4 heuristic (fast O(1), no tokenizer dependency)
- When max_tokens is set, the response is capped at approximately that many tokens
- Every response from these tools includes a "📏 ~N tokens" footer showing estimated size
- When a response is budget-capped, a "⚡ Budget-capped" notice is prepended to the footer

**Tools that support max_tokens**:
- mem_context: Incremental build — stops adding observations when budget would be exceeded
- mem_search: Incremental build — stops adding results when budget would be exceeded
- mem_timeline: Post-hoc truncation — builds full response, then truncates to budget
- sdd_get_context: Post-hoc truncation — builds full response, then truncates to budget
- sdd_context_check: Post-hoc truncation — builds full response, then truncates to budget

**When to use max_tokens**:
- When you know your remaining context window budget and want to stay within it
- When fetching context for a sub-agent with a smaller context window
- When you need a quick overview and want to cap verbosity beyond what detail_level provides

**max_tokens vs detail_level**: These are complementary:
- detail_level controls WHAT is included (summary vs standard vs full content)
- max_tokens controls HOW MUCH total output, regardless of detail level
- Use detail_level first to control content type, then max_tokens as a hard cap if needed

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
adaptive change pipeline instead of the full 8-stage SDD pipeline.

### When to Use Changes vs Full Pipeline
- **Full pipeline** (sdd_init_project): Brand new projects from scratch
- **Change pipeline** (sdd_change): Any modification to an existing codebase

### How It Works
Each change has a TYPE and SIZE that determine the pipeline stages.
ALL flows include a mandatory context-check stage (see below).

**Types**: feature, fix, refactor, enhancement
**Sizes**: small (4 stages), medium (5 stages), large (6-7 stages)

### Stage Flows by Type and Size

**Fix**:
- small: describe → context-check → tasks → verify
- medium: describe → context-check → spec → tasks → verify
- large: describe → context-check → spec → design → tasks → verify

**Feature**:
- small: describe → context-check → tasks → verify
- medium: propose → context-check → spec → tasks → verify
- large: propose → context-check → spec → clarify → design → tasks → verify

**Refactor**:
- small: scope → context-check → tasks → verify
- medium: scope → context-check → design → tasks → verify
- large: scope → context-check → spec → design → tasks → verify

**Enhancement**:
- small: describe → context-check → tasks → verify
- medium: propose → context-check → spec → tasks → verify
- large: propose → context-check → spec → clarify → design → tasks → verify

### Context-Check Stage (Research: IEEE 29148, Femmer et al. 2017, Bohner & Arnold)

The context-check stage is a MANDATORY gate in every change flow. It prevents
conflicts with existing specs, detects ambiguity early, and classifies impact.
Even a small change can break a business rule — context-check catches that.

When context-check is the current stage:

1. Call sdd_context_check with the change description and optional project_name
   - The tool SCANS filesystem and memory, returning a structured report
   - It does NOT analyze — YOU analyze the report using the heuristics below

2. Read the returned report — it contains:
   - Existing SDD artifacts (proposals, requirements, business rules, design)
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

### Good vs Bad Questions (Research: IREB Elicitation Techniques)

Bad (vague, answerable with yes/no):
- "Is this change safe?" → too vague, no actionable answer
- "Will this break anything?" → invites unverified "no"
- "Are there any edge cases?" → too broad, produces hand-waving

Good (specific, evidence-based, probing):
- "FR-012 requires email notifications on status change. Your change modifies
  the status enum — which notification templates need updating?"
- "The business rule says 'orders over $500 require manager approval'. Your
  change removes the approval step — is this rule being deprecated?"
- "The existing design uses JWT with 15-min expiry. Your change adds a
  'remember me' feature — what should the extended token lifetime be?"

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
   Architecture Decision Record (Michael Nygard format)
   - With active change: saves ADR file + updates change record
   - Without active change: saves to memory only (standalone ADR)

### Important Rules
- Only ONE active change at a time
- Complete or archive a change before starting a new one
- Generate REAL content for each stage — no placeholders
- All flows end with verify — use it to validate the change
- ADRs can be captured at any time during a change
- Context-check is MANDATORY — never skip it, even for small changes

### Wave Assignments in Tasks Stage
When writing content for the **tasks** stage (both project pipeline and change pipeline),
include execution wave assignments to enable parallel execution:
- Group tasks into waves based on dependencies
- Wave 1: tasks with no dependencies (can all run in parallel)
- Wave 2: tasks that depend only on Wave 1 tasks (can run in parallel with each other)
- Continue for Wave 3, 4, etc.
- Format as a clear section in the tasks content (e.g., "## Execution Waves")
- For the project pipeline, use the wave_assignments parameter in sdd_create_tasks

### Wave Execution — Multi-Agent Orchestration
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
total time if available, and any issues encountered during execution.`
}
