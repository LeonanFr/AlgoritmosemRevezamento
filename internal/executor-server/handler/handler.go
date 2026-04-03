package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"Algorithms/internal/executor"
	"Algorithms/internal/executor-server/config"
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

	langDef, err := executor.GetLanguage(req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "exec-*")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
		}
	}(tmpDir)

	var filename string
	switch req.Language {
	case "java":
		filename = "Main.java"
	case "kotlin":
		filename = "code.kt"
	default:
		filename = "code" + langDef.Extension
	}

	codePath := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(codePath, []byte(req.Code), 0644); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if langDef.NeedCompile {
		compileOutput, err := executor.Compile(r.Context(), langDef, tmpDir, codePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(executor.Result{
				Verdict: executor.VerdictCompilationError,
				Message: string(compileOutput),
			})
			return
		}
	}

	testCases := make([]executor.TestCase, len(req.Inputs))
	for i := range req.Inputs {
		testCases[i] = executor.TestCase{
			Input:    req.Inputs[i],
			Expected: req.Expected[i],
		}
	}

	if req.Mode == "submit" || req.Mode == "" {
		for _, tc := range testCases {
			ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeLimit)*time.Second)
			output, _, err := executor.RunCase(ctx, langDef, tmpDir, tc.Input, timeLimit, memLimit)
			cancel()

			var v executor.Verdict
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					v = executor.VerdictTimeLimitExceeded
				} else {
					v = executor.VerdictRuntimeError
				}
			} else if !executor.CompareOutput(string(output), tc.Expected) {
				v = executor.VerdictWrongAnswer
			} else {
				v = executor.VerdictAccepted
			}

			if v != executor.VerdictAccepted {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(executor.Result{
					Verdict: v,
					Message: executor.VerdictToString(v),
				})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(executor.Result{
			Verdict: executor.VerdictAccepted,
			Message: "Accepted",
		})
		return
	}

	parallel := h.cfg.CasesParallel
	idxChan := make(chan int, len(testCases))
	resChan := make(chan caseResult, len(testCases))
	var wg sync.WaitGroup

	for w := 0; w < parallel; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range idxChan {
				tc := testCases[idx]
				ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeLimit)*time.Second)
				output, rt, err := executor.RunCase(ctx, langDef, tmpDir, tc.Input, timeLimit, memLimit)
				cancel()
				resChan <- caseResult{idx, output, rt, err}
			}
		}()
	}

	for i := range testCases {
		idxChan <- i
	}
	close(idxChan)
	go func() {
		wg.Wait()
		close(resChan)
	}()

	results := make([]executor.TestCaseResult, len(testCases))
	overallVerdict := executor.VerdictAccepted
	totalTime := 0.0
	severity := map[executor.Verdict]int{
		executor.VerdictTimeLimitExceeded: 1,
		executor.VerdictRuntimeError:      2,
		executor.VerdictWrongAnswer:       3,
		executor.VerdictAccepted:          4,
	}
	currentSeverity := 4

	for res := range resChan {
		var v executor.Verdict
		if res.err != nil {
			if errors.Is(res.err, context.DeadlineExceeded) {
				v = executor.VerdictTimeLimitExceeded
			} else {
				v = executor.VerdictRuntimeError
			}
		} else if !executor.CompareOutput(string(res.output), testCases[res.index].Expected) {
			v = executor.VerdictWrongAnswer
		} else {
			v = executor.VerdictAccepted
		}

		results[res.index] = executor.TestCaseResult{
			Number:   res.index + 1,
			Passed:   v == executor.VerdictAccepted,
			Input:    testCases[res.index].Input,
			Expected: testCases[res.index].Expected,
			Output:   string(res.output),
			Time:     res.runTime,
		}
		totalTime += res.runTime
		if s, ok := severity[v]; ok && s < currentSeverity {
			currentSeverity = s
			overallVerdict = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(executor.Result{
		Verdict:   overallVerdict,
		TestCases: results,
		TotalTime: totalTime,
	})
}
