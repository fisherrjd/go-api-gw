# Solutions: Goroutines, Channels, Select, Context

## Exercise 1 — Fan-out + collect

```go
func fanOut(ctx context.Context, ids []int) []string {
    results := make(chan string, len(ids)) // buffered so goroutines never block

    for _, id := range ids {
        id := id // shadow for Go < 1.22
        go func() {
            select {
            case <-ctx.Done():
                // don't send; channel is buffered so this doesn't leak
                return
            case <-time.After(10 * time.Millisecond):
                results <- fmt.Sprintf("user-%d", id)
            }
        }()
    }

    // If ctx can cancel, some goroutines may not send.
    // For simplicity here: collect exactly len(ids) results (assumes all succeed).
    // In production: use WaitGroup + separate done channel.
    out := make([]string, 0, len(ids))
    for i := 0; i < len(ids); i++ {
        out = append(out, <-results)
    }
    return out
}
```

**Key point:** buffer the results channel to `len(ids)` so goroutines can send without a matching receiver. If you use an unbuffered channel and don't drain it, goroutines leak.

---

## Exercise 2 — First-result wins

```go
func firstOf(ctx context.Context, fns []func(context.Context) (string, error)) string {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    results := make(chan string, len(fns)) // buffer so goroutines don't block

    for _, fn := range fns {
        fn := fn
        go func() {
            v, err := fn(ctx)
            if err == nil {
                results <- v
            }
        }()
    }

    select {
    case v := <-results:
        return v
    case <-ctx.Done():
        return ""
    }
}
```

**Key point:** cancel the context immediately when you get a winner — this signals the other goroutines to stop. Without cancel, the losers keep running (goroutine leak).

---

## Exercise 3 — Timeout with select

```go
func withTimeout(work func() int, d time.Duration) (int, bool) {
    ch := make(chan int, 1) // buffer 1 so goroutine doesn't leak on timeout
    go func() {
        ch <- work()
    }()

    select {
    case v := <-ch:
        return v, true
    case <-time.After(d):
        return 0, false
    }
}
```

**Key point:** the channel must be buffered. If `work()` finishes after the timeout, the goroutine tries to send on `ch`. With an unbuffered channel and no receiver, it leaks forever. With a buffer of 1, the send completes and the goroutine exits cleanly.

---

## Mental model: goroutine lifecycle rules

1. Every goroutine must have a way to **exit**. If it can block forever, you have a leak.
2. If a goroutine sends to a channel, **someone must receive** — or the channel must be buffered enough.
3. Closing a channel is the idiomatic signal for "no more data." Only the **sender** closes.
4. Use `ctx.Done()` as an escape hatch in any goroutine that might block.
