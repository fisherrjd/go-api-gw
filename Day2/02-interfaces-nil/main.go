package main

import (
	"errors"
	"fmt"
)

// ====================================================================
// LESSON: Interfaces and Nil
// ====================================================================
//
// This section contains THE most common Go interview bug: the typed nil.
// Java has no equivalent. It requires rethinking what "nil" means.

// ====================================================================
// PATTERN 1: Interfaces are satisfied implicitly
// ====================================================================
//
// No `implements` keyword. If a type has the right methods, it satisfies
// the interface. This is the opposite of Java's explicit `implements`.
//
// Convention: "Accept interfaces, return structs."
// Accept interfaces = callers can pass anything that fits
// Return structs = callers get the full concrete type, not a box

type Storer interface {
	Save(key, value string) error
	Load(key string) (string, error)
}

type MemoryStore struct {
	data map[string]string
}

func NewMemoryStore() *MemoryStore { // return concrete type, not Storer
	return &MemoryStore{data: make(map[string]string)}
}

func (m *MemoryStore) Save(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *MemoryStore) Load(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("load %q: not found", key)
	}
	return v, nil
}

// This function accepts an interface — any Storer works.
func process(s Storer, key, value string) error {
	if err := s.Save(key, value); err != nil {
		return fmt.Errorf("process: %w", err)
	}
	return nil
}

func lessonImplicit() {
	fmt.Println("=== Implicit interface satisfaction ===")
	store := NewMemoryStore()    // *MemoryStore
	_ = process(store, "k", "v") // *MemoryStore satisfies Storer — no cast needed
	fmt.Println("*MemoryStore satisfies Storer automatically")
}

// ====================================================================
// PATTERN 2: THE TYPED NIL TRAP
// ====================================================================
//
// An interface value has two components: (type, value).
// An interface is nil ONLY when BOTH type and value are nil.
//
// When you return a nil pointer of a concrete type as an interface,
// the interface holds (type=*MyError, value=nil) — NOT nil!
//
// This is the most surprising Go behavior for developers from other languages.

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

// BUGGY version — classic interview bug
func riskyOpBuggy(fail bool) error {
	var err *AppError // nil pointer to AppError

	if fail {
		err = &AppError{Code: 500, Message: "something broke"}
	}

	return err // BUG: returns (*AppError)(nil) when fail=false
	// Interface value: (type=*AppError, value=nil) — NOT nil!
}

// CORRECT version
func riskyOpCorrect(fail bool) error {
	if fail {
		return &AppError{Code: 500, Message: "something broke"}
	}
	return nil // returns (type=nil, value=nil) — actually nil
}

func lessonTypedNil() {
	fmt.Println("\n=== THE TYPED NIL TRAP ===")

	// Buggy version: returns (*AppError)(nil)
	err := riskyOpBuggy(false)
	fmt.Printf("buggy (fail=false): err=%v, err==nil: %v\n", err, err == nil)
	// Output: err=<nil>, err==nil: false  <-- SURPRISING

	// err.Error() would call (*AppError).Error() with a nil receiver
	// If Error() dereferences e, this panics.

	// Correct version: returns untyped nil
	err2 := riskyOpCorrect(false)
	fmt.Printf("correct (fail=false): err=%v, err==nil: %v\n", err2, err2 == nil)
	// Output: err=<nil>, err==nil: true  <-- expected
}

// ====================================================================
// PATTERN 3: Nil receivers — methods can be called on nil pointers
// ====================================================================
//
// In Go, you CAN call a method on a nil pointer — it just passes nil as
// the receiver. If the method dereferences the receiver, it panics.
// If it handles nil explicitly, it's fine (and sometimes useful).

func (e *AppError) IsRetryable() bool {
	if e == nil {
		return false // nil receiver is safe if you check
	}
	return e.Code >= 500
}

func lessonNilReceiver() {
	fmt.Println("\n=== Nil receiver ===")
	var e *AppError                                              // nil
	fmt.Println("nil receiver, IsRetryable():", e.IsRetryable()) // false, no panic
	// This works because IsRetryable checks `if e == nil`.
	// If it did `return e.Code >= 500` without the nil check, it would panic.
}

// ====================================================================
// PATTERN 4: Interface comparison — both type AND value must match
// ====================================================================

var ErrSentinel = errors.New("sentinel")

type wrappedErr struct{ inner error }

