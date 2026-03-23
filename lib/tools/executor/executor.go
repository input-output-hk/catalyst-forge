package executor

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/executor.go . Executor
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/wrapped_executor.go . WrappedExecuter

// Executor is an interface for executing commands.
type Executor interface {
	// Execute executes the given command
	Execute(command string, args ...string) ([]byte, error)

	// LookPath searches for an executable named file in the directories named
	// by the PATH environment variable.
	LookPath(file string) (string, error)
}

// WrappedExecuter is an interface for executing commands using a specific
// command.
type WrappedExecuter interface {
	Execute(args ...string) ([]byte, error)
}
