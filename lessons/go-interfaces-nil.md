# Go Interfaces and the Typed Nil

## Prerequisites

- Familiarity with interfaces in Java or Python (duck-typing concepts).
- Basic Go syntax: struct declarations, pointer types (`*T`), and function signatures.
- Understanding of what a nil/null pointer means in any language.

---

## Core concepts

1. **Implicit interface satisfaction** — A type satisfies an interface by having the right methods; no declaration needed.
2. **Interface internals: the (type, value) pair** — Every interface variable holds two fields: a concrete type descriptor and a pointer to the concrete value.
3. **The typed nil trap** — A nil pointer wrapped in an interface is NOT a nil interface, because the type field is still populated.
4. **Nil receivers** — Go allows calling a method on a nil pointer; it only panics if the method body dereferences the receiver without checking.
5. **Interface comparison semantics** — `==` on two interface values compares both the type and value; use `errors.Is` to traverse error chains.
6. **"Accept interfaces, return structs" convention** — Functions that accept behavior should use interfaces; functions that produce values should return concrete types.
7. **Pointer vs value receivers and method sets** — Which receiver you declare determines which method set is satisfied, affecting interface compliance.

---

## Mental model

Think of an interface variable as a **two-slot envelope**.

```
┌─────────────────────────────┐
│  interface value            │
│  ┌───────────┬───────────┐  │
│  │   TYPE    │   VALUE   │  │
│  │ *AppError │   0xABC   │  │
│  └───────────┴───────────┘  │
└─────────────────────────────┘
```

The envelope is nil **only when both slots are empty**. If you slide a typed nil pointer into the VALUE slot, the TYPE slot is still stamped — the envelope is not empty. This is the entire source of the typed nil trap. Every confusing behavior in this lesson flows from this two-slot picture.

In Java, every reference is its own envelope, and `null` means the reference itself is empty. Go separates the "what kind of thing" (type) from the "where is the thing" (value). When you assign a `*AppError` to an `error` interface, Go fills the TYPE slot with `*AppError` immediately, regardless of whether VALUE is nil.

---

## Worked examples

### Example 1 — Implicit satisfaction (trivial)

**Goal:** see that no `implements` keyword exists.

```go
type Storer interface {
    Save(key, value string) error
    Load(key string) (string, error)
}

type MemoryStore struct{ data map[string]string }

func (m *MemoryStore) Save(key, value string) error { /* ... */ }
func (m *MemoryStore) Load(key string) (string, error) { /* ... */ }

func process(s Storer) { /* uses s.Save / s.Load */ }

store := NewMemoryStore()  // type is *MemoryStore
process(store)             // compiles — no cast, no declaration needed
```

**Reasoning:** The compiler checks: "does `*MemoryStore` have a `Save` and a `Load` with exactly those signatures?" Yes. Assignment succeeds. This is structural typing, identical in spirit to Python duck-typing, but checked at compile time.

Java equivalent would require `class MemoryStore implements Storer`. Go requires nothing — the methods themselves are the declaration.

**Compile-time check trick:** If you want the compiler to assert satisfaction explicitly (useful in library code), add one line anywhere:

```go
var _ Storer = (*MemoryStore)(nil)  // fails at compile time if interface not satisfied
```

---

### Example 2 — The typed nil trap (the interview bug)

**Goal:** understand why `err == nil` returns `false` when you expect `true`.

```go
type AppError struct{ Code int; Message string }
func (e *AppError) Error() string { return fmt.Sprintf("%d: %s", e.Code, e.Message) }

// BUGGY
func riskyOpBuggy(fail bool) error {
    var err *AppError          // zero value: nil pointer of type *AppError

    if fail {
        err = &AppError{Code: 500, Message: "broke"}
    }

    return err  // <-- always returns the interface; TYPE slot = *AppError
}

err := riskyOpBuggy(false)
fmt.Println(err == nil)  // false — envelope has TYPE=*AppError, VALUE=nil
```

Step through the two-slot envelope:

```
return err
  → Go wraps *AppError in the error interface
  → envelope: TYPE=*AppError, VALUE=(nil pointer)
  → envelope is NOT empty
  → err == nil checks: is envelope empty? NO.
```

**The fix — always return untyped nil for "no error":**

