// Exercise 2: HTTP context, request lifecycle, and response bodies
//
// This is an API gateway handler that:
//   1. Checks an Authorization header
//   2. Calls a downstream user service to validate the token
//   3. Injects the user ID into the request context
//   4. Calls the next handler in the chain
//
// Expected behavior:
//   - Missing or invalid "Authorization" header → 401
//   - Downstream call honors the caller's context (cancellation, deadlines)
//   - Validated user ID is available downstream via context key
//   - No goroutine / resource leaks
//
// There are 3 bugs. Find them.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type contextKey string

const userIDKey contextKey = "userID"

// --- downstream client ---

type AuthClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
}

// ValidateToken calls the downstream auth service.
// Returns the userID on success, or an error.
func (c *AuthClient) ValidateToken(ctx context.Context, token string) (int, error) {
	// BUG ZONE — context is intentionally wrong here
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/validate", nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	// BUG ZONE — response body is never closed
	// (hint: what happens to connections over time?)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("auth service returned %d", resp.StatusCode)
	}

	var result struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return result.UserID, nil
}

// --- middleware ---

func AuthMiddleware(client *AuthClient, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := client.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- handler ---

type ProfileHandler struct{}

func (h *ProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// BUG ZONE — what can go wrong with this type assertion?
	userID := r.Context().Value(userIDKey).(int)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user_id": userID,
		"message": "profile ok",
	})
}

// --- drain helper used in a loop ---

func processResponses(urls []string) error {
	for _, url := range urls {
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			return err
		}
		// BUG ZONE — what's wrong with defer here?
		defer resp.Body.Close()
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	client := NewAuthClient("http://auth-service")
	mux := http.NewServeMux()
	mux.Handle("/profile", AuthMiddleware(client, &ProfileHandler{}))
	fmt.Println("listening on :8080")
	http.ListenAndServe(":8080", mux)
}
