# Go Pointers, Values, and Slice Semantics

## Prerequisites

- Basic Go syntax: struct declarations, method definitions, `make`, `append`.
- Familiarity with Java's distinction between primitives (`int`, `boolean`) and object references — the analogies throughout this lesson build on it.
- For how pointer receivers interact with interface satisfaction, see [go-interfaces-nil.md](go-interfaces-nil.md).

---

## Core concepts

1. **Value receiver** — A method that receives a copy of the struct; mutations inside the method are local and invisible to the caller.
2. **Pointer receiver** — A method that receives the address of the struct; mutations affect the original.
3. **Slice header** — The three-field struct `(ptr, len, cap)` that Go passes around to represent a slice; the backing array it points to is separate and shared.
4. **Element mutation vs header mutation** — Writing to `s[i]` reaches through the pointer to the shared backing array; reassigning `s` or changing `len` via `append` only affects the local header copy.
5. **append and reallocation** — `append` returns a new header; if the backing array had room, both headers point to the same array (sharing is live); if it had to grow, a fresh array is allocated (sharing ends).
6. **Subslice aliasing** — `original[low:high]` produces a new header pointing into the same backing array, so writes via the subslice mutate the original.
7. **Never copy a mutex** — A `sync.Mutex` encodes lock state in its fields; copying the struct copies that state, producing two independent mutexes with potentially corrupted state.

---

## Mental model

Think of a slice exactly as if it were Java's `ArrayList` internals exposed as a plain struct.

```
┌─────────────────────────────────────────────────────┐
│  ArrayList (Java)       ≈  Slice header (Go)        │
│                                                     │
│  Object[] elementData   →  ptr  (address of array)  │
│  int size               →  len  (elements visible)  │
│  elementData.length     →  cap  (array's capacity)  │
└─────────────────────────────────────────────────────┘
```

When Java passes an `ArrayList` to a method, it passes the reference to the ArrayList object — the header and the backing array all travel together. When Go passes a slice to a function, it passes **a copy of the three-field struct** — only the header is copied; the backing array is not. The function can mutate elements through the shared pointer, but any change to `len` or `cap` (including via `append`) only updates the local copy of the header.

The same framing works for receivers:

- **Value receiver** ≈ Java `int` passed by value: the method gets a copy; you can scribble on it all day and the caller's variable is unaffected.
- **Pointer receiver** ≈ Java object reference: the method gets an address; mutating a field changes the original object.

Go's innovation is making this explicit and mandatory. You declare the choice on every method. Getting it wrong compiles fine — the bug is silent.

---

## Worked examples

### Example 1 — Value vs pointer receivers (trivial)

```go
type Counter struct{ count int }

func (c Counter) IncrementBad() { c.count++ }  // value receiver: copy
func (c *Counter) Increment()   { c.count++ }  // pointer receiver: original
func (c Counter) Value() int    { return c.count }
```

```go
c := Counter{}
c.IncrementBad()
fmt.Println(c.Value())  // 0 — the copy was incremented, not c
c.Increment()
fmt.Println(c.Value())  // 1 — c.count was incremented in place
```

**Reasoning:** `IncrementBad` receives a fresh `Counter` on the stack. Incrementing `c.count` inside the method writes to that copy's field. When the function returns, the copy is discarded. The original `c` in the caller was never touched.

`Increment` receives `&c` — the address of the caller's `c`. `c.count++` dereferences that address and increments the field there. Java engineers: this is exactly what happens when you call a mutating method on an object reference.

**Java translation:**

```java
// Value receiver — like this (pointless copy, never done in Java)
void incrementBad(int count) { count++; }  // no effect on caller

// Pointer receiver — like every normal Java method call
void increment(Counter c) { c.count++; }   // mutates the object
```

---

### Example 2 — Slice header copy vs backing array sharing

