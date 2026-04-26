# Exercise Solutions: Interfaces and Nil

## Exercise 1 — The typed nil

**Output:** `err == nil: false` — then we enter the error branch even though `fail=false`.

**Why:** `getError` declares `var err *DBError` (nil pointer), then returns it as `error`. The interface value it returns is `(type=*DBError, value=nil)`. An interface is nil only when **both** type and value are nil. Here, type is `*DBError`, so the interface is not nil.

The value prints as `<nil>` (because the pointer is nil), but the interface itself is non-nil. This is the "lying nil" — `err.Error()` would call `(*DBError).Error()` with a nil receiver.

**Fix:** return untyped nil directly:
```go
func getError(fail bool) error {
    if fail {
        return &DBError{msg: "db connection failed"}
    }
    return nil  // untyped nil — interface is (type=nil, value=nil) — actually nil
}
```

**The rule:** never return a typed nil as an interface. If the return type is `error` (or any interface), return the literal `nil`, not a nil pointer of a concrete type.

---

## Exercise 2 — Nil pointer dereference via method

`svc.log` is nil (zero value for `*Logger`). Calling `svc.log.Log("starting")` invokes `(*Logger).Log` with a nil receiver. The method then does `l.prefix` — dereferencing a nil pointer — which panics.

**Fix option A:** Check nil in the method:
```go
func (l *Logger) Log(msg string) {
    if l == nil {
        return
    }
    fmt.Printf("[%s] %s\n", l.prefix, msg)
}
```

**Fix option B:** Initialize the logger (more idiomatic):
```go
svc := &Service{log: &Logger{prefix: "svc"}}
```

**Fix option C:** Guard at the call site:
```go
if svc.log != nil {
    svc.log.Log("starting")
}
```

Option A is the "safe nil receiver" pattern — useful for optional dependencies like loggers.

---

## Exercise 3 — Interface satisfaction

`*Handler` does NOT satisfy `Doer` because the `Do` method (when uncommented) has a **value receiver** `(h Handler)`, not a pointer receiver. When you write `var d Doer = &Handler{...}`, Go checks if `*Handler` has a `Do` method. It does — Go automatically promotes value receiver methods to pointer types. So `*Handler` DOES satisfy `Doer`.

**However**, if you write `var d Doer = Handler{...}` (not a pointer), it also works, since `Handler` has the value receiver method directly.

**The subtlety:** if `Do` had a pointer receiver `(h *Handler)`, then `Handler` (non-pointer) would NOT satisfy `Doer` — only `*Handler` would.

Rule of thumb: **pointer receiver methods are NOT in the method set of the value type, but value receiver methods ARE in the method set of the pointer type.**

---

## Exercise 4 — Typed nil in real code

`runQuery` returns `(*QueryError)(nil)` on success. The caller assigns this to `var asErr error`. This creates the same typed nil situation: `asErr` is `(type=*QueryError, value=nil)` — not nil.

**Why this pattern appears in production:** the function returns both `*QueryError` (a domain-level error) and `error` (a system-level error). The caller wants to check both. But assigning a nil `*QueryError` to an `error` interface silently creates a non-nil interface.

**Fix:** check the concrete type directly, don't assign to interface:
```go
qErr, err := runQuery("SELECT 1", false)
if err != nil { ... }
if qErr != nil {  // check the concrete pointer, not the interface
    fmt.Println("query error:", qErr)
}
```

Or, change `runQuery` to return `error` instead of `*QueryError` (preferred when callers use `errors.As`):
```go
func runQuery(query string, fail bool) error {
    if fail {
        return &QueryError{Query: query, Err: errors.New("timeout")}
    }
    return nil  // untyped nil — safe
}
```
