# Go sync Primitives: WaitGroup, Mutex, RWMutex, Once

## Prerequisites

- [go-goroutines-channels-context.md](go-goroutines-channels-context.md) — goroutines and channels
- Familiarity with Java's `synchronized`, `ReentrantLock`, or `CountDownLatch` is useful but not required

## Core concepts

- **sync.WaitGroup** — tracks in-flight goroutines; blocks the caller until all finish (Java: `CountDownLatch`)
- **sync.Mutex** — exclusive lock; one goroutine holds it at a time (Java: `synchronized`/`ReentrantLock`)
- **sync.RWMutex** — multiple concurrent readers OR one exclusive writer; use when reads heavily outnumber writes
- **sync.Once** — runs a function exactly once across all concurrent callers; canonical lazy-init pattern
- **Mutex-copy footgun** — copying a `sync.Mutex` copies its internal state, breaking mutual exclusion; always use pointer receivers

## Mental model

A `WaitGroup` is a counter with a gate: add before launching, decrement on finish, block at the gate until zero. A `Mutex` is a token — only one goroutine holds it; all others wait in line. An `RWMutex` is a library: many people can read simultaneously, but a writer must clear the room. A `Once` is a light switch that can only be flipped on — the first goroutine to try flips it; all others just see it's already on and move on.

```
WaitGroup  →  counter-with-gate: Add(n) / Done() / Wait()
Mutex      →  exclusive token:   Lock() / Unlock()
RWMutex    →  library model:     RLock()/RUnlock() (readers) vs Lock()/Unlock() (writers)
Once       →  one-time switch:   Do(fn) — first caller runs fn; all others skip
```

## Worked examples

### Example 1: WaitGroup — fan out and join

```go
var wg sync.WaitGroup

for i := 0; i < 5; i++ {
    wg.Add(1)       // increment BEFORE launching — if inside goroutine, Add and Wait can race
    go func(n int) {
        defer wg.Done() // first line; runs even on panic
        fmt.Printf("worker %d\n", n)
    }(i)
}

wg.Wait() // blocks until counter reaches zero
fmt.Println("all done")
```

Never pass a `WaitGroup` by value — its internal state won't be shared. Pass a pointer or embed it in a struct accessed by pointer.

### Example 2: Mutex — thread-safe counter

```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Inc() { // pointer receiver — critical
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

func (c *Counter) Get() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.value
}
```

**The mutex-copy footgun:**

```go
// BROKEN — value receiver copies the struct (and the mutex inside it)
func (c Counter) BadInc() {
    c.mu.Lock()   // locks a fresh copy — original mutex is untouched
    c.value++     // increments the copy — original value is untouched
    c.mu.Unlock()
}
```

`go vet` reports: `passes lock by value: Counter contains sync.Mutex`. This is a real interview trap — know it cold.

### Example 3: RWMutex — read-heavy cache

```go
type Cache struct {
    mu    sync.RWMutex
    store map[string]string
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock()         // multiple goroutines can hold RLock at the same time
    defer c.mu.RUnlock()
    v, ok := c.store[key]
    return v, ok
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()          // exclusive — blocks all readers and other writers
    defer c.mu.Unlock()
    c.store[key] = value
}
```

When to use `RWMutex` vs `Mutex`: only when profiling shows contention and reads genuinely dominate. A plain `Mutex` is simpler and sufficient for most cases.

### Example 4: Once — lazy singleton initialization

```go
var (
    db       *sql.DB
    dbOnce   sync.Once
)

func getDB() *sql.DB {
    dbOnce.Do(func() {
        var err error
        db, err = sql.Open("postgres", dsn)
        if err != nil {
            panic(err) // panic propagates; subsequent calls will retry
        }
    })
    return db
}
```

This is the correct version of Java's double-checked locking — no `volatile`, no subtle reordering bugs. The Go memory model guarantees that all goroutines see the fully initialized value after `Do` returns.

## Common misconceptions

**"I can use a value receiver on a struct that contains a Mutex."**  
No. A value receiver copies the struct. The copy gets its own Mutex, providing zero mutual exclusion on the original. Always use pointer receivers on types that embed sync primitives.

**"RWMutex is always faster than Mutex."**  
Only when reads heavily outnumber writes and under measurable contention. `RWMutex` has higher internal overhead. Profile first; optimize second.

**"sync.Once swallows panics in the initialization function."**  
No — the panic propagates to the caller. And because the initialization is not considered complete after a panic, the next caller will try again. This can cause repeated panics if the init function is inherently broken.

**"I can copy a WaitGroup between goroutines."**  
Copying a `WaitGroup` (or `Mutex`, `RWMutex`, `Once`) copies internal state at that moment — any pending waits, lock state, etc. All sync types from the `sync` package must not be copied after first use. Pass pointers.

**"wg.Add(1) inside the goroutine is fine."**  
No — `Add` must be called *before* `go`, not inside the goroutine. If `Wait` is called before the goroutine gets scheduled, `Add` never increments the counter and `Wait` returns early, racing with the goroutine.

## Check-yourself questions

1. Where must `wg.Add(1)` appear relative to the `go` statement, and why?
2. What does `go vet` report when you pass a `sync.Mutex` by value to a function?
3. Write a correct thread-safe `Get` method on a struct with a `sync.RWMutex`.
4. Why is `sync.Once` preferred over a `sync.Mutex`-protected boolean flag for lazy initialization?
5. A goroutine panics before calling `wg.Done()`. What happens to `wg.Wait()`?
6. Under what conditions does `RWMutex` perform *worse* than `Mutex`?
7. If `sync.Once.Do(fn)` panics, what happens on the next call to `Do`?
8. What is the idiomatic fix for a struct method that has a value receiver on a type containing a `sync.Mutex`?

<details>
<summary>Answers</summary>

1. Before the `go` statement — in the goroutine's parent goroutine. If placed inside the goroutine body, `Wait()` can return before `Add()` runs because the goroutine hasn't been scheduled yet.
2. `go vet` reports `passes lock by value: T contains sync.Mutex` (or similar). The copy receives its own independent mutex, breaking mutual exclusion.
3. `func (c *Cache) Get(key string) (string, bool) { c.mu.RLock(); defer c.mu.RUnlock(); v, ok := c.store[key]; return v, ok }`
4. `sync.Once` is simpler (no explicit boolean), guaranteed correct under the memory model, and cannot be misused by forgetting to check the flag. The mutex+bool pattern requires double-check logic to avoid contention on every read.
5. `wg.Wait()` blocks forever — the counter never reaches zero. This is why `defer wg.Done()` must be the first statement: `defer` runs even on panic.
6. When writes are frequent or the critical sections are very short — the overhead of the read/write lock mechanism exceeds the savings from concurrent reads.
7. The function is called again on the next `Do` invocation — the `Once` is not marked complete after a panic. The panic propagates to the caller of `Do`.
8. Change the value receiver to a pointer receiver: `func (c *T) Method()` instead of `func (c T) Method()`.

</details>

## Further reading

- [sync package docs](https://pkg.go.dev/sync) — official reference for all primitives
- [Go Memory Model](https://go.dev/ref/mem) — what synchronization guarantees each primitive provides
- [Effective Go: Sharing by Communicating](https://go.dev/doc/effective_go#sharing) — channels vs. mutexes philosophy
- [Go Blog: Share Memory By Communicating](https://go.dev/blog/codelab-share)
