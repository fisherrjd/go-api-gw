# Go Goroutines, Channels, Select, and Context

## Prerequisites

- Basic Go syntax (functions, types, closures)
- Understanding of concurrency concepts (threads, locks) from Python/Java
- See [go-pointers-values.md](go-pointers-values.md) for pointer mechanics referenced here

## Core concepts

- **Goroutine** — a lightweight green thread managed by the Go runtime; started with `go f()`, returns no handle
- **Channel** — a typed, synchronized pipe between goroutines; the primary way to communicate and coordinate
- **Unbuffered channel** — sender blocks until a receiver is ready (synchronous rendezvous)
- **Buffered channel** — sender blocks only when the buffer is full (async up to capacity)
- **select** — waits on multiple channel operations simultaneously; picks randomly when multiple are ready
- **context.Context** — carries a cancellation signal (and optional deadline/values) across goroutine boundaries

## Mental model

Think of goroutines as workers at mail stations. A channel is a conveyor belt between stations. An unbuffered belt stops both sides until a handoff occurs — guaranteed synchronization. A buffered belt lets the sender keep working until the belt fills up. `select` is a worker watching multiple belts, grabbing from whichever moves first. `context.Context` is the factory PA system — when management (the parent) cancels a job, every worker listening on `ctx.Done()` hears the signal and stops.

## Worked examples

### Example 1: Start a goroutine and wait for it

```go
done := make(chan struct{})
go func() {
    fmt.Println("hello from goroutine")
    close(done) // signal completion by closing
}()
<-done // block until goroutine closes the channel
```

`struct{}` is the idiomatic signal type — zero bytes, purely used as a signal.

### Example 2: Producer/consumer with channels

```go
func producer(out chan<- int, n int) {
    for i := 0; i < n; i++ {
        out <- i
    }
    close(out) // sender always closes; receiver range exits on close
}

func main() {
    ch := make(chan int) // unbuffered: each send blocks until main receives
    go producer(ch, 5)
    for v := range ch { // range exits when channel is closed
        fmt.Println(v)
    }

    // Buffered: producer can send 3 items without a receiver waiting
    buf := make(chan int, 3)
    buf <- 10; buf <- 20; buf <- 30
    fmt.Println(<-buf, <-buf, <-buf)
}
```

Directional types enforce intent at compile time:
- `chan<- T` — send-only (producer parameter)
- `<-chan T` — receive-only (consumer parameter)

### Example 3: Timeout with select and context cancellation

```go
// Timeout pattern: race a result against a timer
func withTimeout(work func() int, d time.Duration) (int, bool) {
    result := make(chan int, 1)
    go func() { result <- work() }()
    select {
    case v := <-result:
        return v, true
    case <-time.After(d):
        return 0, false
    }
}

// Context cancellation: fan-out, cancel when first result arrives
func firstOf(ctx context.Context, fns []func(context.Context) (string, error)) string {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel() // cancel remaining workers when we return

    results := make(chan string, len(fns))
    for _, fn := range fns {
        fn := fn
        go func() {
            if v, err := fn(ctx); err == nil {
                results <- v
            }
        }()
    }
    return <-results
}
```

### Example 4: context in HTTP handlers

```go
// CORRECT: use the request's context — it cancels when the client disconnects
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // NOT context.Background()
    result, err := db.QueryContext(ctx, "SELECT ...")
    // if client disconnects mid-query, ctx cancels and QueryContext returns
}

// BROKEN: context.Background() never cancels — goroutine leaks on disconnect
func badHandler(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background() // ignores client cancellation
    db.QueryContext(ctx, "SELECT ...")
}
```

**Always `defer cancel()`** even when using `WithTimeout` — the context may expire naturally, but `cancel()` releases resources immediately instead of waiting for the GC.

## Common misconceptions

**"I can cancel a goroutine directly."**  
Go has no goroutine handle or kill API. You cancel via a channel or `context.Context`. The goroutine must cooperate by checking `ctx.Done()` or a quit channel.

**"Closing a channel stops goroutines listening on it."**  
Closing causes receives to return immediately (zero value, `ok=false`), but the goroutine still runs — it must check `ok` and return explicitly.

**"select is like switch — it picks the first ready case."**  
When multiple cases are ready simultaneously, Go picks one **at random**. Never rely on case order for correctness.

**"context.Background() is fine inside request handlers."**  
Using `context.Background()` in a handler breaks cancellation propagation. Use `r.Context()` so I/O respects client disconnects and timeouts.

**"A buffered channel is always faster."**  
Buffered decouples sender from receiver but doesn't guarantee delivery order under contention, and a full buffer still blocks the sender. Use unbuffered when you need a rendezvous/synchronization guarantee.

## Check-yourself questions

1. What happens if you call `go f()` and the main goroutine exits before `f` finishes?
2. What is the difference between sending on an unbuffered vs. a buffered channel?
3. How do you signal to a goroutine that it should stop? Give two approaches.
4. What happens when you receive from a closed channel? What do you get?
5. In a `select` with three ready cases, which one runs?
6. What does `defer cancel()` protect against when using `context.WithTimeout`?
7. Why must you use `r.Context()` in an HTTP handler instead of `context.Background()`?
8. Write the signature for a function that accepts a send-only channel of strings.

<details>
<summary>Answers</summary>

1. The whole program exits — all goroutines are killed. Use `sync.WaitGroup` or a done channel to wait.
2. Unbuffered: sender blocks until a receiver is ready (rendezvous). Buffered: sender only blocks when the buffer is full.
3. (a) Pass a `context.Context` and check `ctx.Done()`; (b) pass a quit channel and `select` on it.
4. You get the zero value of the channel's type and `ok=false`. It does not panic (unlike sending to a closed channel).
5. Go picks one **at random** — the spec guarantees non-determinism when multiple cases are ready.
6. It frees internal timer resources immediately. Without it, the resources linger until the deadline fires.
7. `r.Context()` is cancelled when the HTTP client disconnects or the server times out the request, propagating cancellation to any downstream I/O. `context.Background()` never cancels, so goroutines and DB queries keep running after the client is gone.
8. `func send(ch chan<- string)`

</details>

## Further reading

- [Go Tour: Concurrency](https://go.dev/tour/concurrency/1) — interactive intro
- [Effective Go: Goroutines](https://go.dev/doc/effective_go#goroutines)
- [context package docs](https://pkg.go.dev/context) — official usage guidance
- [Go Blog: Pipelines and Cancellation](https://go.dev/blog/pipelines) — canonical fan-out/fan-in patterns
- [Go Blog: Context](https://go.dev/blog/context) — motivation and idioms
