---
name: tdd-workflow
description: TDD protocol for Go + Encore — write failing tests first, execute after implementation, report JSON
license: MIT
compatibility: opencode
---

# TDD Workflow Skill

## The Contract

Tests ARE the specification. Implementation exists to make tests pass.
Never write implementation before Phase 1 tests exist and fail.

## Phase 1 — Failing Test Skeletons

Write tests that express intent but FAIL because implementation doesn't exist.

Go example:
```go
func TestCreateResource_Valid(t *testing.T) {
    t.Fatal("not implemented: waiting for backend agent")
}
func TestCreateResource_EmptyName(t *testing.T) {
    t.Fatal("not implemented: waiting for backend agent")
}
```

## Phase 2 — Execution

Run the test suite and capture output:
```bash
encore test ./... 2>&1 | tee /tmp/test-output.txt
echo "Exit: $?"
```

Required structured output:
```json
{"phase":2,"exit_code":0,"verdict":"GREEN|RED","failures":[],"output_tail":"<last 30 lines>"}
```

## Test Coverage Requirements

Every public function needs:
- Happy path with valid inputs
- Each planner edge case
- Each error path (not found, conflict, invalid input)
- Authorization failure (if auth protected)

## Anti-patterns (never do)

- time.Sleep in tests (use mocks/channels)
- t.Skip permanently (comment with reason + ticket)
- Broad catch-all assertions
- Tests that depend on execution order
