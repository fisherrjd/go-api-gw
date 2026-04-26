package main

import (
	"fmt"
	"sync"
)

// ====================================================================
// LESSON: Pointers, Values, Receivers, Slices
// ====================================================================
//
// Java: everything is a reference (except primitives). You never think about this.
// Go: you choose every time. Getting it wrong is silent — no compile error.
//
// Two questions to ask for every method:
//   1. Does this method need to mutate the receiver? → pointer receiver
//   2. Does this type contain a mutex? → always pointer receiver (never copy)

// ====================================================================
// PATTERN 1: Value receiver — does NOT mutate the caller
// ====================================================================

type Counter struct {
	count int
}

// Value receiver: receives a COPY of Counter. Mutation is local.
func (c Counter) IncrementBad() {
	c.count++ // modifies the copy, not the original
}

// Pointer receiver: receives a pointer. Mutation affects the original.
func (c *Counter) Increment() {
	c.count++
}

func (c Counter) Value() int { return c.count }

func lessonReceivers() {
	fmt.Println("=== Value vs pointer receivers ===")

	c := Counter{}

	c.IncrementBad()                              // looks like it increments, but doesn't
	fmt.Println("after IncrementBad:", c.Value()) // 0 — not 1

	c.Increment()                                 // pointer receiver — actually mutates
	fmt.Println("after Increment:   ", c.Value()) // 1
}

// ====================================================================
// PATTERN 2: Slices — header vs backing array
// ====================================================================
//
// A slice is a struct with three fields: (ptr, len, cap)
// When you pass a slice to a function, you pass a COPY of that struct.
//
//   original := []int{1, 2, 3}   // ptr→[1,2,3], len=3, cap=3
//   modify(original)              // function gets copy of (ptr, len, cap)
//
// Inside modify, you can mutate ELEMENTS (same backing array).
// But if you append and it reallocates, caller sees nothing.
// And even if it doesn't reallocate, the caller's len doesn't update.

func modifyElement(s []int) {
	s[0] = 99 // mutates backing array — caller DOES see this
}

func appendToSlice(s []int) {
	s = append(s, 99) // if cap exceeded: new backing array; caller sees nothing
	// even if cap NOT exceeded: caller's len is unchanged
	fmt.Println("inside append:", s)
}

func lessonSlices() {
	fmt.Println("\n=== Slices ===")

	nums := []int{1, 2, 3}

	// Element mutation via index: caller sees it (same backing array)
	modifyElement(nums)
	fmt.Println("after modifyElement:", nums) // [99 2 3]

	// Reset
	nums = []int{1, 2, 3}
	appendToSlice(nums)
	fmt.Println("after appendToSlice:", nums) // [1 2 3] — unchanged
	// append created a new header in the function; caller's header unchanged
}

// ====================================================================
// PATTERN 3: append can or cannot share backing array
// ====================================================================
//
// append(s, x):
//   - If len(s) < cap(s): writes into existing backing array, returns new header
//     with len+1. SHARING STILL ACTIVE — both slices see the backing array.
//   - If len(s) == cap(s): allocates new backing array. NO SHARING.
//
// This is where subslice bugs live.

func lessonAppendSharing() {
	fmt.Println("\n=== append sharing ===")

	// Make a slice with extra capacity
	a := make([]int, 3, 6) // len=3, cap=6: [0,0,0]
	a[0], a[1], a[2] = 1, 2, 3

	// b shares backing array with a (cap has room)
	b := append(a, 4)    // b=[1,2,3,4], writes into a's backing array
	fmt.Println("a:", a) // [1 2 3] — len=3, so a[3]=4 is invisible to a
	fmt.Println("b:", b) // [1 2 3 4]

	// Now mutate a[0] via b — they share the same backing array!
	b[0] = 99
	fmt.Println("after b[0]=99, a:", a) // [99 2 3] — a is affected!
	fmt.Println("after b[0]=99, b:", b) // [99 2 3 4]

	fmt.Println("\n--- subslice shares backing array ---")
	original := []int{1, 2, 3, 4, 5}
	sub := original[1:3] // [2, 3] — same backing array
	sub[0] = 99
	fmt.Println("original after sub[0]=99:", original) // [1 99 3 4 5]
}

