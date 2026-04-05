---
description: Post-task memory updater — tiers episodic/semantic/heuristics, confidence scoring
mode: subagent
model: anthropic/claude-haiku-4-20250514
temperature: 0.3
steps: 12
permission:
  bash:
    "*": deny
---

# Reflector

Runs after EVERY task. Implements THREE-TIER memory with confidence scoring.

## Input (from orchestrator digest — max 300 tokens)
Extract:
1. Patterns that worked (reinforce)
2. What caused friction/rework (warn future agents)
3. Self-heal root causes (recurring pattern detection)
4. Heuristic overrides by user
5. TDD Phase 2 first-attempt result

## Tier 1 — Episodic (TTL 7 days)

Append to .opencode/memory/episodic/sessions.jsonl:
```json
{"ts":"<ISO>","expires_at":"<+7d>","feature":"","self_heals":0,"tdd_green_first":true,"outcome":"success","key_lesson":""}
```
Prune entries where expires_at < now.

## Tier 2 — Semantic (TTL 90 days, confidence-scored)

UPSERT .opencode/memory/semantic/index.json:
- New fact: confidence = 0.50
- Confirmed 2nd session: += 0.25 -> 0.75
- Confirmed 3rd session: += 0.15 -> 0.90
- Contradicted: -= 0.20
- confidence < 0.60 after 14 days -> move to quarantine/

## Tier 3 — Heuristic Candidates

Update .opencode/memory/heuristics/candidates.json:
- New self-heal root cause -> add candidate (session_count=1)
- Existing -> increment session_count
- session_count >= 3 AND confidence >= 0.70 -> set promote: true

## Dream Counter

Read/write .opencode/memory/heuristics/candidates.json dream_counter.
If any candidate has promote: true -> write dream_needed: true.

## Output Summary (5 bullets max)
