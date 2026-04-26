> **Context:** Senior engineer coming from Python. ~Zero Go experience. 6 days out (Sun–Fri), ~3 hrs/day. Interview format: given a problem, troubleshoot / identify the issue. Mixed API + concurrency flavor expected.
> 

> **Plan shape:** Days 1–3 build Go reading fluency. **Day 4 is pure bug-hunt reps** — the highest-ROI day. Day 5 applies those reps to real code. Day 6 is mock interview + consolidation. Do not dilute this by spreading theory across all 6 days.
> 

## Mailgun domain context (useful flavoring)

Mailgun = email infrastructure. High-throughput, deliverability, SMTP, queues, webhooks, retries. The Go debugging problem will likely *look* like one of these shapes:

- A message handler or worker consuming from a queue
- A rate limiter or throttle (per domain, per sender)
- A retry loop with backoff
- A webhook dispatcher fanning out events
- An SMTP-ish state machine or parser
- A small HTTP API handler (auth, validation, enqueue)

None of these are Mailgun-specific — they're standard backend shapes. But recognizing the *domain* faster means you spend less cognitive load figuring out what the code does and more on finding the bug.

## Strategic framing

You're not trying to pass as a Go developer. You're trying to do three things:

1. **Read Go fluently enough** that syntax doesn't eat your debugging time
2. **Know the 10–15 idiomatic footguns** that turn into "find the bug" problems
3. **Demonstrate your debugging process** — which is where your seniority actually shows

Your FastAPI + Spring background gets you ~70% there. You already think in typed request/response models, dependency-injected services, middleware/filter chains, and request-scoped context — all of that survives the language swap. The trap is the 30% where Go looks similar but behaves differently: nil interfaces, value vs pointer receivers, goroutine closures, zero values, error wrapping. That's where bugs hide and what you drill.

## Concept bridge: FastAPI / Spring → Go

| You know (FastAPI) | You know (Spring) | Go equivalent |
| --- | --- | --- |
| `@app.get("/x")` route | `@GetMapping("/x")` | `mux.HandleFunc("GET /x", h)` (Go 1.22+ method-aware routing) or chi's `r.Get("/x", h)` |
| Pydantic `BaseModel` | Java DTO + Jackson | Plain struct with `json:"field"` tags; `json.Unmarshal(body, &dto)` |
| `SQLModel` / `SQLAlchemy` model | JPA `@Entity` | Struct + `database/sql` or `sqlc`/`ent`/`gorm` (no magic — explicit queries) |
| `Depends(get_db)` DI | `@Autowired` / constructor injection | Constructor injection by hand: `NewHandler(db, logger) *Handler`; no framework |
| FastAPI middleware / `BaseHTTPMiddleware` | Servlet `Filter` / Spring `HandlerInterceptor` | `func(next http.Handler) http.Handler` — literally a function wrapping a function |
| `HTTPException(401)` | `ResponseStatusException` | `http.Error(w, "...", http.StatusUnauthorized)` — no exceptions, you write the response |
| `async def`  • `await` | `CompletableFuture` / `@Async` | Goroutines + channels; no `await` keyword, concurrency is explicit |
| Request object `request: Request` | `HttpServletRequest` | `*http.Request` — note `r.Context()` for cancellation, `r.Body` is an `io.ReadCloser` (you must close it) |
| `BackgroundTasks` | `@Async` methods | `go doWork(ctx)` — but you own the lifecycle; no framework will clean up after you |
| Pydantic validation errors | `@Valid`  • `MethodArgumentNotValidException` | Manual validation or libraries like `go-playground/validator`; no magic |
| `pytest` fixtures | JUnit `@BeforeEach` | `testing.T`, table-driven tests, `t.Cleanup()` — no fixtures, just helper funcs |

**Biggest mental shifts from both stacks:**

- **No framework does DI, validation, or error handling for you.** What Spring does with annotations and FastAPI does with `Depends` + Pydantic, Go does with explicit function calls and `if err != nil`. This is a feature, not a bug — but in an interview, a bug often lives in the ceremony that the framework *would* have handled for you.
- **No exceptions.** Every function that can fail returns `(result, error)`. Spring's `try/catch` and FastAPI's `HTTPException` have no equivalent. Forgetting to check `err` is the #1 Go bug.
- **Request lifecycle is explicit.** Spring manages request scope for you; FastAPI handles async contexts. In Go, you pass `ctx := r.Context()` manually to every downstream call. If you don't, cancellation and timeouts silently break.
- **Closing things is your job.** `r.Body`, `*sql.Rows`, HTTP response bodies from clients — all need explicit `defer x.Close()`. Spring's try-with-resources and FastAPI's context managers don't exist here.
- **Structs replace classes.** No inheritance. Composition via embedding. Interfaces are implemented implicitly (no `implements` keyword) — if the methods match, the type satisfies the interface. This flips Spring's explicit interface-programming model.

