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
	logger       *slog.Logger
	redirect     bool
	stderrStream io.Writer
	stdoutStream io.Writer
	workdir      string
}

func (e *LocalExecutor) Execute(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)

	if e.workdir != "" {
		cmd.Dir = e.workdir
	}

	e.logger.Debug("Executing local command", "command", cmd.String(), "workdir", cmd.Dir)
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

		stdoutWriter := e.stdoutStream
		if stdoutWriter == nil {
			stdoutWriter = os.Stdout
		}
		stdoutWriter = io.MultiWriter(stdoutWriter, &buffer)

		stderrWriter := e.stderrStream
		if stderrWriter == nil {
			stderrWriter = os.Stderr
		}
		stderrWriter = io.MultiWriter(stderrWriter, &buffer)

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
			return buffer.Bytes(), err
		}

		if err := <-errChan; err != nil {
			return buffer.Bytes(), err
		}

		return buffer.Bytes(), nil
	}

	return cmd.CombinedOutput()
}

func (e *LocalExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// WithRedirect is an option that configures the LocalExecutor to redirect the
// stdout and stderr of the commands to the stdout and stderr of the local
// process.
func WithRedirect() LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.redirect = true
	}
}

// WithRedirectTo is an option that configures the LocalExecutor to redirect the
// stdout and stderr of the commands to the given writers.
func WithRedirectTo(stdout, stderr io.Writer) LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.redirect = true
		e.stdoutStream = stdout
		e.stderrStream = stderr
	}
}

// WithWorkdir is an option that configures the LocalExecutor to run commands in
// the given working directory.
func WithWorkdir(workdir string) LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.workdir = workdir
	}
}

type WrappedLocalExecutor struct {
	Executor
	command string
}

func (e WrappedLocalExecutor) Execute(args ...string) ([]byte, error) {
	return e.Executor.Execute(e.command, args...)
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

func NewLocalWrappedExecutor(e Executor, command string) WrappedLocalExecutor {
	return WrappedLocalExecutor{
		Executor: e,
		command:  command,
	}
}
