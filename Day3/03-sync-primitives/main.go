package main

import (
	"fmt"
	"sync"
)

// LESSON: sync.WaitGroup, sync.Mutex, sync.RWMutex, sync.Once
//
// These are the four primitives you'll see in every production Go codebase.
// Each has a specific use case and a specific footgun.

// --- sync.WaitGroup ---
//
// Counts goroutines in flight. Add(n) before starting, Done() when finished,
// Wait() blocks until counter hits zero.
//
// Rules:
//   - Add() BEFORE the goroutine starts (not inside it)
//   - Done() via defer as the goroutine's first line
//   - Never copy a WaitGroup (pass by pointer or use at package level)

func lessonWaitGroup() {
	fmt.Println("=== WaitGroup ===")

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1) // increment BEFORE go — if inside goroutine, Add and Wait race
		go func(n int) {
			defer wg.Done()
			fmt.Printf("worker %d\n", n)
		}(i)
	}

	wg.Wait()
	fmt.Println("all done")
}

// --- sync.Mutex ---
//
// Exclusive lock: only one goroutine holds it at a time.
// Use for any shared mutable state.
//
// Lock() blocks until acquired. Unlock() releases.
// Always pair with defer for safety.
//
// FOOTGUN: copying a Mutex copies its lock state — use pointer receivers.
//   go vet will warn: "passes lock by value"

type Counter struct {
	mu    sync.Mutex
	value int
}

func (c *Counter) Inc() { // pointer receiver — critical for Mutex
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func lessonMutex() {
	fmt.Println("\n=== Mutex ===")

	c := &Counter{}
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc()
		}()
	}
	wg.Wait()
	fmt.Println("count:", c.Get()) // always 1000

	// BROKEN — value receiver copies the mutex:
	//   func (c Counter) Inc() { c.mu.Lock() ... }
	//   Each call gets a fresh copy of the mutex — no mutual exclusion.
}

// --- sync.RWMutex ---
//
// Read/Write mutex: multiple readers OR one writer at a time.
// Use when reads are frequent and writes are rare.
//
// RLock()/RUnlock() for reads — they do NOT block each other.
// Lock()/Unlock() for writes — exclusive, blocks all readers and writers.

type Cache struct {
	mu    sync.RWMutex
	store map[string]string
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock() // multiple goroutines can hold RLock simultaneously
	defer c.mu.RUnlock()
	v, ok := c.store[key]
	return v, ok
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock() // exclusive — no readers or writers during write
	defer c.mu.Unlock()
	c.store[key] = value
}

func lessonRWMutex() {
	fmt.Println("\n=== RWMutex ===")
	c := &Cache{store: make(map[string]string)}
	c.Set("greeting", "hello")
	v, ok := c.Get("greeting")
	fmt.Printf("get: %q, found=%v\n", v, ok)
}

// --- sync.Once ---
//
// Guarantees a function runs exactly once, even under concurrent calls.
// Classic use: lazy initialization of a singleton (DB connection, config).
//
// Note: if the function panics, the panic propagates and the function
// is NOT considered to have run — subsequent calls will call it again.

var (
	instance *Cache
	initOnce sync.Once
)

func getCache() *Cache {
	initOnce.Do(func() {
		fmt.Println("initializing cache (runs once)")
		instance = &Cache{store: make(map[string]string)}
	})
	return instance
}

func lessonOnce() {
	fmt.Println("\n=== Once ===")
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = getCache() // only one goroutine triggers initialization
		}()
	}
	wg.Wait()
	fmt.Println("cache:", getCache()) // same instance every time
}

// --- The mutex-copy footgun ---
//
// This is what `go vet` warns about. Know it for interviews.

type BadCounter struct {
	mu    sync.Mutex
	value int
}

func (c BadCounter) BadInc() { // VALUE receiver — COPIES the mutex
	c.mu.Lock()   // locks a copy — no effect on the original
	c.value++     // mutates the copy — no effect on the original
	c.mu.Unlock() // unlocks the copy
}

func lessonMutexCopyBug() {
	fmt.Println("\n=== Mutex copy bug ===")
	fmt.Println("BadCounter.BadInc() uses a value receiver.")
	fmt.Println("Each call gets a fresh Mutex copy — no mutual exclusion.")
	fmt.Println("Counter never actually increments. go vet catches this.")
	fmt.Println("Fix: always use pointer receivers on types containing sync primitives.")
}

func main() {
	lessonWaitGroup()
	lessonMutex()
	lessonRWMutex()
	lessonOnce()
	lessonMutexCopyBug()

	fmt.Println("\n--- EXERCISES ---")
	fmt.Println("1. Implement a thread-safe LRU cache with Get/Set using sync.RWMutex.")
	fmt.Println("   (Hint: reads downgrade to RLock; writes need full Lock)")
	fmt.Println()
	fmt.Println("2. Write a pool of DB connections using sync.Once for init.")
	fmt.Println("   getDB() should return the same *sql.DB every time.")
	fmt.Println()
	fmt.Println("3. Find the bug: what's wrong with this pattern?")
	fmt.Println("   func (s *Server) handle(w http.ResponseWriter, r *http.Request) {")
	fmt.Println("       s2 := *s          // copy the server")
	fmt.Println("       s2.mu.Lock()      // lock the copy")
	fmt.Println("       defer s2.mu.Unlock()")
	fmt.Println("       s2.counter++")
	fmt.Println("   }")
}
