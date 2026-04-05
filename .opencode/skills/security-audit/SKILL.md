---
name: security-audit
description: OWASP-based security audit for changed files — IDOR, injection, secrets, auth bypass
license: MIT
compatibility: opencode
---

# Security Audit Skill

## Scope

Scan ONLY changed files provided by orchestrator. Adversarial mindset.

## OWASP Top 10 Checks

**A01 Broken Access Control**
- Every endpoint that returns user data: does it filter by user_id = auth.user_id?
- Horizontal privilege escalation (IDOR): can user access other users' resources?
- Vertical privilege escalation: can non-admin access admin endpoints?

**A02 Cryptographic Failures**
- Passwords stored as plain text or weak hash?
- Sensitive data logged?
- TLS enforced for external calls?

**A03 Injection**
- SQL: all queries use parameterised placeholders ($1, ?, :param)?
- No fmt.Sprintf/f-strings building SQL dynamically?
- Command injection: no os.exec with user input?

**A05 Security Misconfiguration**
- Debug mode disabled in production code?
- No hardcoded credentials?
- Error messages leak internal stack traces?

## Static Scan Commands

```bash
# Secrets
grep -rn 'password\s*=\s*["'"'"']\|api_key\s*=\s*["'"'"']\|secret\s*=\s*["'"'"']' $CHANGED_FILES

# SQL injection surface
grep -rn 'fmt\.Sprintf.*SELECT\|fmt\.Sprintf.*INSERT\|f"SELECT\|"SELECT.*"+' $CHANGED_FILES

# Dangerous JS
grep -rn 'dangerouslySetInnerHTML\|eval(\|innerHTML\s*=' $CHANGED_FILES
```

## Output (JSON)

```json
{
  "gate": "security-auditor",
  "verdict": "SECURE | INSECURE",
  "findings": [{"severity":"CRITICAL|HIGH|MEDIUM|LOW","file":"","line":0,"issue":"","fix":""}]
}
```

INSECURE = any CRITICAL or HIGH finding.
