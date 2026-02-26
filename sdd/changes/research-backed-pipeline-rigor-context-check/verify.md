# Verify: Research-Backed Pipeline Rigor

## Requirements Coverage Analysis

### Must Have (17 requirements)

| Req | Description | Covered By | Status |
|-----|-------------|------------|--------|
| FR-001 | Context-check in all 12 flows | TASK-002 | ✅ Covered |
| FR-002 | Read filesystem artifacts | TASK-008 | ✅ Covered |
| FR-003 | Search explore observations from memory | TASK-008 | ✅ Covered |
| FR-004 | Produce structured context-check.md report | TASK-008, TASK-012 | ✅ Covered |
| FR-005 | Gate pipeline advancement | TASK-012 (server instructions) | ✅ Covered |
| FR-006 | IEEE 29148 ambiguity heuristics | TASK-012 (server instructions) | ✅ Covered |
| FR-007 | Business-rules as new stage (8 stages) | TASK-004 | ✅ Covered |
| FR-008 | BRG taxonomy categories | TASK-006, TASK-007 | ✅ Covered |
| FR-009 | Declarative rule format (When/Then/Otherwise) | TASK-006, TASK-013 | ✅ Covered |
| FR-010 | Ubiquitous Language glossary | TASK-006, TASK-007 | ✅ Covered |
| FR-011 | New `sdd_create_business_rules` tool | TASK-007 | ✅ Covered |
| FR-012 | All instructions cite research sources | TASK-012, TASK-013 | ✅ Covered |
| FR-013 | EARS syntax in clarify instructions | TASK-013 | ✅ Covered |
| FR-014 | Context-check instructions with full workflow | TASK-012 | ✅ Covered |
| FR-015 | StageContextCheck in types, flows, filenames | TASK-001, TASK-002 | ✅ Covered |
| FR-016 | StageBusinessRules in config, pipeline | TASK-004 | ✅ Covered |
| FR-017 | All existing tests pass + new tests | TASK-003, 005, 009, 010, 014, 015 | ✅ Covered |

**Must Have Coverage: 17/17 (100%)**

### Should Have (4 requirements)

| Req | Description | Covered By | Status |
|-----|-------------|------------|--------|
| FR-018 | Severity classification (critical/warning/info) | TASK-012 (instructions guide AI) | ✅ Covered |
| FR-019 | Phantom reference detection | TASK-012 (instructions guide AI) | ✅ Covered |
| FR-020 | Good vs bad question examples | TASK-012, TASK-013 | ✅ Covered |
| FR-021 | Structured params for business-rules tool | TASK-007 | ✅ Covered |

**Should Have Coverage: 4/4 (100%)**

### Non-Functional (6 requirements)

| Req | Description | Covered By | Status |
|-----|-------------|------------|--------|
| NFR-001 | Zero external dependencies | TASK-015 (go.mod check) | ✅ Covered |
| NFR-002 | Context-check <500ms for 50 changes | TASK-008 (keyword match, not bulk) | ✅ Covered |
| NFR-003 | SRP, DIP, constructor injection | All tasks follow pattern | ✅ Covered |
| NFR-004 | Instructions ≤150% current size | TASK-012, 013 (462 line budget) | ✅ Covered |
| NFR-005 | Business rules as readable markdown | TASK-006 (templates) | ✅ Covered |
| NFR-006 | Backward compatible tool signatures | TASK-015 | ✅ Covered |

**NFR Coverage: 6/6 (100%)**

## Component Coverage

| Component | Tasks | Status |
|-----------|-------|--------|
| `internal/changes/types.go` | TASK-001 | ✅ |
| `internal/changes/flows.go` | TASK-001, TASK-002 | ✅ |
| `internal/changes/flows_test.go` | TASK-003 | ✅ |
| `internal/config/config.go` | TASK-004 | ✅ |
| `internal/config/config_test.go` | TASK-005 | ✅ |
| `internal/pipeline/state_test.go` | TASK-005 | ✅ |
| `internal/templates/templates.go` | TASK-006 | ✅ |
| `internal/tools/business_rules.go` | TASK-007 | ✅ |
| `internal/tools/business_rules_test.go` | TASK-010 | ✅ |
| `internal/tools/context_check.go` | TASK-008 | ✅ |
| `internal/tools/context_check_test.go` | TASK-009 | ✅ |
| `internal/server/server.go` | TASK-011, 012, 013 | ✅ |
| `internal/tools/change_integration_test.go` | TASK-014 | ✅ |

## Consistency Checks

### ✅ Stage Order Consistency
- Greenfield: init → propose → specify → **business-rules** → clarify → design → tasks → validate (8 stages)
- Change pipeline: all flows have context-check at index 1 (after initial stage)
- No stage name conflicts between the two pipelines

### ✅ Design ↔ Spec Alignment
- Design addresses all Must Have requirements
- Design's ADR-002 reflects the clarify answers (business-rules before clarify)
- Design's ADR-003 reflects Anthropic guidance on context specificity

### ✅ Spec ↔ Proposal Alignment
- Spec includes updated FR-007 stage order (from clarify answers)
- Spec's Won't Have list aligns with proposal's Out of Scope

### ⚠️ Minor Gaps Noted

1. **FR-005 implementation detail**: The spec says context-check "gates advancement" via content validation, but the design says "no scoring gate, it's pass/fail." The gating is done through SERVER INSTRUCTIONS telling the AI to address critical issues before advancing — not through Go code. This is consistent with how FR-018 (severity classification) works: the AI classifies, not the tool. **Verdict: acceptable** — machine enforcement here means the AI MUST call sdd_context_check (the tool exists in the flow, can't skip it), and the instructions guide what the AI does with the report.

2. **Template tests**: TASK-006 mentions template tests but there's no dedicated TASK for template test file. Tests are included in TASK-006's acceptance criteria. **Verdict: acceptable** — template tests are part of TASK-006, not a separate task.

3. **Instructions line budget**: The 462-line budget (NFR-004) is tight with all the new content. May need to condense existing sections to make room. **Verdict: monitor during TASK-012/013** — if budget is exceeded, trim verbose sections.

## Risk Assessment

1. **Low**: Context-check keyword matching may produce false positives (irrelevant matches). Mitigation: max 10 results cap + the AI filters relevant ones when generating the report.

2. **Low**: Business-rules stage adds overhead to greenfield pipeline. Mitigation: it's a single stage with clear purpose — not busywork. The user confirmed "no se siente pesado."

3. **Medium**: Server instructions line budget may be tight. Mitigation: condense existing sections, use bullet points over prose, cite sources inline not in separate paragraphs.

4. **Low**: Existing tests may need updates beyond what's listed (cascading from stage order changes). Mitigation: TASK-015 runs full test suite as final safety net.

## Verdict

**PASS** ✅

All 27 requirements (17 Must Have + 4 Should Have + 6 NFR) are covered by tasks. All components have assigned tasks. No critical inconsistencies. Three minor gaps noted — all acceptable and monitored. The implementation can proceed.
