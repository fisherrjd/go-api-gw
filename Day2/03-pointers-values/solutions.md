# Exercise Solutions: Pointers, Values, Slices

## Exercise 1 — Value receiver mutation

**Output:**
```
inside AddItem, items: [apple]
inside AddItem, items: [banana]
cart items: []
cart len: 0
```

`AddItem` has a value receiver `(c Cart)`. It receives a copy of `Cart`. The `append` modifies the copy's `items` field, but the original `cart` variable is unchanged.

**Fix:** use a pointer receiver:
```go
func (c *Cart) AddItem(item string) {
    c.items = append(c.items, item)
}
```

**Why `append` doesn't help here even with a pointer:** the `items` field is a slice header. `append` may create a new backing array. Using a pointer receiver means `c.items = append(...)` updates the actual struct's slice header, which is what we want.

---

## Exercise 2 — append doesn't propagate

`fill` receives a copy of the slice header `(ptr, len=0, cap=10)`. Inside `fill`, `s = append(...)` updates the LOCAL `s` header to `len=3`. The caller's `data` header still has `len=0`.

**Fix option A: return the slice** (idiomatic Go)
```go
func fill(s []string) []string {
    return append(s, "a", "b", "c")
}
// caller:
data = fill(data)
```

**Fix option B: pass a pointer to the slice** (less idiomatic, but valid)
```go
func fill(s *[]string) {
    *s = append(*s, "a", "b", "c")
}
// caller:
fill(&data)
```

Option A is the Go standard. You see it everywhere: `data = append(data, x)`.

---

## Exercise 3 — Subslice sharing (predict output)

```
window:   [99 30 40]
original: [10 99 30 40 50]    ← window[0]=99 wrote into backing array

after append(window, 888):
window:   [99 30 40 888]
original: [10 99 30 40 888]   ← append wrote into original's backing array (cap had room)
```

`window := original[1:4]` creates a slice header pointing INTO `original`'s backing array. `window` has len=3, cap=4 (from index 1 to end of original's capacity).

`window[0] = 99` writes to index 1 of the backing array → `original[1]` changes.

`append(window, 888)` has cap=4 and len=3 — room for one more element. It writes `888` into the backing array at index 4 (0-indexed), which is `original[4]`. So `original` becomes `[10 99 30 40 888]`.

**The safe pattern:** use `original[1:4:4]` (three-index slice) to cap `window` at len=3, so any append allocates a new backing array and can't write into `original`.

---

## Exercise 4 — Mutex copied by value

`NewRateLimiter` returns `RateLimiter` by value. The returned value contains a `sync.Mutex`. Any assignment after that copies the mutex:
```go
rl := NewRateLimiter(10)  // rl is a copy — mutex is copied
```

`go vet` reports: `assignment copies lock value to rl: sync.Mutex`.

The mutex's internal state may include a lock bit. Copying it clones that state, so the copy and the original are independent but potentially desynchronized. Using the copy is a data race waiting to happen.

**Fix option A: return a pointer**
```go
func NewRateLimiter(limit int) *RateLimiter {
    return &RateLimiter{limit: limit}
}
// rl := NewRateLimiter(10)  // *RateLimiter — no copy
```

**Fix option B: embed the mutex as a pointer** (less common, more explicit)
```go
type RateLimiter struct {
    mu      *sync.Mutex  // pointer — copies don't duplicate the mutex
    ...
}
```

Option A is the standard Go pattern. Any struct containing a `sync.Mutex` should almost always be used via pointer.

**Interview signal:** if you see a struct with a mutex returned by value, or a value receiver on a method that locks, flag it immediately. `go vet` catches this but interviewers sometimes disable it.
