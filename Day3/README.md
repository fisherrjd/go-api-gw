# Day 3: Concurrency Primitives

5–6 hours. Weekend day — build and break, not just read.

## Schedule

| Block | Dir | Time | Focus |
|-------|-----|------|-------|
| Core 1 | `01-goroutines-channels/` | 60 min | goroutines, channels, select, context |
| Core 2 | `02-concurrency-bugs/` | 45 min | the 6 classic bugs — spot and fix |
| Core 3 | `03-sync-primitives/` | 45 min | Mutex, RWMutex, Once, WaitGroup |
| Core 4 | `04-race-detector/` | 30 min | run -race, read the output |
| Bonus  | `bonus-worker-pool/` | 60–90 min | build a real concurrent program |

## How to work each section

```bash
cd 01-goroutines-channels
go run main.go        # read output, then read code
# work the EXERCISES at the bottom
# check solutions.md when stuck or done
```

Race detector:
```bash
go run -race main.go
```

## Exit criteria

Say these out loud before moving on:
- What's the difference between a buffered and unbuffered channel?
- How does `select` differ from a regular `switch`?
- What is a goroutine leak? How do you detect one?
- What does `wg.Done()` inside a goroutine vs outside do differently?
- Why is copying a `sync.Mutex` dangerous?
- What does the race detector output tell you?