```go
// CORRECT
func riskyOpCorrect(fail bool) error {
    if fail {
        return &AppError{Code: 500, Message: "broke"}
    }
    return nil  // untyped nil → TYPE=nil, VALUE=nil → truly nil interface
}
```

The rule: **never declare a typed variable to hold your error and then return it.** Return `nil` directly on the happy path, and return the concrete value directly on the error path.

---

### Example 3 — Nil receivers and method calls (realistic)

**Goal:** know when calling a method on nil is safe vs when it panics.

```go
func (e *AppError) IsRetryable() bool {
    if e == nil {        // explicit nil guard
        return false
    }
    return e.Code >= 500
}

var e *AppError          // nil pointer
e.IsRetryable()          // fine — nil is passed as receiver, guard catches it
```

```go
type Logger struct{ prefix string }

func (l *Logger) Log(msg string) {
    fmt.Printf("[%s] %s\n", l.prefix, msg)  // dereferences l.prefix
}

type Service struct{ log *Logger }

svc := &Service{}        // log is nil (zero value for pointer)
svc.log.Log("start")     // PANIC: l is nil, l.prefix dereferences nil
```

**Sequence of what happens in the panic case:**

```
svc.log.Log("start")
  │
  ├─ svc.log  → nil (*Logger)
  ├─ .Log("start")  → call (*Logger).Log with receiver = nil
  └─ inside Log: l.prefix  → dereference nil  → runtime panic
```

The compiler sees `svc.log` as `*Logger` (a non-nil type), so it allows the call. The panic is runtime-only. The fix is either a nil guard inside `Log`, or checking `svc.log != nil` at the call site before dispatching.

---

### Example 4 — Interface comparison and errors.Is

```go
var ErrSentinel = errors.New("sentinel")

// Direct comparison works when you hold the same pointer:
var a error = ErrSentinel
var b error = ErrSentinel
fmt.Println(a == b)  // true — TYPE=*errorString, VALUE=same pointer

// Wrapping breaks == but not errors.Is:
wrapped := &wrappedErr{inner: ErrSentinel}
fmt.Println(error(wrapped) == ErrSentinel)           // false — different TYPE
fmt.Println(errors.Is(wrapped, ErrSentinel))          // true  — unwraps chain
```

`errors.Is` walks the chain by calling `Unwrap()` repeatedly and comparing each level with `==`. Use `==` only when you own both sides and know neither is wrapped. Use `errors.Is` in all other cases.

---

## Pointer vs value receivers and method sets

This determines whether a type satisfies an interface.

| Receiver declared on | Method callable on | Included in method set of |
|---|---|---|
| `func (t T) M()`   | `T` and `*T`       | `T` and `*T`              |
| `func (t *T) M()`  | `*T` only (Go auto-derefs for variables, not interfaces) | `*T` only |

Practical consequence:

```go
type Doer interface { Do() }

type Handler struct{}
func (h Handler) Do() {}    // value receiver

var d Doer = Handler{}      // OK
var d2 Doer = &Handler{}    // also OK — *T includes T's value-receiver methods
```

```go
type Handler2 struct{}
func (h *Handler2) Do() {}  // pointer receiver

var d Doer = &Handler2{}    // OK
var d3 Doer = Handler2{}    // COMPILE ERROR — Handler2 (value) does NOT include pointer-receiver methods
```

**Why?** If Go allowed `Handler2{}` (an addressless temporary) to satisfy a pointer-receiver interface, the runtime would have nowhere to write back mutations. So Go disallows it at compile time.

**Java analogy:** There is no direct equivalent. In Java, `this` is always a reference; Go distinguishes value copies from pointer aliases at the method-set level.

---

## Common misconceptions

**1. "If the value is nil, the interface is nil."**
Wrong. A `(*AppError)(nil)` stored in an `error` variable is not nil. The type slot is occupied. Test with a real `nil` return, not a typed nil variable.

**2. "You can always call a method on nil; it's safe."**
Wrong. You can always *dispatch* to the method — Go will not stop you at the call site. But the method body panics the moment it dereferences the nil receiver. Safety depends entirely on what the method does.

**3. "Accept interfaces, return interfaces — that's more flexible."**
Returning interfaces hides the concrete type from callers. They cannot access extra methods, cannot type-assert cheaply, and get the typed-nil trap for free if they return nil carelessly. Return concrete types (usually a pointer to a struct); let callers decide what interface to assign it to.

