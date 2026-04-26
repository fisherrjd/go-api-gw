// Exercise 3: Goroutines, channels, sync primitives
//
// This is a concurrent job processor. It:
//   1. Accepts a batch of "jobs" (user IDs to enrich)
//   2. Fans them out to worker goroutines
//   3. Collects results into a shared map
//   4. Returns when all workers finish
//
// Expected behavior:
//   - All jobs are processed exactly once
//   - Results map has one entry per job
//   - No data races
//   - No goroutine leaks
//   - processBatch returns only after all workers are done
//
// There are 3 bugs. Find them. One will deadlock, one is a data race,
// one silently drops work.

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job struct {
	UserID int
}

type Result struct {
	UserID int
	Name   string
	Err    error
}

// enrich simulates a downstream call.
func enrich(ctx context.Context, j Job) Result {
	select {
	case <-ctx.Done():
		return Result{UserID: j.UserID, Err: ctx.Err()}
	case <-time.After(10 * time.Millisecond):
		return Result{UserID: j.UserID, Name: fmt.Sprintf("user-%d", j.UserID)}
	}
}

// processBatch fans out jobs to workers and collects results.
func processBatch(ctx context.Context, jobs []Job, workers int) map[int]Result {
	jobCh := make(chan Job)
	results := make(map[int]Result)

	var wg sync.WaitGroup

	// spawn workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			// BUG ZONE — wg.Done() is somewhere wrong
			for j := range jobCh {
				r := enrich(ctx, j)
				// BUG ZONE — concurrent write to shared map with no protection
				results[r.UserID] = r
			}
		}()
	}

	// feed jobs
	go func() {
		for _, j := range jobs {
			jobCh <- j
		}
		close(jobCh)
	}()

	wg.Wait()
	return results
}

// notify sends a notification for each result.
// It fans out across goroutines and uses a channel to collect errors.
func notify(results []Result) []error {
	errCh := make(chan error) // BUG ZONE — buffer size matters here

	var wg sync.WaitGroup
	for _, r := range results {
		wg.Add(1)
		r := r // shadow for Go < 1.22
		go func() {
			defer wg.Done()
			if r.Err != nil {
				errCh <- r.Err
			}
		}()
	}

	// close errCh after all senders finish
	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	return errs
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jobs := make([]Job, 10)
	for i := range jobs {
		jobs[i] = Job{UserID: i + 1}
	}

	results := processBatch(ctx, jobs, 3)
	fmt.Printf("got %d results\n", len(results))

	// convert to slice for notify
	var rs []Result
	for _, r := range results {
		rs = append(rs, r)
	}
	errs := notify(rs)
	fmt.Printf("notify errors: %v\n", errs)
}
