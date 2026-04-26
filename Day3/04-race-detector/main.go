package main

import (
	"fmt"
	"sync"
)

// LESSON: The Race Detector
//
// Run: go run -race main.go
//
// The race detector instruments memory accesses at runtime and reports
// when two goroutines access the same memory concurrently and at least
// one is a write, with no synchronization between them.
//
// Output format:
//   WARNING: DATA RACE
//   Write at 0x... by goroutine N:
//     main.raceOnCounter()
//         /path/to/main.go:42
//   Previous read at 0x... by goroutine M:
//     ...
//   Goroutine N (running) created at:
//     ...
//
// How to read it:
//   - "Write at" + "Previous read" (or write) = the two conflicting accesses
//   - The stack traces show WHERE each access happened
//   - "Created at" shows WHERE the goroutine was launched

// --- Race 1: counter without a lock ---
//
// Two goroutines increment a shared int.
// This is a classic data race — run with -race to see the report.

var counter int // shared, unprotected

func raceOnCounter() {
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // RACE: read-modify-write is not atomic
		}()
	}
	wg.Wait()
	fmt.Println("counter (racy):", counter) // result is non-deterministic
}

// --- Race 2: map concurrent write ---

func raceOnMap() {
	m := map[int]int{}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m[n] = n // RACE: concurrent map writes
		}(i)
	}
	wg.Wait()
}

// --- Fixed versions ---

func fixedCounter() {
	var mu sync.Mutex
	var count int
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			count++
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println("counter (fixed):", count) // always 1000
}

func fixedMap() {
	var mu sync.Mutex
	m := map[int]int{}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			mu.Lock()
			m[n] = n
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	fmt.Printf("map: %d entries\n", len(m))
}

func main() {
	fmt.Println("Run with: go run -race main.go")
	fmt.Println("The race detector will report the races in raceOnCounter and raceOnMap.")
	fmt.Println()

	// Comment these in to see races:
	// raceOnCounter()
	// raceOnMap()

	// These are safe:
	fixedCounter()
	fixedMap()

	fmt.Println()
	fmt.Println("To see races, uncomment raceOnCounter() and raceOnMap() and re-run with -race.")
	fmt.Println()
	fmt.Println("What to look for in the race detector output:")
	fmt.Println("  1. Which goroutines are conflicting")
	fmt.Println("  2. What lines each goroutine accesses")
	fmt.Println("  3. Where each goroutine was created")
	fmt.Println("  4. Whether the conflict is read/write or write/write")
}
