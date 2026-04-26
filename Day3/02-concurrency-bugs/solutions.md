# Solutions: Classic Concurrency Bugs

## Exercise 1 — Spot the bugs in go-bugs/exercise3_concurrency.go

Three bugs in `processBatch`:

**Bug A — wg.Done() never called (deadlock)**
```go
// BROKEN: wg.Done() missing from goroutine body
for j := range jobCh {
    r := enrich(ctx, j)
    results[r.UserID] = r
}
// FIX:
go func() {
    defer wg.Done() // must be inside the goroutine, deferred
    for j := range jobCh {
        ...
    }
}()
```

**Bug B — concurrent map write (data race)**
```go
// BROKEN: multiple goroutines write to results with no lock
results[r.UserID] = r

// FIX: protect with mutex
var mu sync.Mutex
mu.Lock()
results[r.UserID] = r
mu.Unlock()
```

**Bug C — unbuffered errCh in notify() (deadlock)**
```go
// BROKEN: errCh is unbuffered; goroutines sending errors block if
// no one is reading yet. The closer goroutine (wg.Wait+close) can't
// run because senders are blocked. Deadlock.
errCh := make(chan error)

// FIX: buffer to len(results) — worst case all results have errors
errCh := make(chan error, len(results))
```

---

## Exercise 2 — Shared counter with mutex

```go
func safeCounter(n int) int {
    var mu sync.Mutex
    var count int
    var wg sync.WaitGroup

    for i := 0; i < n; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            mu.Lock()
            count++
            mu.Unlock()
        }()
    }
    wg.Wait()
    return count
}
```

Run with `-race`:
```
go run -race main.go
# Should print no race warnings
```

---

## Exercise 3 — Pipeline with context cancellation

```go
func generate(ctx context.Context, nums []int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            select {
            case out <- n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for v := range in {
            select {
            case out <- v * v:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    nums := generate(ctx, []int{1, 2, 3, 4, 5})
    squares := square(ctx, nums)

    for v := range squares {
        fmt.Println(v)
    }
}
```

**Key:** each stage closes its output channel when done. `range in` exits when the input channel closes. `ctx.Done()` is the escape hatch if the pipeline is cancelled mid-way.

---

## Quick-fire: identify the bug type

| Pattern | Bug |
|---------|-----|
| `go func() { use(i) }()` in a loop | Loop variable capture |
| `wg.Add(1); go func() { ... wg.Done() }()` with possible panic before Done | Forgotten Done — use defer |
| `close(ch); ch <- x` | Send on closed channel |
| `ch <- x` (unbuffered) with no goroutine ready to receive | Deadlock |
| Goroutine blocking on channel with no exit | Goroutine leak |
| Two goroutines reading/writing `map[K]V` without lock | Data race |
