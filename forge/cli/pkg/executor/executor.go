package executor

//go:generate go run github.com/matryer/moq@latest -out executor_mock.go . Executor

// Executor is an interface for executing commands.
type Executor interface {
	// Execute executes the given command
	Execute(command string, args []string) ([]byte, error)
}
