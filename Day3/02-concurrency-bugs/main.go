package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LESSON: The 6 Classic Concurrency Bugs
//
// These are the bugs interviewers plant. Learn to spot them instantly.
// Each lessonX() shows the broken version with a comment, then the fix.

// --- Bug 1: Loop variable capture (pre-Go 1.22) ---
//
// In Go < 1.22, loop variables are shared across iterations.
// Goroutines launched in the loop all close over the SAME variable.
// By the time they run, the loop has finished and `i` is its final value.
//
// Go 1.22+ fixed this — each iteration gets its own variable.
// Interviewers still test the old behavior. Know both.

func lessonLoopCapture() {
	fmt.Println("=== Bug 1: Loop variable capture ===")

	// BROKEN (pre-1.22 behavior — all print the same value):
	//   for i := 0; i < 3; i++ {
	//       go func() { fmt.Println(i) }()
	//   }
	// All goroutines likely print "3" because they share `i`.

	// FIX 1: shadow the variable
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		i := i // new variable scoped to this iteration
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("loop (shadowed):", i)
		}()
	}
	wg.Wait()

	// FIX 2: pass as argument
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			fmt.Println("loop (arg):", n)
		}(i)
	}
	wg.Wait()
}

// --- Bug 2: Forgotten wg.Done() → deadlock ---
//
// wg.Wait() blocks until the counter reaches zero.
// If any goroutine panics or returns early without calling Done(),
// Wait() blocks forever.
//
// Fix: always `defer wg.Done()` as the FIRST line inside the goroutine.

func lessonForgottenDone() {
	fmt.Println("\n=== Bug 2: Forgotten wg.Done() ===")

	// BROKEN — if doWork panics, Done never runs:
	//   wg.Add(1)
	//   go func() {
	//       doWork()
	//       wg.Done() // never reached if doWork panics
	//   }()

	// FIX: defer runs even on panic
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done() // first line, always runs
			fmt.Printf("worker %d done\n", n)
		}(i)
	}
	wg.Wait()
	fmt.Println("all workers finished")
}

// --- Bug 3: Send on closed channel → panic ---
//
// Sending to a closed channel panics immediately.
// Receiving from a closed channel returns the zero value (and `false` for the ok).
//
// Rule: only the SENDER closes. Never close from the receiver side.
// If multiple goroutines send, use sync.Once or a separate coordinator.

func lessonSendOnClosed() {
	fmt.Println("\n=== Bug 3: Send on closed channel ===")

	ch := make(chan int, 3)

	// Safe: check ok when receiving
	ch <- 1
	close(ch)

	v, ok := <-ch
	fmt.Printf("received %d, ok=%v\n", v, ok) // 1, true

	v, ok = <-ch
	fmt.Printf("received %d, ok=%v\n", v, ok) // 0, false — channel closed and empty

	// BROKEN — don't do this:
	// close(ch)  // already closed → panic: close of closed channel
	// ch <- 2    // closed channel → panic: send on closed channel

	// Pattern for multiple senders: use sync.Once
	ch2 := make(chan int, 5)
	var once sync.Once
	closeOnce := func() { once.Do(func() { close(ch2) }) }

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ch2 <- n
		}(i)
	}
	go func() {
		wg.Wait()
		closeOnce() // safe — only closes once even if called multiple times
	}()
	for v := range ch2 {
		fmt.Println("ch2 received:", v)
	}
}

// --- Bug 4: Deadlock on unbuffered channel ---
//
// An unbuffered channel send blocks until a receiver is ready.
// If both sender and receiver are in the same goroutine, you deadlock.
// If the receiver goroutine never starts, you deadlock.

func lessonDeadlock() {
	fmt.Println("\n=== Bug 4: Deadlock (demonstration — we avoid it) ===")

	// BROKEN — this deadlocks (don't uncomment in main):
	//   ch := make(chan int)
	//   ch <- 1   // blocks forever — no receiver
	//   <-ch

	// FIX: start the receiver first
	ch := make(chan int)
	go func() { ch <- 42 }() // sender in goroutine
	fmt.Println("received:", <-ch)

	// Or use a buffered channel if you don't need synchronization:
	buf := make(chan int, 1)
	buf <- 42
	fmt.Println("buffered:", <-buf)
}

// --- Bug 5: Goroutine leak from un-cancelled context ---
//
// A goroutine that blocks on I/O or a channel with no escape will run forever.
// The most common cause: goroutines started with context.Background() inside
// a request handler, so they ignore the request's cancellation signal.

func lessonGoroutineLeak() {
	fmt.Println("\n=== Bug 5: Goroutine leak ===")

	// BROKEN — goroutine has no exit path if nothing sends to work:
	//   go func() {
	//       for job := range work { process(job) } // leaks if work never closes
	//   }()

	// FIX: always give goroutines an escape via context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	work := make(chan int)
	go func() {
		for {
			select {
			case j, ok := <-work:
				if !ok {
					return
				}
				fmt.Println("processed:", j)
			case <-ctx.Done():
				fmt.Println("worker: context cancelled, exiting")
				return
			}
		}
	}()

	work <- 1
	work <- 2
	// context expires after 50ms; goroutine exits cleanly
	time.Sleep(60 * time.Millisecond)
}

// --- Bug 6: Data race on shared map ---
//
// Maps in Go are NOT safe for concurrent read+write.
// Concurrent writes, or a write concurrent with a read, is undefined behavior.
// The race detector will catch it; production crashes are unpredictable.
//
// Fix: use sync.Mutex, sync.RWMutex, or sync.Map.

func lessonDataRace() {
	fmt.Println("\n=== Bug 6: Data race on shared map ===")

	// BROKEN — concurrent writes to a plain map = data race:
	//   m := map[int]int{}
	//   var wg sync.WaitGroup
	//   for i := 0; i < 10; i++ {
	//       wg.Add(1)
	//       go func(n int) { defer wg.Done(); m[n] = n }(i) // RACE
	//   }

	// FIX: mutex protects the map
	m := map[int]int{}
	var mu sync.Mutex
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
	fmt.Printf("map has %d entries\n", len(m))
}

func main() {
	lessonLoopCapture()
	lessonForgottenDone()
	lessonSendOnClosed()
	lessonDeadlock()
	lessonGoroutineLeak()
	lessonDataRace()

	fmt.Println("\n--- EXERCISES ---")
	fmt.Println("1. Spot the bug: which bug does exercise3_concurrency.go in go-bugs/ contain?")
	fmt.Println("   (Look at processBatch — find all 3 bugs before reading the code comments)")
	fmt.Println()
	fmt.Println("2. Write a function that starts N goroutines, each incrementing a shared counter.")
	fmt.Println("   Make it correct with a mutex. Then run 'go run -race main.go' to verify.")
	fmt.Println()
	fmt.Println("3. Write a pipeline: generator → transformer → printer, all connected by channels.")
	fmt.Println("   Add context cancellation so all stages exit cleanly.")
}
