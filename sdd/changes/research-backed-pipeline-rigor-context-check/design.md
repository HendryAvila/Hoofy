# Design: Research-Backed Pipeline Rigor

## Architecture Overview

This feature introduces two new pipeline stages and a server instructions rewrite across three subsystems:

1. **`context-check` stage** — new stage in the adaptive change pipeline (all 12 flows)
2. **`business-rules` stage** — new stage in the greenfield project pipeline (8 stages total)
3. **Server instructions rewrite** — research-backed question frameworks in `serverInstructions()`

The design follows existing architectural patterns: SRP (one file per tool), DIP (tools depend on interfaces), constructor injection via `server.go`, and markdown artifacts on filesystem.

### Key Design Principle: Targeted Context, Not Bulk Injection

Per Anthropic's guidance and user requirement: "contextos específicos, no inyectar todo de un golpe." The context-check stage uses keyword-matched search, not bulk loading. This applies to both filesystem artifact scanning and memory search.

---

## Component 1: Context-Check Stage (Change Pipeline)

### 1.1 New Stage Constant

In `internal/changes/types.go`, add:

```go
StageContextCheck ChangeStage = "context-check" // after initial stage, before spec/tasks
```

### 1.2 Updated Flow Registry

In `internal/changes/flows.go`, insert `StageContextCheck` into ALL 12 flows:

```
Fix:
  small:  describe → context-check → tasks → verify
  medium: describe → context-check → spec → tasks → verify
  large:  describe → context-check → spec → design → tasks → verify

Feature:
  small:  describe → context-check → tasks → verify
  medium: propose → context-check → spec → tasks → verify
  large:  propose → context-check → spec → clarify → design → tasks → verify

Refactor:
  small:  scope → context-check → tasks → verify
  medium: scope → context-check → design → tasks → verify
  large:  scope → context-check → spec → design → tasks → verify

Enhancement:
  small:  describe → context-check → tasks → verify
  medium: propose → context-check → spec → tasks → verify
  large:  propose → context-check → spec → clarify → design → tasks → verify
```

Position: ALWAYS second (index 1). After initial stage, before everything else.

### 1.3 Stage Filename

Add to `stageFilenames` map:

```go
StageContextCheck: "context-check.md"
```

### 1.4 Context-Check Tool Architecture

**File**: `internal/tools/context_check.go`

**Dependencies** (constructor injection):
- `changes.Store` — load active change, list completed changes
- `*memory.Store` — search explore observations (nullable — works without memory)

**Constructor**:
```go
type ContextCheckTool struct {
    changeStore changes.Store
    memStore    *memory.Store  // nullable
}

func NewContextCheckTool(cs changes.Store, ms *memory.Store) *ContextCheckTool
```

**MCP Tool Parameters**:
- `change_description` (required, string) — the change description to analyze
- `project_name` (optional, string) — project name for memory search filtering

**Handle logic** (what the tool does INTERNALLY):

1. **Find project root** — `findProjectRoot()` (existing helper)

2. **Scan filesystem artifacts** (targeted, not bulk):
   - Read `sdd/business-rules.md` if exists
   - Read `sdd/requirements.md` if exists
   - Read `sdd/proposal.md` if exists
   - Read `sdd/design.md` if exists
   - List completed changes in `sdd/changes/` — read only `change.json` (title + description), NOT full artifacts

3. **Keyword-match completed changes**: Extract keywords from current change description (split by spaces, filter stop words, lowercase). Match against completed change titles/descriptions. Return only matches (max 10).

4. **Search memory for explore observations** (if `memStore` is not nil):
   - Call `memStore.Search(changeDescription, SearchOptions{Type: "explore", Project: projectName, Limit: 5})`
   - These become "additional context" in the report

5. **Scan convention files** (when no SDD artifacts found):
   - Check for `CLAUDE.md`, `AGENTS.md`, `README.md`, `.cursor/rules/`, `CONTRIBUTING.md`
   - Read first 200 lines of each found file (not full injection)
   - Flag as "project conventions detected but no formal SDD specs"

6. **Build structured report** (returned as tool result — AI uses it to generate the context-check.md artifact):

