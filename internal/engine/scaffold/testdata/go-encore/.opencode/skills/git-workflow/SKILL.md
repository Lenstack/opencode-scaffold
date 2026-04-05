---
name: git-workflow
description: Git workflow conventions — conventional commits, branching, PR process, release
license: MIT
compatibility: opencode
---

# Git Workflow Skill

## Commit Format (Conventional Commits)

```
<type>(<scope>): <description>

[optional body]

[optional footer: BREAKING CHANGE, fixes #123]
```

Types: feat, fix, docs, test, perf, refactor, chore, ci, build, revert

Examples:
- feat(auth): add JWT refresh token endpoint
- fix(users): prevent IDOR on profile update
- test(orders): add edge cases for empty cart

## Branch Strategy

```
main          <- production (protected, never direct push)
  +- feat/feature-slug      <- feature branches
  +- fix/bug-slug           <- bug fixes
  +- chore/task-slug        <- maintenance
```

## PR Checklist

- [ ] Tests pass (CI green)
- [ ] No console.log/fmt.Println in production files
- [ ] No hardcoded secrets
- [ ] CHANGELOG updated
- [ ] ADR created if design decision was made
- [ ] Self-review done

## Release Process

1. Update CHANGELOG (Keep a Changelog format)
2. Bump version in go.mod/package.json/Cargo.toml
3. git tag vX.Y.Z
4. Push tag -> CI creates release

## Do Not

- Force push to main
- Merge without passing CI
- Commit secrets or large binaries
