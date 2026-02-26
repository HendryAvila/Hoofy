# Spec: Research-Backed Pipeline Rigor

## Must Have

### Context-Check Stage (Change Pipeline)

- **FR-001**: The change pipeline MUST include a `context-check` stage in ALL 12 flow variants (4 types × 3 sizes). It is inserted AFTER the initial stage (describe/propose/scope) and BEFORE the next stage (spec/tasks/design).
- **FR-002**: The context-check stage MUST read existing filesystem artifacts from `sdd/` — including `proposal.md`, `requirements.md`, `business-rules.md`, `design.md`, and all completed change artifacts in `sdd/changes/*/`.
- **FR-003**: The context-check stage MUST search persistent memory for `sdd_explore` observations filtered by: (a) current project name, (b) recent observations only. These are presented as "additional context", not primary source.
- **FR-004**: The context-check stage MUST produce a structured context report as its output artifact (`context-check.md`), containing: (a) existing artifacts found, (b) ambiguity flags detected in the change description, (c) potential conflicts with existing specs/rules, (d) impact classification (breaking/non-breaking/patch), (e) clarifying questions if any.
- **FR-005**: The context-check stage MUST gate pipeline advancement — the AI cannot call `sdd_change_advance` for context-check without producing a substantive context report. This is enforced by the tool (content validation), not by instructions alone.
- **FR-006**: Ambiguity detection MUST use IEEE 29148 Requirements Smells heuristics: flag subjective language, ambiguous adverbs/adjectives, superlatives, negative statements, comparative phrases, non-verifiable terms, and terms implying totality.

### Business Rules Artifact (Greenfield Pipeline)

- **FR-007**: The greenfield pipeline MUST include a new `business-rules` stage. The stage order becomes: init → propose → specify → clarify → **business-rules** → design → tasks → validate (8 stages total).
- **FR-008**: The business-rules artifact (`sdd/business-rules.md`) MUST categorize rules using BRG taxonomy: Definitions (terms), Facts (relationships between terms), Constraints/Action Assertions (behavioral boundaries), and Derivations (computed/inferred knowledge).
- **FR-009**: Each business rule MUST follow declarative structure: "When \<condition\> Then \<imposition\> [Otherwise \<consequence\>]" — per Business Rules Manifesto (Ronald Ross, v2.0).
- **FR-010**: The business-rules artifact MUST include a Ubiquitous Language glossary section capturing domain terms and their precise definitions (per DDD / Eric Evans).
- **FR-011**: A new MCP tool `sdd_create_business_rules` MUST be created to save the business rules artifact and advance the pipeline. It follows the same pattern as other stage tools (AI generates content, tool saves it).

### Server Instructions Rewrite

- **FR-012**: ALL question frameworks in `serverInstructions()` MUST cite their research source. Accepted sources: IREB, IEEE 29148, EARS (Rolls-Royce), Business Rules Manifesto (BRG/Ronald Ross), Requirements Smells (Femmer et al. 2017), Event Storming (Alberto Brandolini), DDD (Eric Evans).
- **FR-013**: The clarify stage instructions MUST use EARS syntax patterns (Ubiquitous, State-driven, Event-driven, Optional, Unwanted behavior) as a framework for requirement validation.
- **FR-014**: The context-check stage instructions MUST guide the AI through: (1) artifact scanning, (2) ambiguity detection using Requirements Smells heuristics, (3) conflict detection via traceability analysis, (4) impact classification using SemVer model, (5) question generation for unresolved issues.

### Pipeline Integrity

- **FR-015**: The new `context-check` stage MUST be added to the `ChangeStage` enum, `FlowRegistry`, `stageFilenames`, and all related type/validation code in the `changes` package.
- **FR-016**: The new `business-rules` stage MUST be added to `config.StageOrder`, `config.Stages`, and all related pipeline configuration in the `config` and `pipeline` packages.
- **FR-017**: All existing tests MUST continue to pass after the changes. New tests MUST cover: (a) all 12 updated flow variants, (b) context-check stage advancement, (c) business-rules stage in greenfield pipeline, (d) the new MCP tool.

