# Go Concurrency Bugs and the Race Detector

## Prerequisites

- [go-goroutines-channels-context.md](go-goroutines-channels-context.md) — goroutines, channels, select, context
- [go-sync-primitives.md](go-sync-primitives.md) — WaitGroup, Mutex, RWMutex, Once

## Core concepts

- **Loop variable capture** — pre-Go 1.22, all goroutines in a loop share the same loop variable
- **Forgotten wg.Done()** — a goroutine that exits without calling `Done()` deadlocks `wg.Wait()` forever
- **Send on closed channel** — panics immediately; only the sender should close
- **Unbuffered channel deadlock** — sender and receiver in the same goroutine block each other
- **Goroutine leak** — a goroutine that can never exit because it has no escape path
- **Data race on shared map** — Go maps are not safe for concurrent read+write
- **Race detector** — a runtime tool (`-race` flag) that reports unsynchronized concurrent memory access

## Mental model

Think of each bug as a specific "trap" interviewers plant in code. They're patterns, not accidents — once you've seen each shape, spotting it takes seconds. The race detector is your safety net in CI: it instruments every memory access at runtime and fires an alarm when two goroutines touch the same memory without synchronization, at least one being a write.

```
Interviewers plant one of these 6:

  1. loop var capture  →  goroutines all print the same value
  2. missing Done()   →  program hangs at wg.Wait()
  3. send on closed   →  panic: send on closed channel
  4. chan deadlock     →  fatal error: all goroutines are asleep
  5. goroutine leak   →  goroutine count grows unbounded
  6. map data race    →  concurrent map read and map write (or worse)
```

## Worked examples

### Bug 1: Loop variable capture (pre-Go 1.22)

```go
// BROKEN — all goroutines close over the same `i`; likely all print 3
for i := 0; i < 3; i++ {
    go func() { fmt.Println(i) }()
}

// FIX A: shadow the variable (creates a new `i` scoped to this iteration)
for i := 0; i < 3; i++ {
    i := i
    go func() { fmt.Println(i) }()
}

// FIX B: pass as argument
for i := 0; i < 3; i++ {
    go func(n int) { fmt.Println(n) }(i)
}
```

Go 1.22+ (released Feb 2024) makes each loop iteration its own variable, fixing this by default. Interviewers still test the old behavior — know both.

### Bug 2: Forgotten wg.Done()

```go
// BROKEN — if work() panics, Done never runs; wg.Wait() hangs forever
go func() {
    work()
    wg.Done() // never reached on panic
}()

// FIX — defer runs even on panic
go func() {
    defer wg.Done() // first line, always executes
    work()
}()
```

Rule: `defer wg.Done()` is always the first statement inside the goroutine body.

### Bug 3: Send on closed channel

```go
ch := make(chan int, 1)
ch <- 1
close(ch)

v, ok := <-ch  // ok=true, v=1  (value was buffered before close)
v, ok  = <-ch  // ok=false, v=0 (closed and empty — zero value, no panic)

// PANICS:
close(ch)  // panic: close of closed channel
ch <- 2    // panic: send on closed channel
```

**Rule: only the sender closes.** For multiple senders, use `sync.Once`:

```go
var once sync.Once
closeOnce := func() { once.Do(func() { close(ch) }) }
// any sender can call closeOnce() — only the first call does anything
```

### Bug 4: Deadlock on unbuffered channel

```go
// BROKEN — same goroutine is both sender and receiver; blocks forever
ch := make(chan int)
ch <- 1  // blocks waiting for a receiver that doesn't exist yet
<-ch

// FIX A: sender in a goroutine
ch := make(chan int)
go func() { ch <- 42 }()
fmt.Println(<-ch)

// FIX B: buffered channel (no synchronization needed)
ch := make(chan int, 1)
ch <- 42
fmt.Println(<-ch)
```

### Bug 5: Goroutine leak

```go
// BROKEN — goroutine blocks on work forever if no jobs arrive and channel never closes
go func() {
    for job := range work { process(job) } // leaks if work never closes
}()

// FIX — give every goroutine a ctx.Done() escape
go func() {
    for {
        select {
        case j, ok := <-work:
            if !ok { return }
            process(j)
        case <-ctx.Done():
            return // exits cleanly when context is cancelled
        }
    }
}()
```

Leak detection: `runtime.NumGoroutine()` in tests, or goleak library in CI.

### Bug 6: Data race on shared map

