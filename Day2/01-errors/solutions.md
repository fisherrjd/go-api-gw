# Exercise Solutions: Errors

## Exercise 1 — Ignored error

`name` is `""` — the zero value for string. Go doesn't panic or throw; the error is silently lost and execution continues with a meaningless value.

**Fix:** check the error every time.
```go
name, err := fetchUser(0)
if err != nil {
    fmt.Println("error:", err)
    return
}
fmt.Printf("name: %q\n", name)
```

**Interview pattern:** "ignoring errors" is the #1 Go bug. In production code it usually looks like `if _, err := db.Exec(...); true { ... }` — subtler but same root cause.

---

## Exercise 2 — Shadowed err

The `:=` in `if result, err := step()` declares **new** variables `result` and `err` scoped to the if block. The outer `var err error` is never touched.

When `step()` returns an error, `err == nil` is false, so we go to the else branch and print the inner err. But the outer `err` stays nil. The function returns nil even though `step()` failed.

**Fix:** declare `result` before the if, then use `=` (not `:=`) to assign into existing variables:
```go
var err error
result, err := step()   // := is fine here — result is new
if err != nil {
    return fmt.Errorf("exercise2: %w", err)
}
fmt.Println("result:", result)
return nil
```

Or, if you need the if-init pattern, use `=` after declaring both variables:
```go
var result string
var err error
result, err = step()    // = assigns to outer vars
if err != nil {
    return err
}
```

**Why it's subtle:** Go requires at least one new variable on the left of `:=`, so `if result, err :=` looks valid — and is, syntactically. The compiler won't warn you. The outer `err` just silently stays nil.

---

## Exercise 3 — == instead of errors.Is

`handleGetUser` wraps `ErrNotFound` twice:
```
handleGetUser: loadUser(0): not found
```
The resulting `err` is a `*fmt.wrapError` value. It is **not** the same pointer as `ErrNotFound`, so `==` returns false.

`errors.Is` walks the chain via `Unwrap()` until it finds a match.

**Fix:**
```go
if errors.Is(err, ErrNotFound) {
    fmt.Println("not found")
}
```

**When == is OK:** only when comparing against errors that are never wrapped (rare). In practice, always use `errors.Is` for sentinel comparisons.

---

## Exercise 4 — type assertion instead of errors.As

`submitForm(-5)` returns a `*fmt.wrapError` wrapping a `*ValidationError`. A direct type assertion `err.(*ValidationError)` checks the outermost type only — it fails because the outer type is `*fmt.wrapError`.

`errors.As` calls `Unwrap()` repeatedly until it finds a value assignable to `*ValidationError`.

**Fix:**
```go
var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Printf("field: %s\n", ve.Field)
}
```

**Note:** `errors.As` takes a pointer-to-pointer (`&ve`). The target type must be a pointer to the error type you're looking for, or an interface type.
