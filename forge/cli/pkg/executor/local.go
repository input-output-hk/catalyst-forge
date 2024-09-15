package executor

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
)

// LocalExecutorOption is an option for configuring a LocalExecutor.
type LocalExecutorOption func(e *LocalExecutor)

// LocalExecutor is an Executor that runs commands locally.
type LocalExecutor struct {
	logger   *slog.Logger
	redirect bool
}

func (e *LocalExecutor) Execute(command string, args []string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	e.logger.Debug("Executing local command", "command", cmd.String())

	if e.redirect {
		var buffer bytes.Buffer
		errChan := make(chan error, 2)

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return nil, err
		}

		stdoutWriter := io.MultiWriter(os.Stdout, &buffer)
		stderrWriter := io.MultiWriter(os.Stderr, &buffer)

		if err := cmd.Start(); err != nil {
			return nil, err
		}

		go func() {
			_, err := io.Copy(stdoutWriter, stdoutPipe)
			errChan <- err
		}()

		go func() {
			_, err := io.Copy(stderrWriter, stderrPipe)
			errChan <- err
		}()

		if err := cmd.Wait(); err != nil {
			return nil, err
		}

		if err := <-errChan; err != nil {
			return nil, err
		}

		return buffer.Bytes(), nil
	}

	return cmd.CombinedOutput()
}

// WithRedirect is an option that configures the LocalExecutor to redirect the
// stdout and stderr of the commands to the stdout and stderr of the local
// process.
func WithRedirect() LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.redirect = true
	}
}

// NewLocalExecutor creates a new LocalExecutor with the given options.
func NewLocalExecutor(logger *slog.Logger, options ...LocalExecutorOption) *LocalExecutor {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	e := &LocalExecutor{
		logger: logger,
	}

	for _, option := range options {
		option(e)
	}

	return e
}
