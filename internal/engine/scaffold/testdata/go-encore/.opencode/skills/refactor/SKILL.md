---
name: refactor
description: Safe refactoring patterns — green tests first, one change per commit, extract functions/hooks
license: MIT
compatibility: opencode
---

# Refactoring Skill

## Golden Rule

Tests MUST be green before you start refactoring. Get to GREEN first, then refactor.
Never mix refactoring with feature work — separate commits.

## Triggers for Extraction

- Go function > 50 lines -> extract to named functions
- TypeScript component > 80 lines -> extract subcomponents or custom hooks
- Repeated code in 3+ places -> extract shared utility
- Magic values -> typed constants

## Go Patterns

```go
// Before: long function
func (s *Service) Process(ctx context.Context, req *Req) (*Resp, error) {
    // 80 lines...
}

// After: extract and name each step
func (s *Service) Process(ctx context.Context, req *Req) (*Resp, error) {
    if err := s.validateRequest(ctx, req); err != nil { return nil, err }
    item, err := s.fetchItem(ctx, req.ID); if err != nil { return nil, err }
    return s.applyTransformation(ctx, item, req)
}
```

## React Patterns

```tsx
// Extract data logic to custom hook
function useResourceData(id: string) {
    const { data, isLoading, error } = useQuery({ queryKey: ['resource', id], ... })
    return { data, isLoading, error }
}
// Component is now pure presentation
```

## Checklist Per Refactoring Step

- [ ] Tests green before starting
- [ ] Make ONE semantic change
- [ ] Tests still green after
- [ ] No functional change (refactoring != bug fixing)
- [ ] Commit with type: refactor(scope): description
