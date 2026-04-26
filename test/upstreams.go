package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type response struct {
	Upstream string `json:"upstream"`
	Version  string `json:"version"`
	Path     string `json:"path"`
	Method   string `json:"method"`
}

func makeHandler(version string, port int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{
			Upstream: fmt.Sprintf("localhost:%d", port),
			Version:  version,
			Path:     r.URL.Path,
			Method:   r.Method,
		})
	})
}

func main() {
	upstreams := []struct {
		version string
		port    int
	}{
		{"v1", 8001},
		{"v3", 8002},
		{"v4", 8003},
	}

	var wg sync.WaitGroup
	for _, u := range upstreams {
		u := u
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := fmt.Sprintf(":%d", u.port)
			log.Printf("upstream %s listening on %s", u.version, addr)
			if err := http.ListenAndServe(addr, makeHandler(u.version, u.port)); err != nil {
				log.Fatalf("upstream %s failed: %v", u.version, err)
			}
		}()
	}
	wg.Wait()
}
