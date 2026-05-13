package executor

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type JVMWorker struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
}

type JVMPool struct {
	workers chan *JVMWorker
}

var (
	globalPool    *JVMPool
	poolClasspath string
	poolXmx       string
)

func InitJVMPool(size int, classpath, xmx string) error {
	poolClasspath = classpath
	poolXmx = xmx
	pool := &JVMPool{
		workers: make(chan *JVMWorker, size),
	}
	for i := 0; i < size; i++ {
		w, err := createWorker()
		if err != nil {
			return err
		}
		pool.workers <- w
	}
	globalPool = pool
	return nil
}

func createWorker() (*JVMWorker, error) {
	cmd := exec.Command("java", "-Xmx"+poolXmx, "-cp", poolClasspath, "Worker")

	var stderr strings.Builder
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	worker := &JVMWorker{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	ready, err := worker.stdout.ReadString('\n')
	if err != nil || strings.TrimSpace(ready) != "READY" {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()

		return nil, fmt.Errorf(
			"worker não enviou READY: readErr=%v; stdout=%q; stderr=%q; classpath=%q",
			err,
			strings.TrimSpace(ready),
			strings.TrimSpace(stderr.String()),
			poolClasspath,
		)
	}

	return worker, nil
}

func GetJVMWorker() *JVMWorker {
	for {
		w := <-globalPool.workers

		if pingWorker(w) {
			return w
		}

		killWorker(w)

		for {
			newW, err := createWorker()
			if err == nil {
				return newW
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

func pingWorker(w *JVMWorker) bool {
	if w == nil || w.stdin == nil || w.stdout == nil {
		return false
	}

	if _, err := fmt.Fprintln(w.stdin, "PING"); err != nil {
		return false
	}

	type pingResult struct {
		resp string
		err  error
	}

	ch := make(chan pingResult, 1)

	go func() {
		resp, err := w.stdout.ReadString('\n')
		ch <- pingResult{resp: resp, err: err}
	}()

	select {
	case result := <-ch:
		return result.err == nil && strings.TrimSpace(result.resp) == "PONG"

	case <-time.After(2 * time.Second):
		return false
	}
}

func killWorker(w *JVMWorker) {
	if w == nil || w.cmd == nil || w.cmd.Process == nil {
		return
	}

	_ = w.cmd.Process.Kill()
	_ = w.cmd.Wait()
}

func ReleaseJVMWorker(w *JVMWorker, tainted bool) {
	if tainted {
		killWorker(w)

		for {
			newW, err := createWorker()
			if err == nil {
				globalPool.workers <- newW
				return
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	globalPool.workers <- w
}

func readUntilSep(reader *bufio.Reader) string {
	var sb strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "---SEP---" {
			break
		}
		sb.WriteString(line)
	}
	return strings.TrimRight(sb.String(), "\r\n")
}

func (w *JVMWorker) Execute(lang, code string, testCases []TestCase, timeLimit int) *Result {
	fmt.Fprintln(w.stdin, lang)
	fmt.Fprintln(w.stdin, code)
	fmt.Fprintln(w.stdin, "---END_CODE---")
	fmt.Fprintln(w.stdin, timeLimit)

	statusLine, err := w.stdout.ReadString('\n')
	if err != nil {
		return &Result{Verdict: VerdictRuntimeError, Message: "Worker falhou na compilacao"}
	}
	status := strings.TrimSpace(statusLine)

	if status == "COMPILATION_ERROR" {
		msg := readUntilSep(w.stdout)
		return &Result{Verdict: VerdictCompilationError, Message: strings.TrimSpace(msg)}
	}

	if status != "COMPILED_OK" {
		return &Result{Verdict: VerdictRuntimeError, Message: "Erro inesperado do worker: " + status}
	}

	var results []TestCaseResult
	overallVerdict := VerdictAccepted
	totalTime := 0.0

	for i, tc := range testCases {
		fmt.Fprintln(w.stdin, "RUN_CASE")
		fmt.Fprintln(w.stdin, tc.Input)
		fmt.Fprintln(w.stdin, "---SEP---")

		caseStatusLine, err := w.stdout.ReadString('\n')
		if err != nil {
			overallVerdict = VerdictRuntimeError
			break
		}
		caseStatus := strings.TrimSpace(caseStatusLine)

		if caseStatus == "TIME_LIMIT_EXCEEDED" {
			_ = readUntilSep(w.stdout)
			overallVerdict = VerdictTimeLimitExceeded
			results = append(results, TestCaseResult{
				Number: i + 1, Passed: false, Input: tc.Input, Expected: tc.Expected, Output: "", Time: float64(timeLimit),
			})
			break
		} else if caseStatus == "RUNTIME_ERROR" {
			errorMsg := readUntilSep(w.stdout)
			overallVerdict = VerdictRuntimeError
			results = append(results, TestCaseResult{
				Number: i + 1, Passed: false, Input: tc.Input, Expected: tc.Expected, Output: errorMsg, Time: 0,
			})
			break
		} else if caseStatus == "OK" {
			timeStr, _ := w.stdout.ReadString('\n')
			var runTime float64
			fmt.Sscanf(strings.TrimSpace(timeStr), "%f", &runTime)

			output := readUntilSep(w.stdout)
			passed := CompareOutput(output, tc.Expected)

			results = append(results, TestCaseResult{
				Number: i + 1, Passed: passed, Input: tc.Input, Expected: tc.Expected, Output: output, Time: runTime,
			})
			totalTime += runTime

			if !passed {
				overallVerdict = VerdictWrongAnswer
				break
			}
		} else {
			overallVerdict = VerdictRuntimeError
			break
		}
	}

	fmt.Fprintln(w.stdin, "STOP_CASES")

	return &Result{
		Verdict:   overallVerdict,
		TestCases: results,
		TotalTime: totalTime,
	}
}
