# Proposal: sdd_reverse_engineer

## Problem Statement

When Hoofy is introduced to an **existing project**, the change pipeline (`sdd_change`) operates blind. The `context-check` stage scans for SDD artifacts (`business-rules.md`, `design.md`, `requirements.md`) — but in a project that never went through `sdd_init_project`, **none of these exist**. The fallback reads convention files (CLAUDE.md, README.md), but that's surface-level context — it doesn't capture architecture patterns, business rules, tech stack decisions, or ADRs.

**Result**: When a user says "I want to add a new service", the AI has no way to know that the project uses hexagonal architecture, that services must implement a specific interface, or that a business rule forbids certain behaviors. The new service gets generated in a style that may violate every existing pattern.

## Target Users

- **Developers adopting Hoofy on existing projects** — they already have working code with implicit conventions and want Hoofy to understand their codebase before making changes.
- **Teams onboarding AI assistants** — they need the AI to respect existing architecture, naming, patterns, and rules without manually documenting everything upfront.

## Proposed Solution

A new MCP tool `sdd_reverse_engineer` that acts as a **SCANNER** — it reads key files from the project (directory structure, configs, entry points, convention files, existing ADRs) and returns a **structured report** to the AI. The AI then analyzes this report and generates 3 SDD artifacts:

1. **`sdd/business-rules.md`** — Domain terms, relationships, constraints, derivations (BRG taxonomy)
2. **`sdd/design.md`** — Architecture pattern, tech stack, components, data model, ADRs
3. **`sdd/requirements.md`** — What the system currently does, captured as formal requirements

The tool is DUMB (consistent with Hoofy's design philosophy: tools are storage/scanners, AI generates content). It collects evidence; the AI does the thinking.

### Automatic Trigger in sdd_change

When `sdd_change` is called and no SDD artifacts exist in the project, the tool response **signals that reverse engineering is needed**. The AI should then:
1. Call `sdd_reverse_engineer` to scan the project
2. Analyze the scan results and generate the 3 artifacts (using existing `sdd_create_business_rules`, `sdd_create_design`, `sdd_generate_requirements` — OR writing directly to `sdd/` to avoid pipeline stage requirements)
3. Continue with the change pipeline (context-check will now find the artifacts)

### What the Scanner Reads

The tool scans the following categories of evidence:

1. **Directory structure** — tree output to infer architecture (monorepo, hexagonal, MVC, etc.)
2. **Package manifests** — `package.json`, `go.mod`, `requirements.txt`, `Cargo.toml`, `pom.xml` → tech stack
3. **Convention files** — `CLAUDE.md`, `AGENTS.md`, `README.md`, `CONTRIBUTING.md`, `.cursor/rules/`
4. **Config files** — `tsconfig.json`, `.eslintrc`, `Dockerfile`, `docker-compose.yml`, CI configs
5. **Entry points** — `main.go`, `index.ts`, `app.py`, `cmd/*/main.go` → application structure
6. **Existing ADR files** — `docs/adr/`, `adr/`, `doc/decisions/` → prior architectural decisions
7. **Database schemas** — migrations, schema files, ORM models → data model
8. **API definitions** — OpenAPI/Swagger specs, route files → API contracts

### What the Scanner Does NOT Do
- It does NOT read all source files (too expensive)
- It does NOT run the project or execute any code
- It does NOT make architectural judgments — that's the AI's job
- It does NOT modify any existing files
- It does NOT require `sdd_init_project` to have been run

## Out of Scope

- Analyzing test files for behavioral specifications (future enhancement)
- Generating task breakdowns from existing code (tasks are forward-looking)
- Replacing the full `sdd_init_project` pipeline for new projects
- Running static analysis or linting tools
- Reading private/encrypted files or environment variables

## Success Criteria

1. After `sdd_reverse_engineer` runs on an existing project, `context-check` finds real artifacts with actual architecture and business rules
2. A subsequent `sdd_change` for "add a new endpoint" produces suggestions consistent with the project's existing patterns
3. The scan completes in under 5 seconds for projects up to 10,000 files
4. The scanner degrades gracefully — missing files are skipped, not errors
5. Works with at least: Go, Node.js/TypeScript, Python, and Rust project structures

## Open Questions

- Should the scanner also read a sample of actual source files (e.g., 2-3 representative files per detected component) to better understand coding patterns?
- Should generated artifacts be marked as "auto-generated" to distinguish them from human-curated specs?
- Should sdd_change auto-trigger the reverse engineer, or should it return a message suggesting the user run it manually first?
