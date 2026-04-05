---
name: performance
description: Performance analysis — DB queries, bundle size, goroutine lifecycle, React rendering
license: MIT
compatibility: opencode
---

# Performance Skill

## Database

**N+1 Anti-pattern** (the most common perf killer):
```go
// BAD: N+1
for _, order := range orders {
    items, _ := db.QueryRow(ctx, "SELECT * FROM items WHERE order_id = $1", order.ID)
}

// GOOD: batch
items, _ := db.Query(ctx, "SELECT * FROM items WHERE order_id = ANY($1)", orderIDs)
```

**Pagination**: Always keyset, never OFFSET for large tables:
```sql
-- OFFSET becomes slow at scale
SELECT * FROM resources ORDER BY id LIMIT 20 OFFSET 10000;

-- Keyset cursor stays fast
SELECT * FROM resources WHERE id > $1 ORDER BY id LIMIT 20;
```

**Index checklist**: Every WHERE, JOIN, ORDER BY column should be indexed.

## Go Backend

- Goroutines must have guaranteed termination (context cancellation)
- No mutex held during I/O -> use channels
- Large structs in hot paths -> pass by pointer
- JSON marshalling heavy -> consider protobuf or sonic

## React / Next.js

Bundle budgets:
- Initial JS: < 150 KB gzip
- Per-route JS: < 100 KB gzip

```tsx
// Heavy component -> dynamic import
const HeavyChart = dynamic(() => import('./HeavyChart'), { ssr: false })

// Always next/image, never raw <img>
import Image from 'next/image'

// Stable query keys (avoid object literals inline)
const queryKey = useMemo(() => ['resource', id], [id])
```
