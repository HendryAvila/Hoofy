# Clarify: Pipeline Rigor — Ambiguity Resolution

## Questions & Answers

### Q1: Where does `business-rules` sit in the greenfield pipeline?

**Answer**: BEFORE clarify. Updated stage order:
```
init → propose → specify → business-rules → clarify → design → tasks → validate
```
**Rationale**: Business rules are extracted FROM requirements. The Clarity Gate then evaluates requirements + business rules together for completeness. Rules inform clarity, not the other way around.

**Impact**: FR-007 stage order updated. Config `StageOrder` must reflect this.

---

### Q2: Does context-check access `memory.Store` directly or via AI instructions?

**Answer**: Option A — the Go tool receives `memory.Store` as a constructor dependency and performs the search internally.

**Rationale**: User explicitly chose machine enforcement over AI discretion: "Estoy tratando de no dejar tanta libertad a la IA porque me encuentro con muchas ambigüedades."

**Impact**: `ContextCheckTool` struct needs `memory.Store` in addition to `changes.Store`. Registered in `server.go` composition root with both dependencies.

---

### Q3: How many completed changes should context-check scan?

**Answer**: Only keyword-matched changes, NOT all changes.

**Rationale**: Anthropic's guidance on AI context management: "Más contexto no es mejor. Es mejor un contexto finito y específico, no necesariamente corto, pero tampoco intrínsecamente largo." Dumping 200 changes into context causes hallucinations, not prevents them.

**Impact**: Context-check extracts keywords from current change description, then searches completed change artifacts (title, description) for matches. Returns only relevant changes. NFR-002 (500ms bound) is naturally satisfied — keyword search is bounded by result count, not total changes.

**Design note**: This aligns with the existing progressive disclosure pattern in the memory system: targeted search → specific retrieval, never bulk injection.

---

### Q4: Is context-check a blocking gate or just a report?

**Answer**: It's a conditional gate. When 0 issues found → produces "all clear" report → advances normally. When issues found → gates advancement until resolved.

**Rationale**: Unlike Clarity Gate (which ALWAYS requires 8-dimension scoring), context-check is pass/fail. If there's nothing wrong, don't create friction. If there IS something wrong, block.

**Impact**: `sdd_change_advance` for context-check stage validates content but does NOT require a minimum score. It validates that the report was generated (not empty/placeholder) and that no `critical` severity findings remain unresolved.

---

### Q5: What happens when no prior SDD artifacts exist?

**Answer**: Context-check should:
1. Scan project root for convention files: `CLAUDE.md`, `AGENTS.md`, `.cursor/rules/`, `.github/`, `README.md`
2. Extract any implicit business rules, constraints, or conventions from those files
3. If still unclear → ask the user about undocumented business rules
4. Never skip silently — always report what was found (even if "no prior context found, checked: [list of files scanned]")

**Rationale**: A project without SDD artifacts still has context. Convention files contain implicit rules. The user wants active context discovery, not passive "nothing found, moving on."

**Impact**: Context-check needs a fallback scan list: `[CLAUDE.md, AGENTS.md, .cursor/rules/, README.md, CONTRIBUTING.md, .github/CODEOWNERS]`. The scan is keyword-targeted, not full file injection.

---

## Spec Updates Required

Based on clarification answers, the following spec items need revision:

1. **FR-007**: Stage order changes from `clarify → business-rules → design` to `business-rules → clarify → design`
2. **FR-003**: Clarified — `memory.Store` is a direct dependency of the tool, not an AI instruction
3. **FR-002**: Clarified — completed changes are keyword-searched, not bulk-scanned
4. **FR-005**: Clarified — conditional gate (blocks on critical findings, passes on all-clear)
5. **NEW**: Context-check scans project root convention files when no SDD artifacts exist

## Key Design Principle Confirmed

> "Quiero manejar contextos específicos, no inyectar todo de un golpe."

This means:
- Targeted context loading via keyword search (not bulk injection)
- Progressive disclosure pattern (search → retrieve relevant → present)
- Finite, specific context windows (not "dump everything")
- The current memory pattern (context → search → timeline → get_observation) is the RIGHT model — apply to context-check
