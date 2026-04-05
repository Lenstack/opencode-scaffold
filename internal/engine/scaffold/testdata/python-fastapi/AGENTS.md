# Project Production Rules
# Stack: Python + FastAPI | Framework: fastapi | Template: Standard Production | Generated: 2026-04-05
#
# This file is loaded by OpenCode via "instructions" in opencode.json.
# All agents read these rules. Edit freely — commit to Git.

## Stack Context
- Backend: python
- Frontend: N/A
- Framework: fastapi
- Database: false

## Agent Pipeline (execute in order, no exceptions)

```
[User Request]
      |
      v
  orchestrator (primary agent)
      |
      +- Phase 0: @explore   -> run "ocs discover" to index project
      |                       load heuristics from "ocs memory list --tier heuristic"
      |
      +- Phase 1: @planner   -> acceptance criteria, edge cases, task breakdown
      |
      +- Phase 2: @architect -> ADR, data model, API contract
      |
      +- Phase 3: @tester    -> Phase 1: write FAILING tests (TDD contract)
      |
      +- Phase 4: implement  -> against failing tests
      |
      +- Phase 5: @tester    -> Phase 2: execute tests (must be GREEN)
      |           RED -> self-heal -> retry Phase 4 (max 2 attempts)
      |
      +- Phase 6: @reviewer + @security (parallel)
      |           Each scans CHANGED FILES ONLY
      |           FAIL -> unified fix -> retry once
      |
      +- Phase 7: cleaner (bash: gofmt/eslint/debug-removal)
      |
      +- Phase 8: Definition of Done (10 checks via bash)
      |
      +- Phase 9: @reflector -> update memory via "ocs memory" commands
```

## Non-Negotiable Rules (ALL agents must respect)

1. **TDD mandatory**: tester writes failing tests BEFORE any implementation
2. **No hardcoded secrets**: use env vars / config / secrets managers only
3. **No debug code in production**: no fmt.Println, console.log, debugger in non-test files
4. **All DB migrations additive**: never DROP COLUMN, RENAME COLUMN, or DROP TABLE
5. **All TODOs need ticket numbers**: TODO(#123) not bare TODO
6. **ADR for every design decision**: write to docs/adr/NNNN-slug.md
7. **Reflector runs after EVERY task**: memory must be updated
8. **Changed files only in reviews**: never re-read the full codebase in gates
9. **Skills loaded on-demand**: agents use the skill tool, not _index.md
10. **Self-heal max 2 retries**: then escalate to user with full error

## Definition of Done (10 items — orchestrator validates all via bash)

1. No console.log/fmt.Println in production files (grep verified)
2. No hardcoded secrets (grep verified)
3. All DB migrations are additive (grep verified)
4. All new public functions have tests (coverage verified)
5. All planner acceptance criteria addressed
6. ADR created or updated
7. CHANGELOG.md updated
8. No TODO without ticket number
9. Backend build passes
10. Frontend build passes (if applicable)

## Memory Protocol

Memory is stored in LevelDB at .opencode/data/ — managed by the ocs binary.

- **Tier 1 (Episodic)**: TTL 7 days — query via "ocs memory search --tier episodic"
- **Tier 2 (Semantic)**: TTL 90 days, confidence-scored — query via "ocs memory search --tier semantic"
- **Tier 3 (Heuristics)**: Permanent, promoted by @dreamer — query via "ocs memory list --tier heuristic"
- **Quarantine**: Facts with confidence < 0.60 after 14 days — auto-pruned

Reflector must:
- Run "ocs memory prune" to clean expired entries
- UPSERT semantic facts (new=0.50, confirmed=+0.25, contradicted=-0.20)
- Move facts with confidence < 0.60 to quarantine
- Check if dream_needed (candidates with session_count >= 3)

## Discovery

Project indexing is handled by the ocs binary:
- Run "ocs discover" for full reindex
- Run "ocs discover --incremental" for changed files only (uses checksum)
- Results stored in LevelDB at .opencode/data/
- No external dependencies (no Python3, no shell scripts)
