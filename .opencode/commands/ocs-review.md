---
description: Review changed files for quality, security, and correctness
agent: reviewer
---

You are in review mode. Load @skill code-review and @skill security-audit.

Get the list of changed files:
```bash
git diff --name-only HEAD
```

Review ONLY those files. Output structured JSON scorecard followed by prose.
Do NOT modify any files.
