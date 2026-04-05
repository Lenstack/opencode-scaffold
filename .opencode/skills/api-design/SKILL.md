---
name: api-design
description: REST API design conventions — naming, versioning, error shapes, pagination, idempotency
license: MIT
compatibility: opencode
---

# API Design Skill

## Resource Naming

- Plural nouns: /resources, /users, /orders
- Nested for ownership: /users/{id}/orders
- Actions as verbs only when no noun fits: /auth/refresh

## HTTP Methods

- GET: read (idempotent, cacheable)
- POST: create or non-idempotent action
- PUT: full replacement (idempotent)
- PATCH: partial update (idempotent)
- DELETE: remove (idempotent)

## Status Codes

```
200 OK           - GET, PUT, PATCH success
201 Created      - POST success (return created resource)
204 No Content   - DELETE success
400 Bad Request  - validation error (return field-level errors)
401 Unauthorized - missing/invalid auth
403 Forbidden    - valid auth, insufficient permission
404 Not Found    - resource doesn't exist
409 Conflict     - unique constraint, optimistic lock
422 Unprocessable- semantic validation failure
429 Too Many Req - rate limit hit
500 Internal     - unexpected server error (never leak internals)
```

## Error Response Shape (consistent)

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "human-readable message",
    "details": [{"field": "name", "issue": "required"}]
  }
}
```

## Pagination - keyset (not OFFSET)

```json
{"data": [...], "cursor": "opaque_next_cursor", "has_more": true}
```

## Idempotency

- Provide Idempotency-Key header for POST endpoints that create resources
- Return cached response if key already seen (within 24h)

## Versioning

- URL versioning: /v1/resources, /v2/resources
- Never break v1 - add fields, never remove
