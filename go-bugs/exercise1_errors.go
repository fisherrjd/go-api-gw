// Exercise 1: Error handling & typed nil
//
// This is a small user-lookup service. It parses a JSON request body,
// validates the user ID, fetches the user from a store, and returns JSON.
//
// Expected behavior:
//   - POST /user with body {"id": 42} returns {"id": 42, "name": "alice"}
//   - POST /user with body {"id": 0}  returns 400 with "invalid user id"
//   - POST /user with a missing ID     returns 400 with "invalid user id"
//   - POST /user with unknown ID       returns 404 with "user not found"
//
// There are 3 bugs. Find them.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// --- domain types ---

type UserRequest struct {
	ID int `json:"id"`
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// NotFoundError is a typed sentinel for missing records.
type NotFoundError struct {
	ID int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("user %d not found", e.ID)
}

// --- store ---

var store = map[int]User{
	42: {ID: 42, Name: "alice"},
	99: {ID: 99, Name: "bob"},
}

func fetchUser(id int) (*User, *NotFoundError) {
	u, ok := store[id]
	if !ok {
		return nil, &NotFoundError{ID: id}
	}
	return &u, nil
}

// --- handler ---

type UserHandler struct {
	prefix string
}

func (h UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := validate(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, nfe := fetchUser(req.ID)
	// BUG ZONE — something is off with how nfe is checked
	var nfeErr error = nfe
	if nfeErr != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// BUG ZONE — prefix is supposed to tag logs; handler should mutate its own state
	h.prefix = fmt.Sprintf("[user:%d]", user.ID)
	fmt.Printf("%s serving request\n", h.prefix)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// --- validation ---

func validate(req UserRequest) error {
	if req.ID <= 0 {
		return errors.New("invalid user id")
	}
	return nil
}

// --- wiring ---

func newUserHandler() *UserHandler {
	return &UserHandler{prefix: "[user:?]"}
}

func main() {
	mux := http.NewServeMux()

	h := newUserHandler()
	mux.Handle("POST /user", h)

	// BUG ZONE — demonstrating the shadowed-err pattern; this always logs nil
	var globalErr error
	if config, globalErr := loadConfig(); globalErr != nil {
		fmt.Println("config error:", globalErr)
	} else {
		fmt.Println("config loaded:", config)
	}
	fmt.Println("startup error was:", globalErr) // what does this print?

	fmt.Println("listening on :8080")
	http.ListenAndServe(":8080", mux)
}

func loadConfig() (string, error) {
	return "default-config", nil
}