```go
func modifyElement(s []int) {
    s[0] = 99  // writes through ptr into shared backing array
}

func appendToSlice(s []int) {
    s = append(s, 99)       // may or may not reallocate
    // either way: only s (local header) is updated
    // caller's header is unchanged
}
```

```go
nums := []int{1, 2, 3}

modifyElement(nums)
fmt.Println(nums)  // [99 2 3] — element visible through shared ptr

nums = []int{1, 2, 3}
appendToSlice(nums)
fmt.Println(nums)  // [1 2 3] — caller's len is still 3, header unchanged
```

**Reasoning step by step for `modifyElement`:**

```
Caller:   nums → header{ptr=0xA000, len=3, cap=3}
                         │
                         └──► backing array: [1, 2, 3]

modifyElement(s):
  s is a copy:  header{ptr=0xA000, len=3, cap=3}
                         │
                         └──► same backing array: [1, 2, 3]

  s[0] = 99 writes to 0xA000[0]
                         │
                         └──► backing array: [99, 2, 3]

Return: caller's ptr=0xA000 still points there → sees [99, 2, 3] ✓
```

**Reasoning for `appendToSlice`:**

```
nums has len=3, cap=3. append must allocate:
  new array: 0xB000 → [1, 2, 3, 99]
  s (local) becomes header{ptr=0xB000, len=4, cap=6}

Return: caller's nums still holds header{ptr=0xA000, len=3, cap=3}
        — it never saw 0xB000
```

The fix: return the new slice and reassign at the call site (the idiomatic Go pattern for append).

```go
func fill(s []int) []int {
    return append(s, 99)
}
nums = fill(nums)  // caller takes the new header
```

---

### Example 3 — append sharing and subslice aliasing (interview-level)

**Scenario A: append into spare capacity — sharing remains.**

```go
a := make([]int, 3, 6)  // [0,0,0], len=3, cap=6
a[0], a[1], a[2] = 1, 2, 3

b := append(a, 4)  // room in backing array; no reallocation
// a: ptr=0xA, len=3, cap=6
// b: ptr=0xA, len=4, cap=6  ← same backing array!

b[0] = 99
fmt.Println(a)  // [99 2 3]  ← a is affected even though it "didn't" see the append
fmt.Println(b)  // [99 2 3 4]
```

`a` still has `len=3`, so `a[3]` (the `4` written by append) is outside its visible window. But both headers point at the same array, so mutating `b[0]` shows up in `a[0]`. This is a latent aliasing bug: two slices that look independent can silently share memory.

**Scenario B: subslice.**

```go
original := []int{1, 2, 3, 4, 5}
sub := original[1:3]  // [2, 3], ptr points into original's array at offset 1
```

```
original: ptr→[1, 2, 3, 4, 5], len=5, cap=5
sub:      ptr→[_, 2, 3, 4, 5], len=2, cap=4
               ↑ same array, shifted by 1
```

```go
sub[0] = 99
fmt.Println(original)  // [1 99 3 4 5] — original[1] was overwritten

sub = append(sub, 888) // len(sub)=2 < cap(sub)=4: no reallocation
fmt.Println(original)  // [1 99 3 888 5] — original[3] was overwritten!
```

`append` wrote `888` into `original[3]` because `sub` still shared the backing array and had capacity. This is the subslice trap: **capacity, not length, governs whether a write escapes the subslice's visible window.**

To break the aliasing intentionally, use the three-index slice expression:

```go
sub := original[1:3:3]  // cap clamped to 3; next append forces reallocation
```

---

### Example 4 — Never copy a mutex

```go
type SafeCounter struct {
    mu    sync.Mutex
    count int
}

func (c *SafeCounter) Inc() {  // pointer receiver — correct
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

The bug surface is assignment or a value-returning constructor:

```go
type RateLimiter struct {
    mu      sync.Mutex
    counter int
    limit   int
}

func NewRateLimiter(limit int) RateLimiter {  // BUG: returns by value
    return RateLimiter{limit: limit}
}

