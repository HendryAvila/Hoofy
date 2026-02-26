# Context Check: Wave Orchestration Instructions

## Existing Context

### Current Wave Instructions (server.go, lines 793-801)
There is already a "Wave Assignments in Tasks Stage" section that tells the AI HOW TO GENERATE waves. But it says NOTHING about how to EXECUTE them. This is the gap we're filling.

### Prior Change: `add-wave-metadata-to-task-output-for-parallel` (completed)
This change added the wave_assignments parameter to `sdd_create_tasks` and the instruction block at lines 793-801. It focused on GENERATION of waves, not execution. Our change is the natural continuation.

### Sub-Agent Namespace Section (server.go, lines 572-607)
Already describes multi-agent patterns with namespace isolation. Our orchestration instructions should reference this — when launching parallel agents, each should use its own namespace.

### Server Instructions Structure
The instructions block runs from ~line 300 to ~line 801. It covers:
- Project pipeline (stages, how to generate content)
- Change pipeline (flows, stages, how to generate content)
- Memory tools usage
- Context engineering features (detail_level, namespace, max_tokens, progress)
- Wave generation instructions (the 9-line block we're extending)

## Impact Analysis

### Files Affected
- `internal/server/server.go` — ONLY file. Adding ~30-40 lines of instructions.

### Risk Assessment
- **LOW RISK**: Instructions-only change, no logic, no tools, no tests needed
- The server instructions are a raw string — adding a section cannot break anything
- No schema changes, no API changes, no migration needed

## Key Design Decisions Needed

1. **Where to place** the new section — immediately after the existing "Wave Assignments" block (line 801) is natural
2. **How to describe capabilities** without naming specific clients
3. **Whether to reference namespace** for parallel agent memory isolation
4. **Level of specificity** — too vague = AI ignores it, too prescriptive = doesn't adapt to different clients