**4. "Value receiver and pointer receiver are interchangeable."**
They affect the method set. If an interface requires a method with a pointer receiver, only `*T` satisfies it — not `T`. Mixing them without understanding the method set rule is a common compile error in Go for newcomers.

**5. "`errors.Is` and `==` are equivalent for errors."**
Only true for non-wrapped errors stored in the same variable. Any wrapping with `fmt.Errorf("%w", err)` breaks `==` but not `errors.Is`.

**6. "The typed nil panic only happens when you return errors."**
The trap applies to any interface type, not just `error`. If you have an interface `Storer` and return a nil `*MemoryStore` as a `Storer`, the same non-nil interface results.

---

## Check-yourself questions

1. What two fields does every Go interface value hold internally?
2. You write `var err *MyError; return err` from a function returning `error`. Is `err == nil` true at the call site? Why or why not?
3. What is the correct way to return "no error" from a function with signature `func foo() error`?
4. A method `func (l *Logger) Log(msg string)` is called on a nil `*Logger`. When does it panic, and when is it safe?
5. If `*MemoryStore` has both `Save` and `Load` declared with pointer receivers, does a plain `MemoryStore` (value, not pointer) satisfy `Storer`? Why?
6. You have `var a error = fmt.Errorf("wrap: %w", ErrSentinel)`. Does `a == ErrSentinel` return true? What should you use instead?
7. Why does "accept interfaces, return structs" reduce the chance of the typed nil trap?
8. You see `var _ Storer = (*MemoryStore)(nil)` at package level in a library. What does this line do and why is it useful?

<details>
<summary>Answers</summary>

**1.** A concrete type descriptor (often called the `itab` or type pointer) and a pointer to the concrete value. An interface is nil only when both are nil/zero.

**2.** No, `err == nil` is false. The function wraps `(*MyError)(nil)` into the `error` interface. The type slot is set to `*MyError`, so the envelope is not empty. The caller sees a non-nil interface holding a nil pointer.

**3.** `return nil` — untyped nil. This sets both the type and value slots of the returned interface to nil, producing a truly nil interface.

**4.** The call itself succeeds (Go dispatches to the method with `nil` as the receiver). It panics only if the method body dereferences the receiver (e.g., reads a field). If the method begins with `if l == nil { return }` or never accesses fields, it is safe.

**5.** No. The methods are declared on `*MemoryStore` (pointer receiver). The method set of a plain `MemoryStore` value does not include pointer-receiver methods. Only `*MemoryStore` satisfies `Storer`. This produces a compile error if you try `var s Storer = MemoryStore{}`.

**6.** No. `fmt.Errorf("%w", ...)` creates a new error value that wraps `ErrSentinel`; it is a different pointer with a different type. `a == ErrSentinel` is false. Use `errors.Is(a, ErrSentinel)` which unwraps the chain.

**7.** When you return a concrete struct pointer, the caller receives a `*MyStruct`, not an interface. There is no interface envelope to be non-nil while holding a nil pointer. The typed nil trap only manifests when the return type is an interface. Keeping return types concrete eliminates the opportunity for the bug.

**8.** It is a compile-time assertion. The blank identifier assignment forces the compiler to check that `*MemoryStore` implements `Storer`. If it does not (e.g., a method was renamed or accidentally removed), the build fails immediately rather than at the point of use. It documents intent without runtime cost.

</details>

---

## Further reading

- [Go specification — Interface types](https://go.dev/ref/spec#Interface_types) — authoritative definition of interface satisfaction and method sets.
- [Go FAQ — Why is my nil error value not equal to nil?](https://go.dev/doc/faq#nil_error) — the Go team's canonical explanation of the typed nil trap.
- [Go blog — Errors are values](https://go.dev/blog/errors-are-values) — Rob Pike on idiomatic error handling patterns.
- [Go blog — Error wrapping](https://go.dev/blog/go1.13-errors) — covers `errors.Is`, `errors.As`, and `%w`.
- [Russ Cox — Go Data Structures: Interfaces](https://research.swtch.com/interfaces) — low-level walkthrough of the `itab` implementation; explains why the type slot exists separately from the value slot.
- [Effective Go — Interfaces and other types](https://go.dev/doc/effective_go#interfaces) — canonical style guidance including the "accept interfaces, return structs" rationale.
