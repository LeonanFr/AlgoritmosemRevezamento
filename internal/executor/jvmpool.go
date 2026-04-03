package executor

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
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
		cmd := exec.Command("java", "-Xmx"+xmx, "-cp", classpath, "Worker")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		worker := &JVMWorker{
			cmd:    cmd,
			stdin:  stdin,
			stdout: bufio.NewReader(stdout),
		}
		pool.workers <- worker
	}
	globalPool = pool
	return nil
}

func GetJVMWorker() *JVMWorker {
	return <-globalPool.workers
}

func ReleaseJVMWorker(w *JVMWorker, tainted bool) {
	if tainted {
		_ = w.cmd.Process.Kill()
		cmd := exec.Command("java", "-Xmx"+poolXmx, "-cp", poolClasspath, "Worker")
		stdin, _ := cmd.StdinPipe()
		stdout, _ := cmd.StdoutPipe()
		_ = cmd.Start()
		newWorker := &JVMWorker{
			cmd:    cmd,
			stdin:  stdin,
			stdout: bufio.NewReader(stdout),
		}
		globalPool.workers <- newWorker
	} else {
		globalPool.workers <- w
	}
}

func (w *JVMWorker) Execute(lang, code string, testCases []TestCase, timeLimit int) *Result {
	fmt.Fprintln(w.stdin, lang)
	fmt.Fprintln(w.stdin, code)
	fmt.Fprintln(w.stdin, "---END_CODE---")
	fmt.Fprintln(w.stdin, len(testCases))
	for _, tc := range testCases {
		fmt.Fprintln(w.stdin, tc.Input)
		fmt.Fprintln(w.stdin, tc.Expected)
	}
	fmt.Fprintln(w.stdin, timeLimit)

	verdict, _ := w.stdout.ReadString('\n')
	verdict = strings.TrimSpace(verdict)

	switch verdict {
	case "ACCEPTED":
		timeLine, _ := w.stdout.ReadString('\n')
		var totalTime float64
		_, _ = fmt.Sscanf(timeLine, "%f", &totalTime)
		return &Result{Verdict: VerdictAccepted, TotalTime: totalTime}
	case "COMPILATION_ERROR":
		msg, _ := w.stdout.ReadString('\n')
		return &Result{Verdict: VerdictCompilationError, Message: strings.TrimSpace(msg)}
	case "RUNTIME_ERROR":
		caseLine, _ := w.stdout.ReadString('\n')
		var caseNum int
		_, _ = fmt.Sscanf(caseLine, "%d", &caseNum)
		return &Result{Verdict: VerdictRuntimeError, Message: fmt.Sprintf("Erro no caso %d", caseNum+1)}
	case "TIME_LIMIT_EXCEEDED":
		caseLine, _ := w.stdout.ReadString('\n')
		var caseNum int
		_, _ = fmt.Sscanf(caseLine, "%d", &caseNum)
		return &Result{Verdict: VerdictTimeLimitExceeded, Message: fmt.Sprintf("TLE no caso %d", caseNum+1)}
	case "WRONG_ANSWER":
		caseLine, _ := w.stdout.ReadString('\n')
		var caseNum int
		_, _ = fmt.Sscanf(caseLine, "%d", &caseNum)
		return &Result{Verdict: VerdictWrongAnswer, Message: fmt.Sprintf("WA no caso %d", caseNum+1)}
	default:
		return &Result{Verdict: VerdictRuntimeError, Message: "Resposta invalida"}
	}
}
