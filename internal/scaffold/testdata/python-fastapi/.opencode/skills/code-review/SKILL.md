---
name: code-review
description: Structured code review for changed files only — correctness, idioms, maintainability, test quality
license: MIT
compatibility: opencode
---

# Code Review Skill

## Scope

Review ONLY the files listed by the orchestrator (git diff output).
Do NOT read the entire codebase. Ignore unchanged files.

## Dimensions

**Correctness (40%)**
- Does code match acceptance criteria?
- All edge cases from planner covered?
- No logic errors, nil dereferences, off-by-one?
- Error paths handled correctly?

**Idioms (20%)**
- Go: error wrapping, defer, goroutine lifecycle, interfaces
- TypeScript: strict types, no any, proper async/await
- Python: type hints, proper async, context managers
- Language-specific patterns respected?

**Maintainability (20%)**
- Function length: Go <=50 lines, TS/Py <=80 lines
- Cyclomatic complexity: warn if > 10
- No magic numbers/strings — typed constants
- No unnecessary comments (code should be self-explanatory)

**Test Quality (20%)**
- Phase 1 skeletons fully implemented?
- Planner edge cases all covered?
- No anti-patterns (time.Sleep, permanent t.Skip, etc.)?

## Scoring

1=Very Poor, 2=Poor, 3=Acceptable, 4=Good, 5=Excellent

overall = (correctness * 0.4) + (idioms * 0.2) + (maintainability * 0.2) + (test_quality * 0.2)

CHANGES_REQUIRED if: overall < 3.5 OR any blocker exists.

## Output Format (JSON first)

```json
{
  "gate": "code-reviewer",
  "verdict": "LGTM | CHANGES_REQUIRED",
  "scores": {"correctness":4,"idioms":5,"maintainability":4,"test_quality":4,"overall":4.2},
  "blockers": [{"file":"","line":0,"issue":"","fix":""}],
  "suggestions": []
}
```
