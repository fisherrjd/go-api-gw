package main

import (
	"context"
	"fmt"
	"time"
)

// LESSON: Goroutines, Channels, Select, Context
//
// Goroutines are cheap green threads. `go f()` starts one.
// Channels are typed, synchronized message pipes between goroutines.
// select lets a goroutine wait on multiple channels at once.
// context.Context carries cancellation signals across goroutine boundaries.

// --- Part 1: Goroutines ---
//
// `go func()` starts a goroutine. It returns immediately — no handle, no join.
// You need channels or sync primitives to wait for it.

func lessonGoroutines() {
	fmt.Println("=== Goroutines ===")

	done := make(chan struct{})
	go func() {
		fmt.Println("goroutine: hello")
		close(done) // signal completion
	}()
	<-done // block until goroutine closes the channel
	fmt.Println("goroutine finished")
}

// --- Part 2: Channels ---
//
// Unbuffered: `make(chan T)` — sender blocks until a receiver is ready.
// Buffered:   `make(chan T, n)` — sender blocks only when buffer is full.
//
// Range over a channel reads until it's closed.
// Receive-only parameter type: `<-chan T`
// Send-only parameter type:    `chan<- T`

func producer(out chan<- int, n int) {
	for i := 0; i < n; i++ {
		out <- i
	}
	close(out) // sender closes; receiver range will exit
}

func lessonChannels() {
	fmt.Println("\n=== Channels ===")

	// Unbuffered: each send blocks until main receives
	ch := make(chan int)
	go producer(ch, 5)
	for v := range ch {
		fmt.Println("got:", v)
	}

	// Buffered: producer can fill buffer without a matching receiver
	buf := make(chan int, 3)
	buf <- 10
	buf <- 20
	buf <- 30
	// buf <- 40 // would block — buffer full
	fmt.Println("buffered:", <-buf, <-buf, <-buf)
}

// --- Part 3: Select ---
//
// select picks whichever case is ready. If multiple are ready, one is chosen
// at random (not first-wins). A `default` case makes it non-blocking.

func lessonSelect() {
	fmt.Println("\n=== Select ===")

	fast := make(chan string, 1)
	slow := make(chan string, 1)

	fast <- "fast result"
	// slow is empty

	select {
	case v := <-fast:
		fmt.Println("got:", v)
	case v := <-slow:
		fmt.Println("got:", v)
	default:
		fmt.Println("nothing ready") // would print if both were empty
	}

	// Timeout pattern: race a result against a timer
	result := make(chan int, 1)
	go func() {
		time.Sleep(5 * time.Millisecond)
		result <- 42
	}()

	select {
	case v := <-result:
		fmt.Println("result:", v)
	case <-time.After(1 * time.Second):
		fmt.Println("timeout")
	}
}

// --- Part 4: context.Context ---
//
// Context carries: deadlines, cancellation signals, and request-scoped values.
// Always pass ctx as the FIRST argument to any function that does I/O or blocks.
// Use r.Context() in HTTP handlers — NOT context.Background() — or cancellation breaks.
//
// ctx.Done() returns a channel that closes when the context is cancelled.
// ctx.Err() returns the reason: context.Canceled or context.DeadlineExceeded.

func doWork(ctx context.Context, id int) error {
	select {
	case <-time.After(50 * time.Millisecond): // simulates real work
		fmt.Printf("worker %d: done\n", id)
		return nil
	case <-ctx.Done():
		fmt.Printf("worker %d: cancelled: %v\n", id, ctx.Err())
		return ctx.Err()
	}
}

func lessonContext() {
	fmt.Println("\n=== Context ===")

	// WithTimeout: automatically cancels after duration
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel() // always defer cancel — prevents goroutine leak even if ctx expires first

	for i := 0; i < 3; i++ {
		if err := doWork(ctx, i); err != nil {
			fmt.Println("stopping early:", err)
			break
		}
	}

	// WithCancel: manually cancel, e.g., when one worker fails
	ctx2, cancel2 := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel2() // pull the plug
	}()
	doWork(ctx2, 99) // will be cancelled before 50ms
}

func main() {
	lessonGoroutines()
	lessonChannels()
	lessonSelect()
	lessonContext()

	fmt.Println("\n--- EXERCISES ---")
	fmt.Println("See below. Comment out lessonX() calls above and work one at a time.")
	fmt.Println()
	exerciseHints()
}

// --- EXERCISES ---
//
// Work these after reading the lesson above. Check solutions.md when done.

func exerciseHints() {
	fmt.Println("Exercise 1: Fan-out + collect")
	fmt.Println("  Write fanOut(ctx, ids []int) []string")
	fmt.Println("  Start one goroutine per id. Each goroutine sleeps 10ms then sends")
	fmt.Println("  fmt.Sprintf(\"user-%d\", id) to a results channel.")
	fmt.Println("  Collect all results and return them. No goroutine leaks.")
	fmt.Println()

	fmt.Println("Exercise 2: First-result wins")
	fmt.Println("  Write firstOf(ctx context.Context, fns []func(context.Context) (string, error)) string")
	fmt.Println("  Run all fns concurrently. Return the first non-error result.")
	fmt.Println("  Cancel the context when you have a winner.")
	fmt.Println()

	fmt.Println("Exercise 3: Timeout with select")
	fmt.Println("  Write withTimeout(work func() int, d time.Duration) (int, bool)")
	fmt.Println("  Return (result, true) if work finishes in time, (0, false) on timeout.")
	fmt.Println("  Use select + time.After. No context needed.")
}
