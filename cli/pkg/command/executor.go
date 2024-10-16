package command

type LanguageExecutor struct {
	executor map[string]Executor
}

func NewDefaultLanguageExecutor() LanguageExecutor {
	return LanguageExecutor{
		executor: map[string]Executor{
			"sh": ShellLanguageExecutor{},
		},
	}
}

type Executor interface {
	GetExecutorCommand() string
	GetExecutorArgs(content string) []string
}

// shell
type ShellLanguageExecutor struct{}

func (e ShellLanguageExecutor) GetExecutorCommand() string {
	return "sh"
}

func (e ShellLanguageExecutor) GetExecutorArgs(content string) []string {
	return formatArgs([]string{"-c", "$"}, content)
}