---

## Day 1 (Thu 4/23, 3 hrs) — Go for Python people, fast

**Goal:** read any Go file without syntax being a speed bump.

- **90 min** — [A Tour of Go](https://go.dev/tour) — skip exercises, just read. You'll finish it.
- **45 min** — Write a tiny HTTP handler with middleware on [Go Playground](https://go.dev/play). Smallest auth-ish thing possible: middleware that checks a header, handler that returns JSON. Feel the muscle of `http.HandlerFunc`, `http.ServeMux`, `json.Marshal`, struct tags.
- **45 min** — Read stdlib `net/http` handler example + one real middleware (chi or gorilla/mux README examples). Not to memorize — to recognize shapes.

**Exit criteria:** `func (s *Server) handleX(w http.ResponseWriter, r *http.Request)` reads as easily as a FastAPI route or a Spring `@RestController` method.

---

## Day 2 (Fri 4/24, 3 hrs) — Errors, interfaces, nil: the Python-trap zone

This is where Python intuition actively misleads you. Spend real time here.

- **60 min — Error handling idioms.** `if err != nil`, wrapping with `fmt.Errorf("...: %w", err)`, `errors.Is`, `errors.As`, sentinel vs typed errors. Common bugs: ignored errors, shadowed `err` in `:=`, wrapping vs not wrapping.
- **60 min — Interfaces and nil.** The infamous **typed nil** problem — an interface holding `(*MyError)(nil)` is NOT equal to `nil`. Classic interview bug. Also: implicit interface satisfaction, accept interfaces / return structs.
- **60 min — Pointers vs values.** When does a method mutate? Pointer vs value receivers. Slices are weird (header is a value, backing array is shared) — `append` can or cannot mutate the caller's slice depending on capacity. Maps are reference-like. Bug goldmine territory.

**Exit criteria:** you can explain the typed-nil trap out loud and predict whether a given method call mutates its receiver.

---

## Day 3 (Sat 4/25, 5–6 hrs) — Concurrency primitives [WEEKEND — go deeper]

You have more time today. Don't use it for more theory — use it to **build and break** concurrent programs. That's where concurrency intuition actually forms.

**Core (3 hrs, same as before):**

- **60 min** — goroutines, channels (buffered vs unbuffered), `select`, `context.Context` — especially `ctx.Done()` and cancellation propagation. You already debug this conceptually in the auth gateway; just learn Go's spelling.
- **45 min — The classic concurrency bugs:**
    - Loop variable capture in goroutines (fixed in Go 1.22, still interview fodder — know both versions)
    - Forgotten `wg.Done()` / deadlock
    - Send on closed channel (panics)
    - Deadlock on unbuffered channel with no receiver
    - Goroutine leaks from un-cancelled contexts
    - Data races on shared maps / shared state
- **45 min** — `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`. When each is appropriate. Common bug: **copying a struct that contains a mutex** (vet will warn you; interviewers love this).
- **30 min** — Run `go run -race` on something with an intentional race. See what the race detector output looks like. Interviewers sometimes hand it to you.

**Weekend bonus (2–3 hrs):**

- **60–90 min — Build something small and concurrent.** Pick one:
    - A **worker pool**: N goroutines consuming from a job channel, with graceful shutdown via context cancellation
    - A **fan-out/fan-in pipeline**: input channel → N workers → output channel, with error propagation
    - A **rate limiter** using a ticker + buffered channel (token bucket-ish)
    
    Write it, run it, break it deliberately (remove a `wg.Done()`, close a channel twice, forget `ctx` cancellation), watch what happens. This is worth more than 3 hours of reading.
    
- **45–60 min — Read goroutine-heavy stdlib.** Skim `net/http`'s `Server.Serve` loop or `context` package source. See how the stdlib actually structures goroutine lifecycle.
- **30 min — Race detector practice.** Write 2–3 programs with deliberate races. Run with `-race`. Learn to read the output fast — races are a common interview bug and the output is distinctive.

**Exit criteria:** you can sketch the 5 most common goroutine bugs without notes, AND you've built and debugged at least one real concurrent program.

---

## Day 4 (3 hrs) — Deliberate debugging practice

Stop reading, start doing. You need reps finding bugs, not more theory.

- **90 min — Bug hunts.** Pick a repo of intentionally buggy Go. Options:
    - [gophercises](https://gophercises.com) has bug-fix style exercises
    - Search GitHub for "go interview bugs" / "find the bug go"
    - Ask an LLM to generate buggy Go snippets matching the categories above and solve them timed (10 min each)
- **60 min — Read real production-ish Go.** Skim [chi](https://github.com/go-chi/chi) middleware package, or a small auth library like [golang-jwt/jwt](https://github.com/golang-jwt/jwt). Trace one full request path. Goal: get comfortable navigating an unfamiliar Go codebase quickly — that's literally the interview.
- **30 min — Debugging toolkit recap.** `go vet`, `go run -race`, `go test -run`, `fmt.Printf("%+v\n", x)` for structs, `%#v` for full Go syntax. `log/slog` basics. Know how to drop a `panic("here")` as a quick probe.

**Exit criteria:** you've solved 5+ bug-hunt snippets end to end.

---

## Day 5 (3 hrs) — Mock the interview + consolidate

- **90 min — Full mock.** Have an LLM or friend hand you a ~50–150 line Go program with 2–3 planted bugs. Think out loud the whole time. Narrate:
    1. What the code is supposed to do
    2. What you'd check first (inputs? goroutine lifecycle? error paths?)
    3. Your hypothesis → how you'd verify it → fix
    
    **Do this twice.** Once API-flavored, once concurrency-flavored.
    
- **60 min — Cheat sheet.** Write your own one-pager. Act of writing is the point, not the artifact. Sections: syntax cribs, top 10 Go bugs, debugging commands, "things I'd ask the interviewer" (inputs, expected behavior, error contract, concurrency assumptions).
- **30 min — Rest / light review.** Don't cram. Re-read the cheat sheet once before bed. Sleep matters more than one extra hour. Interview is tomorrow.

---

## The debugging playbook (the thing that actually wins)

This is where your seniority shows. When they drop code in front of you:

1. **Clarify the contract first.** "What's this supposed to do? What inputs? What's the observed vs expected behavior?" Don't skip this — it's what seniors do and juniors don't.
2. **Read before running.** Skim the whole thing top to bottom. Identify: entry points, goroutines, shared state, error paths, external calls.
3. **Form a hypothesis out loud.** "My suspicion is X because of Y. I'd verify by Z."
4. **Check the Go-specific suspects in order:**
    - Error handling: is `err` checked? shadowed? wrapped correctly?
    - nil: any interface comparisons? typed-nil risk?
    - Receivers: value receiver where pointer was intended (silent non-mutation)?
    - Goroutines: who cancels? who waits? shared state protected?
    - Channels: who closes? buffered or not? could this deadlock?
    - Slices/maps: capacity surprises? concurrent map access?
5. **Narrate the fix and the why.** Don't just patch — explain the root cause.

---

## Go-specific bug cheat sheet (memorize)

- `for i, v := range xs { go func() { use(v) }() }` — pre-1.22 captures loop variable; all goroutines see last value. Fix: `v := v` shadow or arg-pass.
- `var err *MyError = nil; var e error = err; e == nil // false` — typed nil. An interface is nil only if both type and value are nil.
- Value receiver `func (s MyStruct) Set(x int) { s.x = x }` — doesn't mutate the caller's struct.
- `append(s, x)` may or may not share backing array with `s`. Don't assume.
- `map` is not safe for concurrent write. Use `sync.Map` or mutex.
- Sending on a closed channel panics. Closing a channel twice panics. Only the sender should close.
- `context.Background()` in a request handler instead of `r.Context()` — breaks cancellation.
- `defer` in a loop: resources held until function returns, not iteration. Classic file-handle leak.
- Shadowed `err` inside `if x, err := f(); err != nil { ... }` — outer `err` never sees the value.
- Copying `sync.Mutex` by value — lock doesn't travel. `go vet` catches this.

---

## What to say if you get stuck in the interview

- "I'm less familiar with Go idioms specifically, but the shape of this bug reminds me of [X problem in Python/gateway work]. Let me trace through it."
- Own the gap, pivot to your debugging process. Seniors being honest about a language gap while demonstrating strong fundamentals reads way better than faking it.

---

## Resources

- [A Tour of Go](https://go.dev/tour) — syntax primer
- [Effective Go](https://go.dev/doc/effective_go) — idioms; skim, don't read cover to cover
- [Go by Example](https://gobyexample.com) — quick reference for any construct
- [Dave Cheney's blog](https://dave.cheney.net) — error handling and idioms
- [Go Playground](https://go.dev/play) — test snippets instantly
- `go vet`, `go run -race`, `go test ./...` — your debugging triad

---

# Quizlet Interview Prep (Sr Engineer — New Community Feature)

> **Format:** Onsite-style block. 3 rounds: System Design, Coding-Backend Applied Screen, Cross-Functional Partners.
> 

> **Language for coding screen:** Python (play to your strength — don't be a hero with Go).
> 

> **Overlap strategy:** Frontload Go through Fri. Pivot fully Sat (Day 7). Weave 15 min/day of behavioral story capture starting now.
> 

## Why the role context matters

A "new community feature" at Quizlet = **consumer-scale, social-graph, read-heavy, UGC territory.** This is *not* your Charter auth gateway domain. You need to mentally re-index on a different problem space. The interviewer expects you to think in terms of feeds, fanout, moderation, and engagement — not identity providers and policy enforcement.

**Your edge:** most candidates hand-wave auth, rate-limiting, and abuse-prevention in social system designs. You won't. Lean on this.

## System Design: topics to own

The community round will almost certainly be a variant of "design Twitter / Reddit / Instagram feed / a comment system" flavored for study groups. Be fluent in:

- **Feed generation** — fanout-on-write vs fanout-on-read vs hybrid. When each makes sense (celebrity problem, active vs passive users).
- **Social graph storage** — follows/friends tables, adjacency lists, denormalized follower counts. Read patterns dominate.
- **UGC pipeline** — post → validate → persist → fanout → notify. Usually async via queue (SQS/Kafka).
- **Moderation** — async classifier pipeline, report queue, shadow-ban patterns, spam/abuse signals.
- **Notifications** — fanout, batching, delivery (push/email/in-app), read state tracking.
- **Comments/threads** — nested reply models (adjacency list vs path enumeration vs closure table), pagination strategies.
- **Engagement primitives** — likes/reactions (counter design: eventual consistency, Redis counters, periodic flush).
- **Caching layers** — Redis for hot feeds, CDN for static UGC, write-through vs cache-aside, invalidation strategies.
- **Search over UGC** — Elasticsearch/OpenSearch basics, indexing lag, not necessarily full-text (tags/topics often sufficient).
- **Rate limiting** — token bucket, sliding window, per-user vs per-IP. **Your auth-gateway background shines here.**

## System Design: the round itself (playbook)

1. **Clarify requirements first** (same move as the Go debug playbook):
    - Functional: what does "community" mean here? Posts? Study groups? Comments? DMs?
    - Scale: DAU? Posts/day? Read:write ratio? (Usually 100:1+ for social.)
    - Non-functional: latency targets, consistency requirements, availability SLO.
    - Out of scope: what's explicitly NOT being designed?
2. **Sketch the API** — 3–5 endpoints. This anchors the design.
3. **High-level architecture** — clients → LB → API gateway → services → data stores + async pipeline.
4. **Data model** — pick 2–3 core tables, talk about access patterns, partition/shard keys. **If interviewer asks about DB choice, have a reason.** (Postgres for relational + JSON, Cassandra/DynamoDB for write-heavy fanout, Redis for hot data.)
5. **Deep dive on 1–2 areas** — usually feed generation and/or moderation. Interviewer will steer.
6. **Discuss trade-offs explicitly** — "I chose X because Y, tradeoff is Z." Seniors name tradeoffs; juniors pick and move on.
7. **Address the non-functionals** — auth, rate limiting, abuse, observability, deploy/rollback. **This is where your Charter background differentiates you.**

## Coding-Backend Applied Screen

**Use Python.** Reasoning: cognitive load on the applied screen should go to problem-solving, not syntax recall. You're a senior Python engineer — use that.

- Expect a realistic backend-flavored problem, not a LeetCode puzzle. Examples: parse a log file and compute stats, implement a small rate limiter, build a small cache with TTL, process a stream of events.
- Practice: 3–4 backend-flavored problems from [exercism.io](http://exercism.io) [Python track](https://exercism.org/tracks/python/exercises) or HackerRank backend categories.
- **The narration muscle from Go Day 4–6 applies directly.** Clarify contract → hypothesis → implement → test → refine.
- Don't forget: error handling, edge cases (empty input, malformed input, large input), and basic test cases. Seniors write tests unprompted.

## Cross-Functional Partners Interview

This round is behavioral with a collaboration focus. They want signal on: how you work with PMs, designers, EMs, other eng teams; how you handle disagreement; how you drive alignment; how you communicate upward.

**Core story categories to have written** (not just in your head — **written**):

- Time you disagreed with a PM/EM and how it resolved
- Time you drove cross-team alignment on a technical decision
- Time you advocated for a non-obvious technical approach and convinced others
- Time you had to say no / push back on scope
- Time you mentored or unblocked a teammate
- Time a project went sideways and how you handled it
- Time you delivered bad news (missed deadline, tech debt, security issue)
- Time you influenced without authority

**STAR++ format** — Situation, Task, Action, Result, **plus what you learned and what you'd do differently.** That last part is the senior differentiator.

## Behavioral story capture — 15 min/day starting NOW

This is the single highest-ROI prep item on the entire page. Stories feel crisp in your head and then fall apart under time pressure. Writing them forces structure.

- **Sub-page or bullet list:** "Quizlet behavioral stories"
- **Each day Sun–Fri, spend 15 min** jotting one story in STAR++ format. By Friday you'll have 6 stories written. Goal is 8–10 by Quizlet day.
- Tag each story with which categories it answers. Good stories cover 2–3 categories.
- Don't over-polish. Bullet points are fine. You're building recall scaffolding, not writing an essay.

## Overlap schedule (Go + Quizlet, 8 days)

| Day | Go prep | Quizlet prep (+15 min) |
| --- | --- | --- |
| Sun (1) | Day 1: Go syntax | Start behavioral story list; write 1 story |
| Mon (2) | Day 2: errors/nil/interfaces | Write 1 story |
| Tue (3) | Day 3: concurrency | Write 1 story |
| Wed (4) | Day 4: bug-hunt reps | Skim 1 system design writeup (feed design) |
| Thu (5) | Day 5: real code + toolkit | Write 1 story |
| Fri (6) | Day 6: mock + **Charter interview** | Off — rest after Charter |
| Sat (7) | — | **Full pivot:** 90m system design deep dive + 60m applied coding warmup + 30m behavioral review |
| Sun (8) | — | 1 full system design mock + finalize stories + light coding / **interview day if scheduled** |

**You have exactly ONE day (Wed 4/29) between interviews.** That's your real Quizlet study day. You'll be tired from Tuesday's Mailgun interview — that's fine, focused prep beats exhausted cramming. Don't try to add more theory; use the day for deep work on the canonical problem + behavioral rehearsal.

## Day 6 (Wed 4/29) — detailed Quizlet pivot

Your only dedicated Quizlet day. You'll be tired from Mailgun. Focus beats volume.

- **90 min — System design deep dive.** Pick one canonical problem ("Design Twitter" or "Design Reddit comments") and do a full whiteboard pass, out loud, on paper. Then read 1–2 high-quality writeups ([System Design Primer](https://github.com/donnemartin/system-design-primer), [ByteByteGo free articles](https://blog.bytebytego.com/)) and compare.
- **45 min — Applied coding warmup in Python.** 1–2 medium backend-flavored problems. Narrate out loud. Write tests.
- **30 min — Finalize behavioral stories.** Fill coverage gaps. Practice 2–3 weakest out loud, timed to 2 min each.
- **15 min — Logistics + rest.** Confirm interview details, set up environment, sleep early.

## Thu 4/30 morning (interview day)

- Light review of your behavioral story list only. No new material.
- Re-read your system design playbook steps. Don't try to cram topics.
- Protein + hydration + show up focused.

## Resources for Quizlet prep

- [System Design Primer](https://github.com/donnemartin/system-design-primer) — canonical free resource
- [ByteByteGo blog](https://blog.bytebytego.com/) — free articles are high quality
- [Hello Interview](https://www.hellointerview.com/) — system design walkthroughs, many free
- [High Scalability archives](http://highscalability.com/) — real-world architectures (Instagram, Reddit, Twitter case studies are gold)
- [StaffEng](https://staffeng.com/) — framing for senior/staff-level narratives

## Integration points (where your Charter work helps)

When the Quizlet system design comes up, weave these in naturally — they're differentiators:

- **Auth/session** for the community service — don't hand-wave it. You build this for a living.
- **Rate limiting** per user and per endpoint to prevent abuse. Token bucket in Redis.
- **Abuse prevention** — velocity checks, reputation scores, shadowbans.
- **Observability** — structured logging, metrics, tracing. What would you alert on?
- **Gradual rollout** — feature flags, canary, blast radius. Senior signal.

These are the parts every junior skips. Your gateway work makes them natural.