```go
// BROKEN — concurrent writes to a plain map = undefined behavior
m := map[int]int{}
for i := 0; i < 10; i++ {
    go func(n int) { m[n] = n }(i) // DATA RACE
}

// FIX — protect with mutex
var mu sync.Mutex
for i := 0; i < 10; i++ {
    go func(n int) {
        mu.Lock()
        m[n] = n
        mu.Unlock()
    }(i)
}
```

`sync.Map` is an alternative for specific patterns (mostly-reads or disjoint-key access), but a plain map + mutex is clearer for most cases.

### The race detector

```bash
go run -race main.go
go test -race ./...          # run in CI always
```

Sample output for a counter race:
```
WARNING: DATA RACE
Write at 0x00c000... by goroutine 7:
  main.raceOnCounter()
      /path/main.go:44

Previous read at 0x00c000... by goroutine 6:
  main.raceOnCounter()
      /path/main.go:44

Goroutine 7 (running) created at:
  main.raceOnCounter()
      /path/main.go:40
```

How to read it:
1. **Write at** + **Previous read/write** = the two conflicting accesses
2. Stack traces show **where** each goroutine accessed the memory
3. **Created at** shows **where** each goroutine was launched — that's where you fix the bug

Limitations:
- ~5-10x slowdown — only use in tests/dev, never in production
- Only catches races on **executed code paths** — 100% coverage needed for full confidence
- Does not catch logical races (wrong result without data race) or deadlocks (use `go vet` + timeout)

## Common misconceptions

**"If `-race` passes, there are no races."**  
The race detector is runtime — it only catches races on code paths that actually execute. A test suite with low coverage gives false confidence.

**"The race detector slows production."**  
Correct — never enable `-race` in production. Run it in tests and dev builds; that's the intended workflow.

**"Closing a channel is sufficient synchronization."**  
A channel close synchronizes goroutines that receive from it, but it doesn't protect other shared state. You still need a mutex for shared maps, counters, and structs.

**"Go 1.22 fixed loop capture everywhere."**  
Only for `for` loops with `range` or three-clause form. Closure captures of variables declared *outside* loops are unaffected.

**"Only the panic goroutine is affected by send-on-closed."**  
The entire program crashes on a panic unless recovered. A panicking goroutine that is not recovered takes down the process.

## Check-yourself questions

1. In Go < 1.22, what value do all goroutines in `for i := 0; i < 3; i++ { go func() { fmt.Println(i) }() }` likely print?
2. Why does `defer wg.Done()` need to be the **first** line inside a goroutine?
3. What is the zero-value behavior of receiving from a closed, empty channel?
4. A goroutine sends to an unbuffered channel but no receiver exists. What happens? What is the fix?
5. What are two ways to give a goroutine a clean exit path to prevent a leak?
6. You see `WARNING: DATA RACE / Write at ... by goroutine 7 / Previous read at ... by goroutine 6`. What should you look at first to fix it?
7. Why is `go test -race ./...` not a complete guarantee of no races?
8. What does `sync.Once` do if the function passed to `Do` panics?

<details>
<summary>Answers</summary>

1. All likely print `3` — they share the same `i`, which equals 3 after the loop finishes. Output is non-deterministic but usually all the same final value.
2. Because if the goroutine panics or returns early via any path, the deferred call still runs. Putting `Done()` after other statements means it can be skipped on early exit.
3. You get the zero value of the channel's element type, and the ok flag is `false`. It does not panic.
4. The goroutine (or main) blocks forever — deadlock. Fix: move the send into a goroutine, or use a buffered channel.
5. (a) Pass a `context.Context` and `select` on `ctx.Done()`; (b) pass a quit/done channel and `select` on it; (c) close the input channel so a `range` loop exits.
6. Look at the "Created at" section to find where each goroutine was launched — that's the callsite where you need to add synchronization (mutex, atomic, or channel).
7. It only detects races on code paths that execute during the test run. Untested paths can still contain races.
8. The panic propagates to the caller of `Do`. Subsequent calls to `Do` will call the function again (the Once is not marked complete on panic).

</details>

## Further reading

- [Go Memory Model](https://go.dev/ref/mem) — the spec that defines what is and isn't a race
- [Race Detector docs](https://go.dev/doc/articles/race_detector) — how to use it, what it catches
- [goleak](https://github.com/uber-go/goleak) — Uber's goroutine leak detector for tests
- [Go Blog: Introducing the Go Race Detector](https://go.dev/blog/race-detector)
