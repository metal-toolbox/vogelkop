package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	zaphook "github.com/Sytten/logrus-zap-hook"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
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

type contextKey string

var contextLoggerKey = contextKey("logger")

func NewContextWithLogger(existingCtx context.Context, l *zap.SugaredLogger) context.Context {
	ctx := context.WithValue(existingCtx, contextLoggerKey, l)
	return ctx
}

func LoggerValueFromContext(ctx context.Context) *zap.SugaredLogger {
	logger, _ := ctx.Value(contextLoggerKey).(*zap.SugaredLogger)
	return logger
}

// ZapToLogrus takes a context and converts the zap.SugaredLogger available
// within the context as "logger" to a logrus logger and returns it.
func ZapToLogrus(ctx context.Context) (ll *logrus.Logger, err error) {
	ll = logrus.New()
	ll.ReportCaller = true
	ll.SetOutput(io.Discard)
	sugaredLogger := LoggerValueFromContext(ctx)
	zapLogger := sugaredLogger.Desugar()
	hook, err := zaphook.NewZapHook(zapLogger)
	if err != nil {
		return nil, err
	}

	ll.Hooks.Add(hook)

	return
}
