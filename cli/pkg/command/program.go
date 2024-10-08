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
		return fmt.Errorf("command '%s' not found in markdown", cmd)
	}

	return foundCmd.exec(logger)
}

func (cmd *Command) exec(logger *slog.Logger) error {
	if cmd.lang == nil {
		return fmt.Errorf("command block without specified language")
	}

	lang, ok := NewDefaultLanguageExecutor().executor[*cmd.lang]
	if !ok {
		return fmt.Errorf("only commands running with `sh` can be executed")
	}
	if _, err := exec.LookPath(lang.GetExecutorCommand()); err != nil {
		return fmt.Errorf("command '%s' is unavailable", lang.GetExecutorCommand())
	}

	localExec := executor.NewLocalExecutor(
		logger,
		executor.WithRedirect(),
	)
	_, err := localExec.Execute(lang.GetExecutorCommand(), lang.GetExecutorArgs(cmd.content))

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