func (w *wrappedErr) Error() string { return w.inner.Error() }
func (w *wrappedErr) Unwrap() error { return w.inner }

func lessonInterfaceComparison() {
	fmt.Println("\n=== Interface comparison ===")

	var a error = ErrSentinel
	var b error = ErrSentinel
	fmt.Println("same sentinel, == :", a == b) // true — same pointer

	wrapped := &wrappedErr{inner: ErrSentinel}
	fmt.Println("wrapped == ErrSentinel:        ", error(wrapped) == ErrSentinel)    // false
	fmt.Println("errors.Is(wrapped, ErrSentinel):", errors.Is(wrapped, ErrSentinel)) // true
}

// ====================================================================
// EXERCISES
// ====================================================================

// Exercise 1: THE typed nil bug
// getError(false) is supposed to signal "no error". But it doesn't.
// Predict the output of err == nil, then explain why, then fix it.
type DBError struct{ msg string }

func (e *DBError) Error() string { return e.msg }

func getError(fail bool) error {
	var err *DBError // nil pointer

	if fail {
		err = &DBError{msg: "db connection failed"}
		return err
	}

	return nil // BUG: typed nil — returns (*DBError)(nil) as error interface
}

func exercise1() {
	fmt.Println("\n--- Ex 1: Typed nil ---")

	err := getError(false)
	fmt.Println("err == nil:", err == nil) // what do you expect?

	if err != nil {
		fmt.Println("we entered the error branch — but fail=false!")
		fmt.Printf("err value: %v\n", err)
	}
}

// Exercise 2: Nil pointer dereference via interface
// This compiles fine. When does it panic? Fix it.
type Logger struct {
	prefix string
}

func (l *Logger) Log(msg string) {
	if l == nil {
		return
	}
	fmt.Printf("[%s] %s\n", l.prefix, msg) // BUG: panics if l is nil
}

type Service struct {
	log *Logger
}

func exercise2() {
	fmt.Println("\n--- Ex 2: Nil dereference via method ---")

	svc := &Service{}       // log is nil — zero value for pointer is nil
	svc.log.Log("starting") // what happens here?
}

// Exercise 3: Implicit interface — will this compile?
// Uncomment the assignment and run. If it fails, fix Handler to satisfy Doer.
type Doer interface {
	Do(input string) (string, error)
}

type Handler struct {
	name string
}

func (h *Handler) Do(input string) (string, error) {
	return h.name + ":" + input, nil
}

// Does *Handler satisfy Doer? What's missing?
func exercise3() {
	fmt.Println("\n--- Ex 3: Interface satisfaction ---")

	// Uncomment to test — does this compile?
	var d Doer = &Handler{name: "test"}
	result, _ := d.Do("hello")
	fmt.Println(result)

	fmt.Println("uncomment the assignment in exercise3 to test")
}

// Exercise 4: Typed nil in an error return — the production version
// This is the pattern that shows up in real code. What's wrong with it?
// Fix without changing the function signature.
type QueryError struct {
	Query string
	Err   error
}

func (e *QueryError) Error() string { return fmt.Sprintf("query %q: %v", e.Query, e.Err) }
func (e *QueryError) Unwrap() error { return e.Err }

func runQuery(query string, fail bool) (*QueryError, error) {
	if fail {
		return &QueryError{Query: query, Err: errors.New("timeout")}, nil
	}
	return nil, nil
}

func exercise4() {
	fmt.Println("\n--- Ex 4: Typed nil in real code ---")

	qErr, err := runQuery("SELECT 1", false)
	if err != nil {
		fmt.Println("system error:", err)
		return
	}

	// Caller checks qErr as an error interface
	var asErr error = qErr
	if asErr != nil {
		fmt.Println("query error (should not print):", asErr)
	} else {
		fmt.Println("no error (correct)")
	}
}

func main() {
	// lessonImplicit()
	// lessonTypedNil()
	// lessonNilReceiver()
	// lessonInterfaceComparison()

	fmt.Println("\n\n========== EXERCISES ==========")
	fmt.Println("Predict the output before running. Then fix.\n")

	// exercise1()
	// exercise2() //is disabled — it panics. Read it and understand why, then fix it.
	// fmt.Println("\n--- Ex 2 disabled (panics) — read the code and explain the bug ---")
	// exercise3()
	exercise4()

	fmt.Println("\n--- done: check solutions.md ---")
}
