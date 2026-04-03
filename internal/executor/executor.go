package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Verdict string

const (
	VerdictAccepted          Verdict = "accepted"
	VerdictWrongAnswer       Verdict = "wrong_answer"
	VerdictCompilationError  Verdict = "compilation_error"
	VerdictRuntimeError      Verdict = "runtime_error"
	VerdictTimeLimitExceeded Verdict = "time_limit_exceeded"
)

type TestCase struct {
	Input    string
	Expected string
}

type TestCaseResult struct {
	Number   int     `json:"number"`
	Passed   bool    `json:"passed"`
	Input    string  `json:"input,omitempty"`
	Expected string  `json:"expected,omitempty"`
	Output   string  `json:"output,omitempty"`
	Time     float64 `json:"time"`
}

type Result struct {
	Verdict   Verdict          `json:"verdict"`
	Message   string           `json:"message,omitempty"`
	TestCases []TestCaseResult `json:"testCases,omitempty"`
	TotalTime float64          `json:"totalTime,omitempty"`
}

func GetLanguage(name string) (*Language, error) {
	lang, ok := languages[name]
	if !ok {
		return nil, fmt.Errorf("linguagem indisponivel: %s", name)
	}
	return &lang, nil
}

func CompareOutput(actual, expected string) bool {
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		lines := strings.Split(s, "\n")
		var normLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				fields := strings.Fields(trimmed)
				normLines = append(normLines, strings.Join(fields, " "))
			}
		}
		return strings.Join(normLines, "\n")
	}
	return normalize(actual) == normalize(expected)
}

func Execute(ctx context.Context, code string, lang string, testCases []TestCase, timeLimitSec int, memoryLimitMB int) (*Result, error) {
	langDef, err := GetLanguage(lang)
	if err != nil {
		return nil, err
	}

	if lang == "java" || lang == "kotlin" {
		if globalPool == nil {
			return nil, fmt.Errorf("pool JVM ausente")
		}
		worker := GetJVMWorker()
		res := worker.Execute(lang, code, testCases, timeLimitSec)

		tainted := res.Verdict == VerdictTimeLimitExceeded || res.Verdict == VerdictRuntimeError || res.Verdict == ""
		ReleaseJVMWorker(worker, tainted)
		return res, nil
	}

	tmpDir, err := os.MkdirTemp("", "exec-*")
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar pasta: %w", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	filename := "code" + langDef.Extension
	codePath := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("falha ao salvar script: %w", err)
	}

	if langDef.NeedCompile {
		compileCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		compileOutput, err := Compile(compileCtx, langDef, tmpDir, codePath)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return &Result{Verdict: VerdictTimeLimitExceeded, Message: "tempo limite na compilação"}, nil
			}
			return &Result{Verdict: VerdictCompilationError, Message: string(compileOutput)}, nil
		}
	}

	var results []TestCaseResult
	overallVerdict := VerdictAccepted
	totalTime := 0.0

	for i, tc := range testCases {
		output, runTime, err := RunCase(ctx, langDef, tmpDir, tc.Input, timeLimitSec, memoryLimitMB)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				overallVerdict = VerdictTimeLimitExceeded
			} else {
				overallVerdict = VerdictRuntimeError
				if len(output) == 0 {
					output = []byte(err.Error())
				}
			}
			results = append(results, TestCaseResult{
				Number:   i + 1,
				Passed:   false,
				Input:    tc.Input,
				Expected: tc.Expected,
				Output:   string(output),
				Time:     runTime,
			})
			break
		}

		passed := CompareOutput(string(output), tc.Expected)
		results = append(results, TestCaseResult{
			Number:   i + 1,
			Passed:   passed,
			Input:    tc.Input,
			Expected: tc.Expected,
			Output:   string(output),
			Time:     runTime,
		})
		totalTime += runTime

		if !passed {
			overallVerdict = VerdictWrongAnswer
			break
		}
	}

	return &Result{
		Verdict:   overallVerdict,
		TestCases: results,
		TotalTime: totalTime,
	}, nil
}

func trimSpace(s string) string {
	start, end := 0, len(s)-1
	for start <= end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end >= start && (s[end] == ' ' || s[end] == '\t' || s[end] == '\n' || s[end] == '\r') {
		end--
	}
	if start > end {
		return ""
	}
	return s[start : end+1]
}

func VerdictToString(v Verdict) string {
	switch v {
	case VerdictAccepted:
		return "Accepted"
	case VerdictWrongAnswer:
		return "Wrong Answer"
	case VerdictRuntimeError:
		return "Runtime Error"
	case VerdictTimeLimitExceeded:
		return "Time Limit Exceeded"
	case VerdictCompilationError:
		return "Compilation Error"
	default:
		return ""
	}
}
