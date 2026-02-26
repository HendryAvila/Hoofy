# Specification: Wave Metadata for Task Parallelization

## Functional Requirements

### Must Have

- **FR-001**: `sdd_create_tasks` accepts an optional `wave_assignments` parameter (string, markdown format)
  - When provided, contains wave number assignments for each task (e.g., "TASK-001: Wave 1\nTASK-002: Wave 1\nTASK-003: Wave 2")
  - When omitted, the tool behaves exactly as it does today (backwards compatible)

- **FR-002**: `TasksData` struct includes a `WaveAssignments` field (string)
  - Empty string means no wave metadata (backwards compatible)

- **FR-003**: `tasks.md.tmpl` template renders an "Execution Waves" section when wave data is present
  - When `WaveAssignments` is non-empty, renders a section between Dependency Graph and Acceptance Criteria
  - When `WaveAssignments` is empty, renders nothing (no empty section header, no placeholder)

- **FR-004**: The rendered "Execution Waves" section clearly communicates:
  - Which wave each task belongs to
  - That tasks within the same wave can execute in parallel
  - That wave N+1 starts only after wave N completes

### Should Have

- **FR-005**: `serverInstructions()` includes guidance for the AI to generate wave assignments
  - Explains what waves are (parallel execution groups)
  - Explains the algorithm: no deps = Wave 1, depends only on Wave 1 = Wave 2, etc.
  - Instructs the AI to include wave_assignments when calling `sdd_create_tasks`

- **FR-006**: Wave assignments work naturally with the existing `dependency_graph` parameter
  - The AI derives waves FROM the dependency graph (wave metadata complements, not replaces, the graph)

### Could Have

- **FR-007**: Change pipeline tasks stage content includes wave metadata guidance in server instructions
  - Since `sdd_change_advance` accepts freeform markdown, no tool change needed
  - Guidance tells the AI to include wave sections in tasks stage content

## Non-Functional Requirements

- **NFR-001**: Zero breaking changes — existing tool calls without `wave_assignments` produce identical output
- **NFR-002**: Template rendering with wave data adds < 1ms overhead
- **NFR-003**: No new dependencies (standard Go, existing template engine)

## Constraints

- Must use Go's `text/template` (existing template engine)
- Template must use `{{ if .WaveAssignments }}` conditional rendering
- Parameter is a string (consistent with all other `sdd_create_tasks` parameters)

## Assumptions

- The AI is capable of analyzing task dependencies and assigning wave numbers (this is a reasoning task, not a computation task)
- AI clients that support parallel execution will parse the rendered markdown to extract wave info
- Wave assignments are a hint/recommendation, not a strict execution order