rl := NewRateLimiter(10)  // rl holds a copy of the struct
                          // go vet: "assignment copies lock value"
rl.Allow()                // rl.mu might have been copied mid-lock
```

**Why is this undefined behavior?** `sync.Mutex` uses atomic operations on internal fields to implement a spin/sleep lock. When you copy the struct, you copy the current values of those fields. If the mutex was unlocked at the time of copy, the copy starts unlocked — but any goroutine that holds a pointer to the original can change its state independently of the copy. The two mutexes no longer protect the same region.

**Two fixes:**

1. Return `*RateLimiter` from the constructor (preferred — callers always have a pointer).
2. Make `NewRateLimiter` return `*RateLimiter` and never assign the struct by value elsewhere.

`go vet` catches this. It fires on: struct assignment, function parameters by value, and channel sends of structs containing mutexes.

---

## Common misconceptions

**"Calling a method mutates the struct — that's the normal behavior."**
Not in Go. Value receivers are the default in many tutorials and they do not mutate. You must explicitly use a pointer receiver. The compiler will not warn you if you declare a value receiver on a method that is supposed to mutate.

**"`append` always creates a new backing array."**
Wrong. If `len < cap`, `append` writes into the existing array and returns a new header with `len+1`. The old and new headers share memory. This is the source of the scenario-A aliasing bug above.

**"Passing a slice to a function is pass-by-reference."**
Partially right, accidentally. You are passing the header by value (a copy). Element mutations propagate because both headers point to the same array. But anything that changes `len` or `cap` — including `append` — only changes the local copy. The correct framing is: the pointer inside the header is shared, the header itself is not.

**"A subslice is an independent copy."**
Wrong. `s[1:3]` shares the backing array. Writes to the subslice's elements are visible through the original, and vice versa. If you need isolation, copy explicitly: `copy(dst, src)`.

**"Value receivers are fine as long as I don't need to mutate."**
True for plain data types. False for types containing `sync.Mutex`, `sync.WaitGroup`, or any other type that documents "do not copy." Even if you never mutate, copying those types is undefined behavior. Always use pointer receivers when the struct contains synchronization primitives.

**"I can mix value and pointer receivers on the same type freely."**
You can, but it creates confusion around method sets (which matters for interface satisfaction — see [go-interfaces-nil.md](go-interfaces-nil.md)). The Go convention: if any method on a type needs a pointer receiver, use pointer receivers for all methods on that type.

---

## Check-yourself questions

1. You define `func (c Counter) Reset() { c.count = 0 }`. After calling `c.Reset()`, what is `c.count`? Why?
2. A function receives `s []int` and executes `s[2] = 42`. Does the caller see the change? What if it executes `s = append(s, 42)` instead?
3. `a := make([]int, 4, 8)`. You call `b := append(a, 99)`. What are the `len` and `cap` of `a` and `b`? Do they share a backing array? What happens to `a` if you then write `b[0] = 777`?
4. `original := []int{10, 20, 30, 40, 50}; sub := original[2:4]`. What are the `len` and `cap` of `sub`? What happens to `original` if you execute `sub[0] = 99`? What happens if you then `append(sub, 888)`?
5. Why does copying a `sync.Mutex` produce undefined behavior rather than just duplicating the lock?
6. `NewRateLimiter` returns a `RateLimiter` value (not a pointer). Name two fixes and explain why each one prevents the mutex-copy bug.
7. A colleague says "I'll use value receivers everywhere since Go auto-promotes `v.Method()` to `(&v).Method()` — so I get mutation for free." Where is this reasoning wrong?
8. You have a function `func populate(s []string)` that does `s = append(s, "x", "y")`. A caller passes `data := make([]string, 0, 10)` and then prints `len(data)`. What does it print, and how would you fix `populate` to make the items visible to the caller?

<details>
<summary>Answers</summary>

**1.** `c.count` is unchanged (still whatever it was before). `Reset` receives a copy of the `Counter`. The copy's field is set to zero; the original is not touched. Fix: use `func (c *Counter) Reset() { c.count = 0 }`.

**2.** Yes, the caller sees `s[2] = 42` — both the caller's header and the function's header point to the same backing array; writing through the pointer changes the array in place. No, the caller does not see the `append` result. `append` returns a (potentially new) header with updated `len`. That new header is assigned to the function's local `s`, not to the caller's variable.

**3.** After `b := append(a, 99)`: `a` has `len=4, cap=8`; `b` has `len=5, cap=8`. They share the backing array (no reallocation occurred). Writing `b[0] = 777` changes the underlying array's first element. Because `a`'s pointer points to the same array, `a[0]` becomes `777`.

**4.** `sub` has `len=2` (elements at indices 2 and 3) and `cap=3` (from index 2 to the end of `original`'s backing array, which has length 5). Writing `sub[0] = 99` changes `original[2]` to 99 — same backing array. `append(sub, 888)` has room (`len=2 < cap=3`), so it writes `888` into `original[4]`, making `original` equal to `[10, 20, 99, 40, 888]`. No reallocation; the write escapes the subslice's visible window into `original`.

**5.** `sync.Mutex` stores internal state (a lock word, waiter count) in its fields. Goroutines communicate through those fields using atomic operations. Copying the struct produces a second set of fields with the copied state. If a goroutine later locks the original and a different goroutine operates on the copy, the two are independent mutexes — but they were supposed to guard the same data. The invariants the mutex provides no longer hold. Even if the mutex was unlocked at copy time, `go vet` flags it because there is no safe moment to copy a mutex that participates in concurrent code.

**6.** Fix 1: change `NewRateLimiter` to return `*RateLimiter`. Callers hold a pointer, so no struct copy occurs. Fix 2: ensure no code ever assigns a `RateLimiter` value to another variable after construction (enforce via code review / `go vet`). Fix 1 is strongly preferred — it makes the constraint structural and enforced by the type system.

**7.** Auto-promotion (`v.Method()` → `(&v).Method()`) only works when `v` is an addressable variable. It does not work when the value is a temporary (result of a function call, map lookup, etc.) or when assigning to an interface — in those cases the pointer receiver method is simply not in the method set of the value type. The colleague's code will get silent value-copy bugs in those contexts, and will fail to compile when assigning to interfaces that require pointer-receiver methods.

**8.** It prints `0`. `populate` receives a copy of the header. `append` updates the local copy's `len` to 2, but the caller's header still has `len=0`. Two fixes: (a) return the new slice — `func populate(s []string) []string { return append(s, "x", "y") }` — and assign at call site: `data = populate(data)`; (b) pass a pointer to the slice — `func populate(s *[]string) { *s = append(*s, "x", "y") }` — which is less idiomatic but valid.

</details>

---

## Further reading

- [Go specification — Slice types](https://go.dev/ref/spec#Slice_types) — authoritative definition of slice expressions, capacity, and the three-index form.
- [Go blog — Arrays, slices (and strings): The mechanics of 'append'](https://go.dev/blog/slices-intro) — Rob Pike's canonical walkthrough of the header model and append behavior.
- [Go specification — Method sets](https://go.dev/ref/spec#Method_sets) — defines which methods are in the method set of `T` vs `*T`; governs interface satisfaction.
- [pkg.go.dev/sync — Mutex](https://pkg.go.dev/sync#Mutex) — documentation states explicitly: "A Mutex must not be copied after first use."
- [go vet documentation](https://pkg.go.dev/cmd/vet) — lists all checks including `copylocks`, which catches mutex copies.
- [Go FAQ — Should I define methods on values or pointers?](https://go.dev/doc/faq#methods_on_values_or_pointers) — the Go team's guidance on choosing receiver types.
- [Russ Cox — Go Data Structures](https://research.swtch.com/godata) — low-level memory layout of slices and other Go types, with diagrams.
