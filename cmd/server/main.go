package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"Algorithms/internal/executor"
)

type TestRequest struct {
	Code      string   `json:"code"`
	Language  string   `json:"language"`
	Inputs    []string `json:"inputs"`
	Expected  []string `json:"expected"`
	TimeLimit int      `json:"timeLimit,omitempty"`
	MemLimit  int      `json:"memLimit,omitempty"`
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Inputs) != len(req.Expected) {
		http.Error(w, "inputs and expected must have same length", http.StatusBadRequest)
		return
	}

	testCases := make([]executor.TestCase, len(req.Inputs))
	for i := range req.Inputs {
		testCases[i] = executor.TestCase{
			Input:    req.Inputs[i],
			Expected: req.Expected[i],
		}
	}

	timeLimit := req.TimeLimit
	if timeLimit <= 0 {
		timeLimit = 5
	}
	memLimit := req.MemLimit
	if memLimit <= 0 {
		memLimit = 256
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeLimit)*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, req.Code, req.Language, testCases, timeLimit, memLimit)
	if err != nil {
		http.Error(w, "executor error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		return
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      http.HandlerFunc(testHandler),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Server listening on port %s with timeouts", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
