# Go API Gateway Katas

Raw HTTP proxy implementation. No frameworks. No `httputil.ReverseProxy` until Kata 003.

---

## Kata 001: Transparent Forward

**Goal**: Build the dumbest possible HTTP proxy.

**Requirements**:
- Listen on `:30420`
- Accept any HTTP request
- Forward to a hardcoded upstream (e.g., `http://httpbin.org` or local echo server)
- Return upstream's response unchanged
- Use `http.Client` + `io.Copy` — NOT `httputil.ReverseProxy`

**What You'll Learn**:
- Request/response body streaming (why buffering is death)
- Header preservation quirks
- The `Transfer-Encoding: chunked` reality

**Verification**:
```bash
# Should work
$ curl -v http://localhost:30420/get

# Should stream without loading into memory
$ curl http://localhost:30420/bytes/104857600 > /dev/null
```

**Constraint**: Your implementation must handle 100MB download without >10MB RSS growth.

---

## Kata 002: Headers, Identity & WebSockets

**Goal**: Add gateway semantics without breaking the pipe.

**Requirements**:
- Strip client-sent `X-Forwarded-*` headers (security)
- Generate and inject `X-Request-ID` (UUID v4)
- Add `X-Forwarded-For` with actual client IP
- Handle `Connection: Upgrade` for WebSocket passthrough
- Add structured logging: `method path upstream status duration request_id`

**What You'll Learn**:
- `http.Hijacker` interface
- Why WebSocket breaks your simple `io.Copy` model
- Header canonicalization (`X-Request-Id` vs `X-Request-ID`)

**Verification**:
```bash
# Regular request
$ curl -v -H "X-Forwarded-Proto: evil" http://localhost:30420/get
# Response should NOT contain your evil header back

# WebSocket upgrade
$ wscat -c ws://localhost:30420/ws
# Should stay connected, echo messages
```

**Constraint**: WebSocket must survive 5 minutes idle without 502.

---

## Kata 003: Path Routing (Triage)

**Goal**: Single gateway, multiple upstreams.

**Requirements**:
- `/api/*` → `http://localhost:8001` (API server)
- `/static/*` → `http://localhost:8002` (File server)  
- `/*` → `http://localhost:8003` (Default app)
- Implement your own routing table (map + prefix match)
- NOW you may refactor to use `httputil.ReverseProxy` for the actual proxying
- But routing decision must be YOUR code

**What You'll Learn**:
- Routing happens BEFORE proxy dial
- Path stripping vs preserving (`/api/foo` → upstream sees `/foo` or `/api/foo`?)
- Why you need a 404 vs 502 distinction

**Verification**:
```bash
# Starts 3 upstreams on 8001, 8002, 8003
$ ./test-upstreams.sh &

$ curl http://localhost:30420/api/status  # → 8001
$ curl http://localhost:30420/static/main.css  # → 8002  
$ curl http://localhost:30420/  # → 8003
```

**Constraint**: Routing table reloadable without restart (inotify or HTTP POST to reload).

---

## Kata 004: Load Balancing & Health

**Goal**: Single hostname, many backends.

**Requirements**:
- `/api/*` routes to 3 upstreams: `[localhost:8001, localhost:8002, localhost:8003]`
- Round-robin selection using atomic counter (no locks)
- Health check: mark upstream unhealthy if 3 consecutive requests fail
- Retry with different upstream on connection error (not 5xx response)
- Expose `/health` endpoint showing upstream status

**What You'll Learn**:
- Connection pools vs per-request dials
- Why atomic > mutex for simple counters
- Health state machine (healthy → suspect → unhealthy)

**Verification**:
```bash
# Kill one upstream
$ kill $(lsof -ti:8002)

# Requests should still succeed, routed to 8001/8003
$ for i in {1..10}; do curl -s http://localhost:30420/api/status; done

# Check health page
$ curl http://localhost:30420/health
# Shows 8002 as unhealthy
```

**Constraint**: Failed upstream must be retried within 50ms, total p99 latency <200ms with one down node.

---

## Kata 005: Rate Limiting & Resilience (Bonus)

**Goal**: Protect upstream from abuse.

**Requirements**:
- Per-IP rate limit: 100 req/min, burst 10
- Return 429 with `Retry-After` header
- Circuit breaker: if upstream error rate >50%, bypass for 10s
- Graceful shutdown: drain in-flight requests on SIGTERM

**What You'll Learn**:
- Token bucket algorithm (implement it, don't import)
- Circuit breaker state transitions
- `http.Server.Shutdown` vs `http.Server.Close`

---

## Directory Structure

```
go-api-gw/
├── KATA.md              # This file
├── cmd/
│   └── gateway/
│       └── main.go      # Your entry point (you write this)
├── pkg/
│   ├── proxy/           # Kata 001-002
│   ├── router/          # Kata 003
│   ├── balancer/        # Kata 004
│   └── ratelimit/       # Kata 005
├── test/
│   └── upstreams.go     # Test servers (I can provide this)
└── go.mod
```

**Rule**: Commit after each Kata. Tag it: `git tag kata-001` etc.

---

## Anti-Goals (Don't Do These)

- ❌ Use Gin, Echo, Chi, or any framework
- ❌ Use `httputil.ReverseProxy` before Kata 003
- ❌ Import a rate limiter library (write the token bucket)
- ❌ Use Kubernetes/Docker for testing (local processes only)
- ❌ TLS termination (out of scope, Caddy does this upstream)

---

## Evaluation Checklist

Before moving to next Kata:

- [ ] Code compiles with `go build ./...`
- [ ] `go test ./...` passes (write table-driven tests)
- [ ] `curl` test commands succeed
- [ ] Memory stable under `wrk` load test
- [ ] WebSocket survives 5min (if applicable)

---

## Resources (Read In Order)

1. `go doc net/http/httputil ReverseProxy` — read the source, it's short
2. RFC 7230 Section 5 — Message Routing (the actual spec)
3. Cloudflare blog: "How We Built Rate Limiting..." (for Kata 005)
4. Go Concurrency Patterns: Context — for cancellation propagation

---

## Need Help?

Ask about:
- Specific interface signatures to implement
- How to test a specific edge case
- Whether your approach matches Go idioms

Don't ask for:
- Complete working implementations
- "Best practice" library recommendations
- Code reviews of working code (do that after you finish)
