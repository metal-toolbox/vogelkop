package command

import (
	"errors"
	"fmt"
	"os/exec"
)

var ErrFailedExecution = errors.New("failed execution")

func FailedExecutionError(cmdPath, errMsg string) error {
	return fmt.Errorf("FailedExecution %w : %s \"%s\"", ErrFailedExecution, cmdPath, errMsg)
}

func Call(cmdName string, cmdOptions ...string) (out string, err error) {
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return
	}

	cmd := exec.Command(cmdPath, cmdOptions...)

	outB, err := cmd.CombinedOutput()
	out = string(outB)

	if err != nil {
		err = FailedExecutionError(cmdPath, err.Error())
		return
	}

	return
}
