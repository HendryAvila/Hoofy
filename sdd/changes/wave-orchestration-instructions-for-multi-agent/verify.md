# Verify: Wave Orchestration Instructions

## Checklist

- [x] **New section added after line 801** — "Wave Execution — Multi-Agent Orchestration" at line 803
- [x] **Client-agnostic** — No mention of Claude Code, Cursor, Codex, Antigravity, or any specific client. Uses generic capability tiers (Agent Teams, Sub-Agents, Sequential)
- [x] **All three tiers described** — Tier 1 (shared task list + inter-agent comms), Tier 2 (parallel workers, report back), Tier 3 (sequential fallback)
- [x] **References namespace** — Tier 1 uses `namespace="agent/<task-id>"`, Tier 2 uses `namespace="subagent/<task-id>"`
- [x] **References mem_progress** — All tiers use it for tracking wave/task completion
- [x] **Dependency enforcement** — "NEVER start Wave N+1 until every task in Wave N has succeeded"
- [x] **File conflict prevention** — Step 3 explicitly warns about overlapping file ownership
- [x] **Ask the user** — "If you are unsure which tier you support, ASK THE USER before proceeding"
- [x] **Natural language** — Reads as guidance, not rigid rules
- [x] **Build passes** — `go build ./...` ✅
- [x] **Tests pass** — `go test -race ./...` ✅ (all packages green)
- [x] **No breaking changes** — Instructions-only, no tool/API/schema changes

## Scope Validation

- Only `internal/server/server.go` modified — matches scope from describe stage
- ~47 lines added — within estimated 30-40 range (slightly over due to formatting)
- No new tools, no output format changes, no client-specific code

## Verdict

**PASS** — All acceptance criteria met. The instructions are client-agnostic, cover all three orchestration tiers, reference existing features (namespace, mem_progress), and enforce dependency safety.
