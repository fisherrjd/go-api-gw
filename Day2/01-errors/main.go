package main

import (
	"errors"
	"fmt"
)

// LESSON: Error Handling Idioms
//
// Go has no exceptions. Every fallible function returns (result, error).
// The caller is responsible for checking. Always.
//
// Three idioms:
//   1. Wrap errors with context:   fmt.Errorf("op: %w", err)
//   2. Check for sentinel errors:  errors.Is(err, ErrFoo)
//   3. Extract typed errors:       errors.As(err, &myErr)

// Sentinel errors — named package-level values used as signals.
// errors.Is walks the chain to find them; == only checks the outermost value.
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
)

// --- Pattern 1: Wrap errors — always add context ---
//
// Bad:  return err                              (loses all context)
// Good: return fmt.Errorf("loadUser: %w", err) (wraps with context)
//
// %w (not %v) preserves the chain so errors.Is / errors.As can unwrap it later.

func fetchUser(id int) (string, error) {
	if id == 0 {
		return "", ErrNotFound
	}
	return "alice", nil
}

func loadUser(id int) (string, error) {
	name, err := fetchUser(id)
	if err != nil {
		return "", fmt.Errorf("loadUser(%d): %w", id, err)
	}
	return name, nil
}

func handleGetUser(id int) error {
	_, err := loadUser(id)
	if err != nil {
		return fmt.Errorf("handleGetUser: %w", err)
	}
	return nil
}

func lessonWrapping() {
	fmt.Println("=== Wrapping ===")
	err := handleGetUser(0)
	fmt.Println(err)
	// Output: handleGetUser: loadUser(0): not found
	// Every layer that failed is visible in the message.
}

// --- Pattern 2: errors.Is — find a sentinel anywhere in the chain ---

func lessonErrorsIs() {
	fmt.Println("\n=== errors.Is ===")
	err := handleGetUser(0)

	// == compares pointer identity. A wrapped error is a new value — never equal.
	fmt.Println("err == ErrNotFound:", err == ErrNotFound) // false

	// errors.Is walks the chain via Unwrap() until it finds ErrNotFound.
	fmt.Println("errors.Is(err, ErrNotFound): ", errors.Is(err, ErrNotFound)) // true
}

// --- Pattern 3: errors.As — extract a typed error from the chain ---
//
// Typed error — use when the caller needs structured data, not just a message.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func validateAge(age int) error {
	if age < 0 {
		return &ValidationError{Field: "age", Message: "must be >= 0"}
	}
	return nil
}

func submitForm(age int) error {
	if err := validateAge(age); err != nil {
		return fmt.Errorf("submitForm: %w", err)
	}
	return nil
}

func lessonErrorsAs() {
	fmt.Println("\n=== errors.As ===")
	err := submitForm(-1)

	// Type assertion only checks the outermost type — fails because err is wrapped.
	if _, ok := err.(*ValidationError); !ok {
		fmt.Println("type assertion: failed (expected — error is wrapped)")
	}

	// errors.As unwraps the chain until it finds a *ValidationError.
	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("errors.As: field %q failed: %s\n", ve.Field, ve.Message)
	}
}

// --- Exercises ---
//
// For each exercise:
//   1. Read the code. Predict the output. What SHOULD it print?
//   2. Run main.go and verify your prediction.
//   3. Fix the bug. Run again to confirm.
//   4. See solutions.md for explanations.

// Exercise 1: Ignored error
// fetchUser(0) returns an error. What is `name`? What should we do instead?
func exercise1() {
	fmt.Println("\n--- Ex 1: Ignored error ---")
	name, _ := fetchUser(0) // BUG: _ silently discards the error
	fmt.Printf("name: %q\n", name)
}

// Exercise 2: Shadowed err
// step() always returns an error. This function always returns nil. Why?
func step() (string, error) {
	return "", errors.New("step failed")
}

func exercise2() error {
	fmt.Println("\n--- Ex 2: Shadowed err ---")
	var err error

	// BUG: := declares new `result` and `err` scoped to this block.
	// The outer `err` is never modified.
	if result, err := step(); err == nil {
		fmt.Println("step succeeded:", result)
	} else {
		fmt.Println("inner err:", err) // this err is the block-scoped one
	}

	fmt.Println("outer err:", err) // always nil
	return err                     // always nil — this is the bug
}

// Exercise 3: == instead of errors.Is
// handleGetUser(0) wraps ErrNotFound. Why does this check never fire?
func exercise3() {
	fmt.Println("\n--- Ex 3: == vs errors.Is ---")
	err := handleGetUser(0)

	if err == ErrNotFound { // BUG: wrapped error is not the same pointer
		fmt.Println("not found (you will never see this)")
	} else {
		fmt.Println("fell through — err:", err)
	}
}

// Exercise 4: type assertion instead of errors.As
// submitForm(-5) wraps a *ValidationError. Why does the assertion fail?
func exercise4() {
	fmt.Println("\n--- Ex 4: type assertion vs errors.As ---")
	err := submitForm(-5)

	ve, ok := err.(*ValidationError) // BUG: outer type is *fmt.wrapError, not *ValidationError
	if ok {
		fmt.Printf("field: %s\n", ve.Field)
	} else {
		fmt.Println("type assertion failed — err is:", err)
	}
}

func main() {
	lessonWrapping()
	lessonErrorsIs()
	lessonErrorsAs()

	fmt.Println("\n\n========== EXERCISES ==========")
	fmt.Println("Predict the output of each exercise before reading the result.\n")

	exercise1()
	_ = exercise2()
	exercise3()
	exercise4()

	fmt.Println("\n--- done: check solutions.md ---")
}
