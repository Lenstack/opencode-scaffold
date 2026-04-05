---
name: observability
description: Structured logging, metrics, tracing, alerting conventions for production services
license: MIT
compatibility: opencode
---

# Observability Skill

## Structured Logging

Log WHAT happened, not debug chatter.

```go
// Go - structured key-value pairs
log.Info("resource created", "resource_id", r.ID, "user_id", uid, "duration_ms", elapsed)

// NEVER log: passwords, tokens, PII, full request/response bodies
// NEVER log inside tight loops
```

## Log Levels

- ERROR: something failed that shouldn't (with full error context)
- WARN: degraded but operational (e.g., circuit breaker open)
- INFO: key business events (resource created, user login, payment processed)
- DEBUG: only in development, never in production builds

## Metrics to Always Include

- Request duration (histogram, by endpoint and status)
- Error rate (counter, by type)
- Active connections / queue depth
- Business metrics (orders created, users registered)

## Tracing

- Propagate trace context across service boundaries
- Add spans for DB queries, external HTTP calls, pub/sub
- Tag spans with relevant IDs (user_id, resource_id)

## Alerting Thresholds

- Error rate > 1% for 5 minutes -> page on-call
- p95 latency > 2x baseline -> warning
- p99 latency > 5x baseline -> page
- Panic/crash -> immediate alert

## Health Checks

```
GET /health/live   -> 200 if process is alive
GET /health/ready  -> 200 if can serve traffic (DB connected, etc.)
```