// ====================================================================
// PATTERN 4: Never copy a mutex
// ====================================================================
//
// sync.Mutex has internal state. Copying it copies that state, including
// any current lock status. The copy and original become independent mutexes,
// but their lock states are wrong.
//
// `go vet` will catch this. Interviewers love it.

type SafeCounter struct {
	mu    sync.Mutex
	count int
}

func (c *SafeCounter) Inc() { // pointer receiver — correct
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func lessonMutex() {
	fmt.Println("\n=== Mutex copy ===")

	sc := SafeCounter{}
	sc.Inc()
	sc.Inc()

	// BUG (don't run this):
	// copy := sc  // copies the mutex — go vet: "copying lock value"
	// copy.Inc()  // operates on a copied mutex — undefined behavior

	fmt.Println("count:", sc.count)
	fmt.Println("never copy a struct that contains sync.Mutex — use pointer receivers")
}

// ====================================================================
// EXERCISES
// ====================================================================

// Exercise 1: Value receiver that should mutate
// addItem should add the item to the cart. It doesn't.
// Fix it.
type Cart struct {
	items []string
}

func (c *Cart) AddItem(item string) { // BUG: value receiver
	c.items = append(c.items, item)
	fmt.Println("inside AddItem, items:", c.items)
}

func (c *Cart) Len() int { return len(c.items) }

func exercise1() {
	fmt.Println("\n--- Ex 1: Value receiver mutation ---")
	cart := Cart{}
	cart.AddItem("apple")
	cart.AddItem("banana")
	fmt.Println("cart items:", cart.items) // what do you expect?
	fmt.Println("cart len:", cart.Len())
}

// Exercise 2: append doesn't propagate
// fill is supposed to populate the slice. The caller never sees the items.
// Fix it — two possible approaches.
func fill(s *[]string) {
	*s = append(*s, "a", "b", "c") // BUG: modifies local copy of header
}

func exercise2() {
	fmt.Println("\n--- Ex 2: append doesn't propagate ---")
	data := make([]string, 0, 10)
	fill(&data)
	fmt.Println("data after fill:", data) // what do you expect?
	fmt.Println("len:", len(data))
}

// Exercise 3: Subslice mutation surprise
// Predict what original looks like after the operations below.
// No bug to fix — just understand the output.
func exercise3() {
	fmt.Println("\n--- Ex 3: Subslice sharing (predict output) ---")

	original := []int{10, 20, 30, 40, 50}
	window := original[1:4] // [20, 30, 40] (same cap though so its 6)

	window[0] = 99

	fmt.Println("window:  ", window)   // ?
	fmt.Println("original:", original) // ?

	// Now append to window — does it affect original?
	window = append(window, 888)
	fmt.Println("\nafter append(window, 888):")
	fmt.Println("window:  ", window)   // ? -->
	fmt.Println("original:", original) // ? -->

	// If you're surprised, try: original[4] == ?
}

// Exercise 4: Mutex copied by value
// This code has a mutex bug that go vet will catch.
// Identify it and fix it without running (it's unsafe to run).
type RateLimiter struct {
	mu      sync.Mutex
	counter int
	limit   int
}

func NewRateLimiter(limit int) *RateLimiter { // BUG: returns value, contains mutex
	return &RateLimiter{limit: limit}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.counter >= r.limit {
		return false
	}
	r.counter++
	return true
}

func exercise4() {
	fmt.Println("\n--- Ex 4: Mutex copy (read only — don't run the buggy path) ---")

	// This creates a copy of RateLimiter including its mutex:
	// rl := NewRateLimiter(10)   // RateLimiter value — mutex copied
	// rl.Allow()

	// What's the fix? (Two options — name them both)
	fmt.Println("identify the bug and name the two fixes — see solutions.md")
}

func main() {
	lessonReceivers()
	lessonSlices()
	lessonAppendSharing()
	lessonMutex()

	fmt.Println("\n\n========== EXERCISES ==========")
	fmt.Print("Predict the output before running. Then fix.\n")

	// exercise1()
	// exercise2()
	// exercise3()
	exercise4()

	fmt.Println("\n--- done: check solutions.md ---")
}
