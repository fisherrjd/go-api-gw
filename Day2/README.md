# Day 2: Errors, Interfaces, Nil, Pointers

3 hours. This is where Java and Python intuition actively misleads you.

## Schedule

| Block | Dir | Focus |
|-------|-----|-------|
| 60 min | `01-errors/` | Error idioms, wrapping, Is/As |
| 60 min | `02-interfaces-nil/` | Typed nil — the #1 interview bug |
| 60 min | `03-pointers-values/` | Mutation, slices, append surprises |

## How to work each section

```bash
cd 01-errors
go run main.go        # run first — read the output, then read the code
# work through the EXERCISES section at the bottom of main.go
# check solutions.md when you're stuck or done
```

Or paste any snippet into https://go.dev/play

## Exit criteria

Say these out loud before moving on:
- Why `errors.Is` instead of `==` for sentinel errors?
- What is the typed nil trap? Give a concrete example.
- Why doesn't a value receiver mutation affect the caller?
- What does `append` do when the slice is at capacity vs not?
