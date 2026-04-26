# Go API Gateway Katas — Mailgun Interview Prep

Scoped for ~6-8 hours (Saturday evening → Sunday). Focus: API gateway patterns relevant to Mailgun's email API business.

---

## Kata 001: Transparent Forward ✅ COMPLETE

**Goal**: Basic HTTP proxy with streaming.

**Status**: Done. Tagged `kata-001`.

---

## Kata 002: Headers, Identity & API Auth (2 hours)

**Goal**: Gateway semantics for API traffic (not WebSockets — irrelevant to email APIs).

**Why for Mailgun**: Email APIs need request tracking, authentication, and audit logging for compliance.

**Requirements**:
- Strip client-sent `X-Forwarded-*` headers (security hygiene)
- Generate and inject `X-Request-ID` (UUID v4) for distributed tracing
- Add `X-Forwarded-For` with actual client IP (audit trail)
- **NEW**: Validate `X-API-Key` header — reject with 401 if missing/invalid (simulate Shopify integration auth)
- Structured logging: `timestamp method path api_key status duration request_id`

**What You'll Learn**:
- Header canonicalization pitfalls (`X-Api-Key` vs `X-API-Key`)
- Request context injection (passing request_id downstream)
- Authentication middleware patterns

**Verification**:
```bash
# Should work (valid key)
$ curl -v -H "X-API-Key: valid-key-123" http://localhost:30420/get

# Should reject (no key)
$ curl -v http://localhost:30420/get
# < HTTP/1.1 401 Unauthorized

# Should reject (bad key)
$ curl -v -H "X-API-Key: invalid" http://localhost:30420/get

# Request ID present in logs
$ go run cmd/gateway/main.go 2>&1 | grep "X-Request-ID"
```

**Constraint**: Log format must be parseable (space-delimited or JSON). Include request_id in response headers so clients can correlate.

**Time estimate**: 2 hours (auth validation is the new piece here).

---

## Kata 003: Path Routing for API Versioning (1.5 hours)

**Goal**: Route by path — critical for API evolution at Mailgun scale.

**Why for Mailgun**: `/v3/messages` vs `/v4/messages` — versioned endpoints hit different backends.

**Requirements**:
- `/v1/*` → `http://localhost:8001` (legacy API)
- `/v3/*` → `http://localhost:8002` (current API)
- `/v4/*` → `http://localhost:8003` (beta API)
- `/*` (no version prefix) → `http://localhost:8002` (default to current)
- Implement your own prefix matcher (map[string]http.Handler)
- **Path stripping**: `/v3/messages` → upstream sees `/messages` (not `/v3/messages`)

**What You'll Learn**:
- Routing is pre-proxy decision
- Path manipulation (strip version prefix before forwarding)
- 404 vs 502 distinction (routing miss vs upstream down)

**Verification**:
```bash
# Start test upstreams
$ go run test/upstreams.go

$ curl http://localhost:30420/v1/domains    # → upstream-8001, sees /domains
$ curl http://localhost:30420/v3/messages   # → upstream-8002, sees /messages
$ curl http://localhost:30420/v4/webhooks   # → upstream-8003, sees /webhooks
$ curl http://localhost:30420/messages      # → upstream-8002 (default)
$ curl http://localhost:30420/v2/legacy     # → 404 (version not supported)
```

**Constraint**: Router reloadable via HTTP POST `/admin/reload` (no restart for adding new versions).

**Time estimate**: 1.5 hours.

---

## Kata 004: SKIP

Load balancing is infrastructure-level. Mailgun would use Envoy/HAProxy for this. Skip — not relevant to backend engineering interview.

---

## Kata 005: Rate Limiting (3 hours) — PRIORITY

**Goal**: Protect upstream from abuse. **This is Mailgun's core business** (email sending quotas, API rate limits).

**Why for Mailgun**: Customers have sending limits. Burst protection prevents noisy neighbors. 429 + `Retry-After` is the standard email API pattern.

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
- [ ] Be ready to discuss: "How would you add per-domain rate limits?" (Mailgun has domains)
- [ ] Be ready to discuss: "How would you persist rate limits across restarts?" (Redis? DynamoDB?)

**Optional stretch** (if time permits):
- Webhook signature validation (HMAC-SHA256) — simulates Shopify webhook integration

---

## RBAC Relevance for Interview

**During interview, connect these katas to Mailgun's needs**:

| Kata | Mailgun Parallel |
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
- ❌ Load balancing — Mailgun uses dedicated infrastructure
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
