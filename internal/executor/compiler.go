package executor

import (
	"context"
	"os/exec"
	"time"
)

func Compile(ctx context.Context, lang *Language, workDir string, codePath string) ([]byte, error) {
	if lang.CompileCmd == nil {
		return nil, nil
	}

	args := make([]string, len(lang.CompileCmd))
	for i, arg := range lang.CompileCmd {
		switch arg {
		case "code.c", "code.cpp", "Main.java", "code.kt":
			args[i] = codePath
		default:
			args[i] = arg
		}
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.WaitDelay = 10 * time.Millisecond

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return output, ctx.Err()
		}
	}
	return output, err
}
