package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func runWithLimits(ctx context.Context, lang Language, workDir string, input string, timeLimitSec int, memoryLimitMB int) ([]byte, float64, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeLimitSec)*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	if runtime.GOOS == "linux" && memoryLimitMB > 0 && !isVM(lang.Name) {
		memKB := memoryLimitMB * 1024
		cmdLine := fmt.Sprintf("ulimit -v %d && exec %s", memKB, strings.Join(lang.RunCmd, " "))
		cmd = exec.CommandContext(ctxTimeout, "sh", "-c", cmdLine)
	} else {
		cmd = exec.CommandContext(ctxTimeout, lang.RunCmd[0], lang.RunCmd[1:]...)
	}

	if runtime.GOOS == "linux" {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.Setpgid = true
	}

	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(input)
	cmd.WaitDelay = 10 * time.Millisecond

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 0, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, 0, err
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, 0, err
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctxTimeout.Done():
			if cmd.Process != nil {
				if runtime.GOOS == "linux" {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				} else {
					_ = cmd.Process.Kill()
				}
			}
		case <-done:
		}
	}()

	outBytes, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(stderr)
	err = cmd.Wait()
	close(done)

	elapsed := time.Since(start).Seconds()
	output := append(outBytes, errBytes...)

	if ctx.Err() != nil {
		return output, elapsed, ctx.Err()
	}
	if errors.Is(ctxTimeout.Err(), context.DeadlineExceeded) {
		return output, elapsed, context.DeadlineExceeded
	}
	if err != nil {
		return output, elapsed, err
	}
	return output, elapsed, nil
}

func isVM(langName string) bool {
	return langName == "Java" || langName == "Kotlin" || langName == "JavaScript"
}
