package command

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

var ErrFailedExecution = errors.New("failed execution")

func FailedExecutionError(cmdPath, errMsg string) error {
	return fmt.Errorf("FailedExecution %w : %s \"%s\"", ErrFailedExecution, cmdPath, errMsg)
}

func Call(ctx context.Context, cmdName string, cmdOptions ...string) (out string, err error) {
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return
	}

	cmd := exec.CommandContext(ctx, cmdPath, cmdOptions...)

	outB, err := cmd.CombinedOutput()
	out = string(outB)

	if err != nil {
		err = FailedExecutionError(cmdPath, err.Error())
		return
	}

	return
}
