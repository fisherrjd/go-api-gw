# Solutions: Sync Primitives

## Exercise 3 — Spot the bug

```go
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
    s2 := *s         // copies Server struct — including its sync.Mutex
    s2.mu.Lock()     // locks s2's COPY of the mutex, not s's mutex
    defer s2.mu.Unlock()
    s2.counter++     // mutates s2, not s — counter on original never changes
}
```

**Three things wrong:**
1. `*s` copies the struct, copying the mutex — two goroutines can both "hold the lock" (on different copies).
2. `s2.counter++` mutates the copy, not the original.
3. `go vet` reports: `passes lock by value: Server contains sync.Mutex`.

**Fix:** operate on `s` directly.
```go
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.counter++
}
```

---

## WaitGroup: Add before go, Done via defer

```
// Correct order:
wg.Add(1)       // 1. increment
go func() {     // 2. start goroutine
    defer wg.Done()  // 3. decrement when goroutine exits
    ...
}()
wg.Wait()       // 4. block until zero
```

If Add is inside the goroutine, it races with Wait — the goroutine may not start before Wait sees zero and returns.

---

## When to use which

| Primitive | Use when |
|-----------|----------|
| `sync.WaitGroup` | Waiting for N goroutines to finish |
| `sync.Mutex` | Protecting shared state with mixed reads/writes |
| `sync.RWMutex` | Many reads, infrequent writes — readers don't block each other |
| `sync.Once` | Lazy singleton init, one-time setup |

---

## Interview tip: mutex on structs

If a struct has a `sync.Mutex` field:
- All methods that touch protected fields **must** use pointer receivers.
- Never pass the struct by value — pass a pointer or embed behind an interface.
- `go vet` reports the copy as a warning. Interviewers plant this bug because vet catches it in real code.
