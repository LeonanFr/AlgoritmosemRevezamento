package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func runWithLimits(ctx context.Context, lang Language, workDir string, input string, timeLimitSec int, memoryLimitMB int) ([]byte, float64, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeLimitSec)*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	cmdArgs := buildCommandArgs(lang, memoryLimitMB)

	if runtime.GOOS == "linux" && memoryLimitMB > 0 && !isVM(lang.Name) {
		memKB := memoryLimitMB * 1024
		cmdLine := fmt.Sprintf("ulimit -v %d && exec %s", memKB, strings.Join(cmdArgs, " "))
		cmd = exec.CommandContext(ctxTimeout, "sh", "-c", cmdLine)
	} else {
		cmd = exec.CommandContext(ctxTimeout, cmdArgs[0], cmdArgs[1:]...)
	}

	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(input)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start).Seconds()

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
func buildCommandArgs(lang Language, memoryLimitMB int) []string {
	if memoryLimitMB <= 0 {
		args := make([]string, len(lang.RunCmd))
		copy(args, lang.RunCmd)
		return args
	}

	switch lang.Name {
	case "Java":
		return []string{"java", fmt.Sprintf("-Xmx%dm", memoryLimitMB), "-cp", ".", "Main"}
	case "Kotlin":
		return []string{"java", fmt.Sprintf("-Xmx%dm", memoryLimitMB), "-jar", "code.jar"}
	case "JavaScript":
		return []string{"node", fmt.Sprintf("--max-old-space-size=%d", memoryLimitMB), "code.js"}
	default:
		args := make([]string, len(lang.RunCmd))
		copy(args, lang.RunCmd)
		return args
	}
}
