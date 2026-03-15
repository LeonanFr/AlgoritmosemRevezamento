package executor

import (
	"context"
	"os/exec"
)

func compile(ctx context.Context, lang Language, workDir string, codePath string) ([]byte, error) {
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
	return cmd.CombinedOutput()
}
