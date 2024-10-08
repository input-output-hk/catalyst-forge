package command

import (
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
)

type Program struct {
	name   string
	groups []CommandGroup
}

type CommandGroup struct {
	name     string
	commands []Command
}

type Command struct {
	content  string
	lang     *string
	platform *string
}

func (prog *Program) ProcessCmd(cmd string, logger *slog.Logger) error {
	var foundCmd *Command
	for _, v := range prog.groups {
		if v.GetId() == cmd {
			// TODO: should get the fisrt (most specified) command corresponding to the current host platform
			foundCmd = &v.commands[0]
		}
	}

	if foundCmd == nil {
		return fmt.Errorf("command not found")
	}

	return foundCmd.exec(logger)
}

func (cmd *Command) exec(logger *slog.Logger) error {
	executorCmd, executorArgs := getLangExecutor(cmd.lang)
	if executorCmd == "" {
		return fmt.Errorf("only commands running with `sh` can be executed")
	}

	if _, err := exec.LookPath(executorCmd); err != nil {
		return fmt.Errorf("command '%s' not found in PATH", executorCmd)
	}

	localExec := executor.NewLocalExecutor(
		logger,
		executor.WithRedirect(),
	)
	_, err := localExec.Execute(executorCmd, formatArgs(executorArgs, cmd.content))

	return err
}

func (cg *CommandGroup) GetId() string {
	var result []rune

	for _, char := range cg.name {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			result = append(result, unicode.ToLower(char))
		} else if unicode.IsSpace(char) {
			result = append(result, '-')
		}
	}

	joined := string(result)

	re := regexp.MustCompile(`-+`)
	joined = re.ReplaceAllString(joined, "-")

	return strings.Trim(joined, "-")
}
