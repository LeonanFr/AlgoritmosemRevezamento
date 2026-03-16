package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func Execute(ctx context.Context, code string, lang string, testCases []TestCase, timeLimitSec int, memoryLimitMB int) (*Result, error) {
	langDef, ok := languages[lang]
	if !ok {
		return nil, fmt.Errorf("linguagem indisponivel: %s", lang)
	}

	tmpDir, err := os.MkdirTemp("", "exec-*")
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar pasta: %w", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	var filename string
	switch lang {
	case "java":
		filename = "Main.java"
	case "kotlin":
		filename = "code.kt"
	default:
		filename = "code" + langDef.Extension
	}

	codePath := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("falha ao salvar script: %w", err)
	}

	if langDef.NeedCompile {
		compileOutput, err := compile(ctx, langDef, tmpDir, codePath)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return &Result{
					Verdict: VerdictTimeLimitExceeded,
					Message: "esgotamento de cronometro na fase de build",
				}, nil
			}
			return &Result{
				Verdict: VerdictCompilationError,
				Message: string(compileOutput),
			}, nil
		}
	}

	var results []TestCaseResult
	overallVerdict := VerdictAccepted
	totalTime := 0.0

	for i, tc := range testCases {
		output, runTime, err := runWithLimits(ctx, langDef, tmpDir, tc.Input, timeLimitSec, memoryLimitMB)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				overallVerdict = VerdictTimeLimitExceeded
			} else {
				overallVerdict = VerdictRuntimeError
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

		passed := compareOutput(string(output), tc.Expected)
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

func compareOutput(actual, expected string) bool {
	norm := func(s string) string {
		s = trimSpace(s)
		s = strings.ReplaceAll(s, "\r\n", "\n")
		return s
	}
	return norm(actual) == norm(expected)
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
