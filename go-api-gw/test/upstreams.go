// Test upstream servers for gateway katas
// Run: go run test/upstreams.go
//
// Provides:
//   - :8001 API server (returns JSON with request info)
//   - :8002 Static file server (serves ./testdata if exists)
//   - :8003 Default app (returns "default" + request path)
//   - :8004/8005 Additional upstreams for load balancing tests

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type serverConfig struct {
	port    string
	handler http.Handler
	name    string
}

func main() {
	// Fail after 3 requests (for health check testing)
	failCount := 0
	maxFails := 3

	configs := []serverConfig{
		{
			port: "8001",
			name: "api",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]any{
					"server":     "api-8001",
					"method":     r.Method,
					"path":       r.URL.Path,
					"headers":    r.Header,
					"request_id": r.Header.Get("X-Request-ID"),
					"timestamp":  time.Now().Unix(),
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}),
		},
		{
			port: "8002",
			name: "static",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, "static-8002: %s %s\n", r.Method, r.URL.Path)
			}),
		},
		{
			port: "8003",
			name: "default",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, "default-8003: %s %s\n", r.Method, r.URL.Path)
			}),
		},
		{
			port: "8004",
			name: "api-2",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"server":"api-8004"}`)
			}),
		},
		{
			port: "8005",
			name: "api-3-failing",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				failCount++
				if failCount > maxFails {
					w.WriteHeader(http.StatusServiceUnavailable)
					fmt.Fprintf(w, "failing-8005: simulated failure")
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"server":"api-8005"}`)
			}),
		},
	}

	for _, cfg := range configs {
		go func(c serverConfig) {
			log.Printf("Starting %s server on :%s", c.name, c.port)
			if err := http.ListenAndServe(":"+c.port, c.handler); err != nil {
				log.Fatalf("Server %s failed: %v", c.name, err)
			}
		}(cfg)
	}

	log.Println("Test upstreams running. Ctrl+C to stop.")\t
	select {}
}

// Helper to check if running
func init() {
	if len(os.Args) > 1 && os.Args[1] == "check" {
		// Health check mode
		os.Exit(0)
	}
}