```markdown
# Context Check Report

## Existing Artifacts Found
- [list of sdd/ artifacts with sizes]

## Relevant Prior Changes
- [keyword-matched completed changes with title + description]

## Explore Context (Memory)
- [explore observations found, if any]

## Convention Files
- [CLAUDE.md, AGENTS.md, etc. — summarized, not full content]

## Ambiguity Flags
[The tool does NOT do ambiguity detection — this is a REPORT tool.
 The AI reads this report and uses the research-backed heuristics
 from server instructions to detect ambiguity and write context-check.md]
```

### 1.5 Critical Design Decision: Tool is a SCANNER, Not an ANALYZER

The context-check tool follows the same pattern as ALL Hoofy tools: **it's a storage/retrieval tool, not an AI tool**. The tool:
- SCANS the filesystem and memory
- RETURNS structured findings
- The AI then ANALYZES the findings using the research-backed heuristics from server instructions
- The AI GENERATES the context-check.md content
- The AI calls `sdd_change_advance` with the content

This means the ambiguity detection heuristics (IEEE 29148 Requirements Smells) live in **server instructions**, not in Go code. This is consistent with how the Clarity Gate works: the tool provides the framework, the AI does the analysis.

### 1.6 Alternative Considered: Tool Does Ambiguity Detection

Rejected because:
- Embedding NLP/regex heuristics in Go adds complexity without proportional value
- The AI is BETTER at natural language analysis than regex patterns
- Server instructions can evolve without recompiling
- Keeps the tool simple (SRP: scan and report, not analyze)

### 1.7 Change Advance Integration

`sdd_change_advance` handles context-check like any other stage:
- Content must be non-empty (existing validation)
- No scoring gate (unlike Clarity Gate) — it's pass/fail based on whether critical issues exist
- The server instructions tell the AI that if the context-check report reveals critical conflicts, it MUST address them before advancing

---

## Component 2: Business Rules Stage (Greenfield Pipeline)

### 2.1 New Stage Constant

In `internal/config/config.go`, add:

```go
StageBusinessRules Stage = "business-rules"
```

### 2.2 Updated Stage Order

```go
var StageOrder = []Stage{
    StageInit,
    StagePropose,
    StageSpecify,
    StageBusinessRules,  // NEW — after specify, before clarify
    StageClarify,
    StageDesign,
    StageTasks,
    StageValidate,
}
```

### 2.3 Stage Metadata

```go
StageBusinessRules: {
    Name:        "Business Rules",
    Description: "Extract and document declarative business rules from requirements",
    Order:       3,  // and bump all subsequent stages +1
},
```

### 2.4 Stage Filename

```go
StageBusinessRules: "business-rules.md",
```

### 2.5 Business Rules Tool

**File**: `internal/tools/business_rules.go`

**Dependencies**: `config.Store`, `templates.Renderer`

**Constructor**: Same pattern as other pipeline tools:
```go
type BusinessRulesTool struct {
    store    config.Store
    renderer templates.Renderer
    bridge   StageObserver  // optional
}

func NewBusinessRulesTool(store config.Store, renderer templates.Renderer) *BusinessRulesTool
```

**MCP Tool Parameters** (structured, per FR-021):
- `definitions` (required, string) — domain terms and their precise definitions (Ubiquitous Language glossary)
- `facts` (required, string) — relationships between domain terms
- `constraints` (required, string) — behavioral boundaries and action assertions (When/Then/Otherwise format)
- `derivations` (optional, string) — computed or inferred knowledge from facts and constraints
- `glossary` (optional, string) — additional domain vocabulary beyond definitions

**Handle logic**:
1. Load config, verify `RequireStage(cfg, StageBusinessRules)`
2. Validate required parameters (definitions, facts, constraints non-empty)
3. Render template with all sections
4. Write to `sdd/business-rules.md`
5. Advance pipeline to `StageClarify`
6. Notify bridge

### 2.6 Template

**File**: `internal/templates/business_rules.go` (or embedded in templates.go)

Two variants: guided and expert.

