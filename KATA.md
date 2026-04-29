## Kata 005: Rate Limiting (3 hours) — PRIORITY

**Goal**: Protect upstream from abuse.


**Requirements**:
- Token bucket algorithm — **implement it yourself** (don't import)
- Per-API-key rate limit: 100 req/min, burst 10 (match headers from Kata 002)
- Return 429 with `Retry-After` header (seconds until next allowed request)
- In-memory storage only (no Redis for kata scope)
- Cleanup: evict stale buckets (keys inactive > 10 min)

**What You'll Learn**:
- Token bucket math (fill rate vs capacity)
- Per-key state tracking (map[string]*bucket with sync.RWMutex or sync.Map)
- Why 429 + Retry-After matters for API clients (exponential backoff)

**Verification**:
```bash
# 11 rapid requests with same API key
$ for i in {1..11}; do curl -s -H "X-API-Key: test-123" http://localhost:30420/get; done
# First 10: 200 OK
# 11th: 429 Too Many Requests + Retry-After header

# Wait Retry-After seconds, request works again
$ curl -v -H "X-API-Key: test-123" http://localhost:30420/get
# < HTTP/1.1 200 OK
```

**Constraint**: p99 latency overhead < 1ms. Rate limit check must be O(1).

**Time estimate**: 3 hours (the hard kata — this is your differentiator for the interview).

---

## Sunday: Polish & Interview Prep (1-2 hours)

**Required**:
- [ ] `go test ./...` passes with table-driven tests
- [ ] README.md: Architecture diagram (ASCII or drawn), explain RBAC integration points
- [ ] Be ready to discuss: "How would you add per-domain rate limits?" (Real World Scenario has domains)
- [ ] Be ready to discuss: "How would you persist rate limits across restarts?" (Redis? DynamoDB?)

**Optional stretch** (if time permits):
- Webhook signature validation (HMAC-SHA256) — simulates Shopify webhook integration

---

## RBAC Relevance for Interview

**During interview, connect these katas to Real World Scenario's needs**:

| Kata | Real World Scenario |
|------|------------------|
| API Key validation | Customer auth, domain-level permissions |
| Request ID | Tracing emails through pipeline (message-id correlation) |
| Rate limiting | Sending quotas, burst protection, plan enforcement |
| Path routing | API versioning, webhook endpoints |

**Questions to anticipate**:
- "How would you validate webhook signatures from Shopify?" → HMAC validation middleware, similar to your API key check
- "How do you handle per-customer rate limits at scale?" → Token bucket + Redis cluster, similar to your in-memory approach but distributed
- "What happens when a customer hits their quota?" → 429 + Retry-After, exactly what you implemented

---

## Anti-Goals (Don't Do These)

- ❌ WebSockets — irrelevant to email APIs
- ❌ Load balancing — Real World Scenario uses dedicated infrastructure
- ❌ TLS termination — Out of scope, done at edge
- ❌ Import rate limiter libraries — Implement token bucket yourself

---

## Priority Order for Remaining Time

**Saturday evening (2 hours)**: Kata 002 (headers, auth, logging)
**Sunday morning (1.5 hours)**: Kata 003 (path routing)
**Sunday midday-afternoon (3 hours)**: Kata 005 (rate limiting — your talking point)
**Sunday evening (1-2 hours)**: Tests, README, interview prep

---

## Done Definition

- All katas compile: `go build ./...`
- All katas tested: `go test ./...`
- `curl` verification commands pass
- Can explain architecture and tradeoffs in interview
- Tagged releases: `git tag kata-002`, `git tag kata-003`, `git tag kata-005`
