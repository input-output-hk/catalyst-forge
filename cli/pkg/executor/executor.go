package executor

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/executor.go . Executor

// Executor is an interface for executing commands.
type Executor interface {
	// Execute executes the given command
	Execute(command string, args ...string) ([]byte, error)
}

// WrappedExecuter is an interface for executing commands using a specific
// command.
type WrappedExecuter interface {
	Execute(args ...string) ([]byte, error)
}