**Guided template** includes section explanations:
```markdown
# Business Rules

> Business rules are the DNA of your system. They define what is acceptable
> and what is not — across ALL processes, not tied to any single feature.
> (Source: Business Rules Manifesto, Ronald Ross, v2.0)

## Definitions (Ubiquitous Language)
> Terms that everyone in the project must use consistently.
> Example: "Customer" means a person who has completed at least one purchase.

{{ .Definitions }}

## Facts
> Relationships between terms that are always true.
> Example: A Customer has exactly one Account. An Account can have multiple Orders.

{{ .Facts }}

## Constraints
> Behavioral boundaries. Use the format: When <condition> Then <imposition> [Otherwise <consequence>]
> Example: When an Order total exceeds $500, Then manager approval is required.

{{ .Constraints }}

{{ if .Derivations }}
## Derivations
> Knowledge that is computed or inferred from facts and constraints.
> Example: A Customer is "premium" when their total spend exceeds $10,000 in the last 12 months.

{{ .Derivations }}
{{ end }}

{{ if .Glossary }}
## Glossary
> Additional domain terms and abbreviations.

{{ .Glossary }}
{{ end }}
```

**Expert template** is the same without the explanatory blockquotes.

### 2.7 Template Data Structure

Add to `internal/templates/templates.go`:

```go
type BusinessRulesData struct {
    Definitions string
    Facts       string
    Constraints string
    Derivations string
    Glossary    string
}
```

---

## Component 3: Server Instructions Rewrite

### 3.1 Structure

The `serverInstructions()` function is rewritten with research citations. Each section includes:
- The framework/source in a comment
- Concrete examples of good vs bad questions
- Heuristics the AI should apply

### 3.2 New Sections Added

**Context-Check Stage Instructions** (for change pipeline):
```
### Context-Check Stage (Research: IEEE 29148, IREB Elicitation, Bohnner & Arnold IA)

When context-check is the current stage:

1. Call sdd_context_check with the change description and project name
2. Read the returned report carefully — it contains:
   - Existing specs, business rules, and completed changes
   - Explore observations from memory
   - Convention files from project root
3. Analyze for ambiguity using Requirements Smells heuristics (Femmer et al. 2017):
   - Subjective language: "user-friendly", "fast", "easy to use", "intuitive"
   - Ambiguous adverbs: "often", "sometimes", "usually", "typically"
   - Non-verifiable terms: "high quality", "good performance", "secure"
   - Superlatives: "best", "fastest", "most efficient"
   - Negative statements that hide requirements: "the system shall not..."
4. Check for conflicts with existing specs and business rules:
   - Does this change contradict any existing constraint?
   - Does it modify behavior covered by existing requirements?
   - Does it introduce terms not in the Ubiquitous Language glossary?
5. Classify impact using SemVer model:
   - Breaking: changes existing behavior (existing tests would fail)
   - Non-breaking: adds new behavior without affecting existing
   - Patch: internal change, no behavior modification
6. Generate the context-check.md content with your analysis
7. Call sdd_change_advance with the content

If critical issues are found:
- Present them to the user with specific questions
- Wait for answers before generating the context-check.md
- Include the questions and answers in the artifact

If no issues found:
- Generate a brief "all clear" report documenting what was checked
- Proceed to the next stage
```

**Business Rules Stage Instructions** (for greenfield pipeline):
```
### Stage: Business Rules (Research: BRG, Business Rules Manifesto, DDD)

After requirements are specified, extract business rules:

1. Read the requirements (use sdd_get_context stage=requirements)
2. For each requirement, ask: "Is there an implicit business rule here?"
3. Extract rules into four categories (BRG taxonomy):
   - Definitions: What do domain terms MEAN? Build a Ubiquitous Language (DDD, Eric Evans)
   - Facts: What relationships between terms are ALWAYS true?
   - Constraints: What behavior is NOT allowed? Use "When X, Then Y [Otherwise Z]" format
     (Business Rules Manifesto structure)
   - Derivations: What knowledge is COMPUTED from other rules?
4. Present the extracted rules to the user for validation
5. Call sdd_create_business_rules with the validated content
```

### 3.3 Existing Sections Updated

All existing stage instructions get research citations added:

- **Propose**: IREB elicitation techniques reference
- **Specify**: IEEE 29148 quality attributes for requirements, EARS syntax patterns
- **Clarify**: Explicit citation of the 8 dimensions and their research basis
- **Design**: ADR format reference (Michael Nygard)
- **Tasks**: Wave assignment methodology citation

### 3.4 Size Constraint

NFR-004 limits total instructions to 150% of current size. Current `serverInstructions()` is ~308 lines (lines 228-536). Budget: ~462 lines max. The rewrite must be CONCISE — cite sources without quoting full papers.

---

## Component 4: Pipeline Integration Details

