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
	colors       bool
	logger       *slog.Logger
	redirect     bool
	stdoutStream io.Writer
	stderrStream io.Writer
	workdir      string
}

// NewLocalExecutor creates a new LocalExecutor with the given options.
func NewLocalExecutor(logger *slog.Logger, options ...LocalExecutorOption) *LocalExecutor {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	e := &LocalExecutor{
		logger:       logger,
		stdoutStream: os.Stdout,
		stderrStream: os.Stderr,
	}

	for _, option := range options {
		option(e)
	}

	return e
}

// Execute runs a command with the given arguments and returns the combined output.
func (e *LocalExecutor) Execute(command string, args ...string) ([]byte, error) {
	cmd := e.prepareCommand(command, args...)

	e.logger.Debug("Executing local command",
		"command", cmd.String(),
		"workdir", cmd.Dir,
		"redirect", e.redirect)

	if e.redirect {
		return e.executeWithRedirect(cmd)
	}

	return cmd.CombinedOutput()
}

// LookPath searches for an executable named file in the directories named by the PATH environment variable.
func (e *LocalExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// prepareCommand creates and configures the exec.Cmd instance.
func (e *LocalExecutor) prepareCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)

	if e.workdir != "" {
		cmd.Dir = e.workdir
	}

	if e.colors {
		cmd.Env = append(os.Environ(), e.getColorEnvVars()...)
	}

	return cmd
}

// executeWithRedirect runs the command while capturing and redirecting output.
func (e *LocalExecutor) executeWithRedirect(cmd *exec.Cmd) ([]byte, error) {
	// Buffer to capture all output
	var captureBuffer bytes.Buffer

	// Set up pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Copy output concurrently
	errChan := make(chan error, 2)

	go e.copyOutput(stdoutPipe, e.stdoutStream, &captureBuffer, errChan)
	go e.copyOutput(stderrPipe, e.stderrStream, &captureBuffer, errChan)

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Check for copy errors
	for i := 0; i < 2; i++ {
		if copyErr := <-errChan; copyErr != nil {
			return captureBuffer.Bytes(), copyErr
		}
	}

	return captureBuffer.Bytes(), cmdErr
}

// copyOutput copies data from reader to both the stream and capture buffer.
func (e *LocalExecutor) copyOutput(reader io.Reader, stream io.Writer, capture *bytes.Buffer, errChan chan<- error) {
	// Create a multi-writer that writes to both the stream and capture buffer
	writer := io.MultiWriter(stream, capture)
	_, err := io.Copy(writer, reader)
	errChan <- err
}

// getColorEnvVars returns environment variables that encourage color output
func (e *LocalExecutor) getColorEnvVars() []string {
	return []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"FORCE_COLOR=1",
		"CLICOLOR=1",
		"CLICOLOR_FORCE=1",
		// Git specific
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=color.ui",
		"GIT_CONFIG_VALUE_0=always",
	}
}

// WithColors configures the LocalExecutor to enable colors.
func WithColors() LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.colors = true
	}
}

// WithRedirect configures the LocalExecutor to redirect stdout and stderr
// to the local process's stdout and stderr while also capturing the output.
func WithRedirect() LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.redirect = true
	}
}

// WithRedirectTo configures the LocalExecutor to redirect stdout and stderr
// to the given writers while also capturing the output.
func WithRedirectTo(stdout, stderr io.Writer) LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.redirect = true
		if stdout != nil {
			e.stdoutStream = stdout
		}
		if stderr != nil {
			e.stderrStream = stderr
		}
	}
}

// WithWorkdir configures the LocalExecutor to run commands in
// the given working directory.
func WithWorkdir(workdir string) LocalExecutorOption {
	return func(e *LocalExecutor) {
		e.workdir = workdir
	}
}

// WrappedLocalExecutor wraps an Executor with a specific command,
// allowing for easier repeated execution of the same command with different arguments.
type WrappedLocalExecutor struct {
	Executor
	command string
}

// NewWrappedLocalExecutor creates a new WrappedLocalExecutor that always executes
// the specified command using the provided Executor.
func NewWrappedLocalExecutor(e Executor, command string) *WrappedLocalExecutor {
	return &WrappedLocalExecutor{
		Executor: e,
		command:  command,
	}
}

// Execute runs the wrapped command with the given arguments.
func (e *WrappedLocalExecutor) Execute(args ...string) ([]byte, error) {
	return e.Executor.Execute(e.command, args...)
}
