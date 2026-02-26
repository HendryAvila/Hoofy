# Proposal: Research-Backed Pipeline Rigor

## Problem Statement

Hoofy's change pipeline currently has a blind spot: when a change is created, the AI jumps straight into documentation stages (describe/propose/spec) without first checking whether the change conflicts with existing specifications, breaks established business rules, or contains ambiguous language that will propagate through every subsequent stage. This is the "garbage in, garbage out" problem — if the initial change description is vague or contradicts existing specs, the entire pipeline produces high-quality documentation of the WRONG thing.

Additionally, the greenfield pipeline captures requirements and design but never explicitly extracts and documents **business rules** — the declarative constraints that govern acceptable system behavior across all processes. These rules end up buried in requirements prose, making them invisible for impact analysis during future changes.

## Target Users

- **Corporate development teams** using Hoofy in regulated environments where AI-generated ambiguity is unacceptable. They need machine-enforced gates, not "hope the AI asks good questions."
- **Solo developers and small teams** building products iteratively who make many small-to-medium changes. They need the pipeline to catch conflicts with existing specs BEFORE they write code, not after.
- **AI agents** (Claude, Copilot, Cursor) consuming Hoofy's MCP tools. They need structured, research-backed question frameworks in server instructions — not vague "ask good questions" guidance.

## Proposed Solution

### 1. Context-Check Stage (Change Pipeline)

A new machine-enforced stage called `context-check` inserted into change flows AFTER the initial description/proposal/scope stage and BEFORE spec/tasks. This stage:

- **Reads existing artifacts**: Scans `sdd/` for existing specs, business rules, design docs, and completed changes
- **Detects ambiguity**: Uses IEEE 29148 Requirements Smells heuristics (subjective language, ambiguous adverbs, non-verifiable terms) on the change description
- **Checks for conflicts**: Cross-references the change intent against existing business rules and requirements (traceability analysis per Bohnner & Arnold)
- **Classifies impact**: Uses SemVer-inspired model (breaking/non-breaking/patch) to flag changes that may affect existing behavior
- **Gates advancement**: The stage produces a context report and may require the user to answer clarifying questions before proceeding

### 2. Business Rules Artifact (Greenfield Pipeline)

A new `business-rules.md` artifact generated during the greenfield pipeline (after requirements, before or during clarify). This artifact:

- **Extracts rules from requirements**: Transforms implicit rules buried in FR/NFR prose into explicit, declarative statements
- **Uses BRG taxonomy**: Categorizes rules as Definitions, Facts, Constraints, or Derivations (per Business Rules Group / Ronald Ross)
- **Structures rules declaratively**: Uses "When <condition> Then <imposition> Otherwise <consequence>" format (per Business Rules Manifesto)
- **Builds a Ubiquitous Language glossary**: Captures domain terms and their precise definitions (per DDD / Eric Evans)
- **Becomes the reference for context-check**: Future changes are validated against this artifact

### 3. Research-Backed Server Instructions

Rewrite ALL `serverInstructions()` question frameworks with cited, research-based methodologies:

- **EARS syntax** (Rolls-Royce, used by NASA/Airbus) for requirement patterns
- **IEEE 29148 quality attributes** for requirement validation
- **IREB elicitation techniques** for structured questioning
- **Requirements Smells** (Femmer et al. 2017) for automated ambiguity detection heuristics

## Out of Scope

- **Semantic/vector search over specs**: YAGNI — FTS5 keyword search + the structured context-check stage is sufficient. Already cancelled this in a previous decision.
- **Automated code analysis**: The context-check reads SPECS, not code. Code analysis is a different tool's job.
- **Gherkin/BDD format for business rules**: Too heavy. Markdown with structured sections is lighter and sufficient for our use case (lesson from Cliplin analysis — their Gherkin approach adds complexity without proportional value for an MCP server).
- **ChromaDB or external dependencies**: Zero-dependency constraint. Everything stays in SQLite + filesystem.
- **Breaking changes to existing tool APIs**: All existing MCP tool signatures remain backward-compatible. New parameters are optional.

## Success Criteria

1. **Context-check stage blocks advancement** when it detects unresolved ambiguity or conflicts with existing specs/rules — machine-enforced, not optional AI behavior
2. **Business rules are extractable** from any completed greenfield pipeline as a standalone `business-rules.md` artifact with categorized, declarative rules
3. **Every question framework** in server instructions cites its research source (IREB, IEEE, EARS, BRM, etc.) — zero "invented" frameworks
4. **All existing tests pass** — the new stage integrates cleanly into the flow registry and state machine without breaking existing flows
5. **Change pipeline flows are updated** for ALL 12 type×size combinations with the context-check stage inserted at the appropriate position

## Open Questions

- Should context-check be present in ALL flows (including small fixes), or only medium and large? Small fixes (describe → tasks → verify) might not benefit from a full context check — but even a lightweight "did you check existing specs?" gate could prevent regressions.
- Should business-rules.md be a separate pipeline stage or generated as part of the existing clarify stage? Adding a whole new stage to the greenfield pipeline (8 stages) is heavier — but cramming it into clarify muddies the Clarity Gate's purpose.
- How should the context-check interact with `sdd_explore`? Explore captures pre-pipeline context — should context-check consume explore observations automatically?