### 4.1 Greenfield Pipeline Changes

Files affected:
- `internal/config/config.go` — new stage constant, updated StageOrder, Stages map, stageFilenames
- `internal/pipeline/state.go` — no changes (uses StageOrder dynamically)
- `internal/server/server.go` — register new tool, wire dependencies, update instructions

### 4.2 Change Pipeline Changes

Files affected:
- `internal/changes/types.go` — new StageContextCheck constant
- `internal/changes/flows.go` — update ALL 12 FlowRegistry entries, add stageFilename
- `internal/changes/state.go` — no changes (uses Stages dynamically)
- `internal/server/server.go` — register new tool, wire dependencies

### 4.3 Composition Root Wiring (server.go)

```go
// Context-check tool needs both change store AND memory store
contextCheckTool := tools.NewContextCheckTool(changeStore, memStore)
s.AddTool(contextCheckTool.Definition(), contextCheckTool.Handle)

// Business rules tool follows pipeline pattern
businessRulesTool := tools.NewBusinessRulesTool(store, renderer)
s.AddTool(businessRulesTool.Definition(), businessRulesTool.Handle)
businessRulesTool.SetBridge(bridge)
```

**Important**: `contextCheckTool` must handle `memStore == nil` gracefully (memory subsystem disabled). When nil, it skips explore observation search and just reports filesystem artifacts.

### 4.4 Memory Integration for Context-Check

The context-check tool is registered INSIDE the `if memErr == nil` block for the full version (with memory), and OUTSIDE for the degraded version (without memory). 

Actually, better approach: register it ONCE, pass `memStore` which may be nil. The tool handles nil internally. This is simpler and follows how `ExploreTool` works (it requires memory, so it's inside the block). But context-check should work WITHOUT memory too — it just loses the explore observation search.

```go
// Registered unconditionally — works with or without memory
contextCheckTool := tools.NewContextCheckTool(changeStore, memStore) // memStore may be nil
s.AddTool(contextCheckTool.Definition(), contextCheckTool.Handle)
```

---

## ADR-001: Context-Check as Scanner, Not Analyzer

**Context**: The context-check stage needs to detect ambiguity and conflicts. Two approaches: (A) embed heuristics in Go code, (B) tool scans and reports, AI analyzes.

**Decision**: Option B — tool is a scanner/reporter, AI does analysis guided by server instructions.

**Rationale**: 
- Consistent with how ALL Hoofy tools work (tools are storage, AI generates content)
- AI is better at natural language analysis than regex patterns
- Server instructions can evolve without recompiling
- Keeps the tool simple (SRP)
- The research-backed heuristics (IEEE 29148 Requirements Smells) are better expressed as natural language guidance than as code

**Alternatives rejected**: 
- Embedding NLP regex in Go: adds complexity, fragile, false positives, needs recompile to update
- External NLP service: violates zero-dependency constraint

## ADR-002: Business Rules Before Clarify (Not After)

**Context**: The greenfield pipeline stage order needed to decide whether business-rules comes before or after the Clarity Gate.

**Decision**: Business rules BEFORE clarify: `specify → business-rules → clarify → design`

**Rationale**:
- Business rules are extracted FROM requirements — they need specified requirements as input
- The Clarity Gate should evaluate completeness INCLUDING business rules
- If rules come after clarify, the Clarity Gate can't check whether implicit rules have been captured
- This makes the Clarity Gate more powerful — it now validates both requirements AND rules

**Alternatives rejected**:
- After clarify: Clarity Gate can't evaluate rule coverage
- During clarify: muddies the Clarity Gate's purpose (it scores clarity, not captures rules)

## ADR-003: Keyword-Matched Context, Not Bulk Scan

**Context**: When context-check scans completed changes, it could read all changes or only relevant ones.

**Decision**: Keyword-match only. Extract keywords from the change description, match against completed change titles/descriptions, return max 10 matches.

**Rationale**: Anthropic's guidance — "more context is not better; finite and specific context is better." Bulk injection of 200 completed changes would cause hallucinations, not prevent them. The progressive disclosure pattern (search → retrieve relevant) is the right model.

**Alternatives rejected**:
- Bulk scan all changes: exceeds context window, causes hallucinations
- Most recent N changes: recent ≠ relevant; a change from 6 months ago might be more relevant than yesterday's
