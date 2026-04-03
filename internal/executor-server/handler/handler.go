package handler

import (
	"Algorithms/internal/executor"
	"Algorithms/internal/executor-server/config"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

type ExecuteRequest struct {
	Code      string   `json:"code"`
	Language  string   `json:"language"`
	Inputs    []string `json:"inputs"`
	Expected  []string `json:"expected"`
	TimeLimit int      `json:"timeLimit,omitempty"`
	MemLimit  int      `json:"memLimit,omitempty"`
	Mode      string   `json:"mode,omitempty"`
}

type caseResult struct {
	index   int
	output  []byte
	runTime float64
	err     error
}

type Handler struct {
	cfg  *config.Config
	pool chan struct{}
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg:  cfg,
		pool: make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", h.authMiddleware(h.executeHandler))
	mux.HandleFunc("/health", h.healthHandler)
	mux.HandleFunc("/stress", h.authMiddleware(h.stressHandler))
	return mux
}

func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		expected := "Bearer " + h.cfg.AuthToken
		if token != expected {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) stressHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cmd := exec.Command("go", "run", "cmd/stress-executor/stress.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(output)
		_, _ = w.Write([]byte("\nErro: " + err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(output)
}

func (h *Handler) executeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Inputs) != len(req.Expected) || len(req.Inputs) == 0 {
		http.Error(w, "invalid test cases", http.StatusBadRequest)
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "test"
	}

	var testCases []executor.TestCase
	if mode == "test" {
		limit := 3
		if len(req.Inputs) < limit {
			limit = len(req.Inputs)
		}
		testCases = make([]executor.TestCase, limit)
		for i := 0; i < limit; i++ {
			testCases[i] = executor.TestCase{
				Input:    req.Inputs[i],
				Expected: req.Expected[i],
			}
		}
	} else {
		testCases = make([]executor.TestCase, len(req.Inputs))
		for i := range req.Inputs {
			testCases[i] = executor.TestCase{
				Input:    req.Inputs[i],
				Expected: req.Expected[i],
			}
		}
	}

	timeLimit := req.TimeLimit
	if timeLimit <= 0 {
		timeLimit = h.cfg.DefaultTimeLimit
	}
	memLimit := req.MemLimit
	if memLimit <= 0 {
		memLimit = h.cfg.DefaultMemLimit
	}

	h.pool <- struct{}{}
	defer func() { <-h.pool }()

	result, err := executor.Execute(r.Context(), req.Code, req.Language, testCases, timeLimit, memLimit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if mode == "submit" {
		submitResp := struct {
			Verdict executor.Verdict `json:"verdict"`
			Message string           `json:"message,omitempty"`
		}{
			Verdict: result.Verdict,
		}

		if result.Verdict != executor.VerdictAccepted {
			if result.Message != "" {
				submitResp.Message = result.Message
			} else if len(result.TestCases) > 0 {
				for _, tc := range result.TestCases {
					if !tc.Passed {
						submitResp.Message = fmt.Sprintf("Falha no caso %d", tc.Number)
						break
					}
				}
			}
			if submitResp.Message == "" {
				submitResp.Message = executor.VerdictToString(result.Verdict)
			}
		}

		_ = json.NewEncoder(w).Encode(submitResp)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}