## Should Have

- **FR-018**: The context-check report SHOULD include a severity classification for each finding: `critical` (blocks advancement — must be resolved), `warning` (should be addressed but doesn't block), `info` (for awareness only).
- **FR-019**: The context-check SHOULD detect when a change description references components or requirements IDs (FR-XXX) that don't exist in current specs — catching "phantom references."
- **FR-020**: Server instructions SHOULD include concrete examples of good vs bad questions for each stage, demonstrating research-backed questioning (e.g., "Instead of asking 'who are the users?', ask 'What are the top 3 tasks your primary user performs daily?' — per IREB elicitation techniques").
- **FR-021**: The business-rules tool SHOULD accept categorized parameters (definitions, facts, constraints, derivations, glossary) rather than a single content blob, matching the structured approach of other pipeline tools like `sdd_create_design`.

## Non-Functional Requirements

- **NFR-001**: Zero external dependencies — context-check reads filesystem + SQLite memory. No new dependencies in `go.mod`. CGO_ENABLED=0 must remain.
- **NFR-002**: Context-check filesystem scan MUST complete in under 500ms for a project with up to 50 completed changes and 10 spec files — no recursive directory walks without bounds.
- **NFR-003**: All new code MUST follow existing conventions: SRP (one file per tool), DIP (depend on interfaces), constructor injection via `server.go` composition root.
- **NFR-004**: Server instructions total size SHOULD NOT exceed 150% of current size — research citations must be concise, not verbose academic prose.
- **NFR-005**: Business rules artifact MUST be human-readable markdown — no custom DSL, no YAML, no JSON. Developers should be able to read and edit it directly.
- **NFR-006**: Backward compatibility — all existing MCP tool signatures remain unchanged. New tools and parameters are additive only.

## Could Have

- **FR-022**: Context-check COULD generate a visual dependency map showing which existing requirements/rules are affected by the change.
- **FR-023**: Business rules COULD include traceability links back to the requirements they were derived from (e.g., "Derived from FR-003").
- **FR-024**: The server instructions COULD include a "research appendix" section listing all cited sources with links, so curious users can read the original papers/frameworks.

## Won't Have (This Version)

- **FR-025**: Will NOT implement automated code analysis — context-check reads specs, not source code.
- **FR-026**: Will NOT implement Gherkin/BDD format for business rules — markdown is lighter and sufficient.
- **FR-027**: Will NOT implement semantic/vector search for conflict detection — FTS5 keyword search + structured comparison is sufficient (per previous YAGNI decision).
- **FR-028**: Will NOT add a dedicated "impact analysis" tool — impact classification is embedded within context-check, not a standalone tool.

## Assumptions

- The `sdd/` directory structure already exists when context-check runs (the change pipeline requires `sdd_init_project` or a previous change to have created it).
- Business rules are generated by the AI based on conversation with the user — the tool is a storage tool, same pattern as all other pipeline tools.
- A project may NOT have business rules yet (e.g., first change before greenfield pipeline completes) — context-check handles this gracefully by noting "no business rules found."

## Constraints

- Must build with Go 1.25, `CGO_ENABLED=0`
- Must follow `mcp-go v0.44.0` SDK patterns for tool registration
- Server instructions are a single string returned by `serverInstructions()` — no external files
- All stage artifacts are markdown files in `sdd/` or `sdd/changes/<slug>/`

## Dependencies

- `changes` package (flows, types, state machine) — core modifications
- `config` package (stage order for greenfield pipeline) — new stage addition
- `pipeline` package (if business-rules stage needs clarity-gate-like behavior)
- `tools` package (new tool files)
- `templates` package (new templates for business-rules)
- `server` package (composition root wiring + server instructions rewrite)
- `memory` package (context-check needs to query explore observations)
