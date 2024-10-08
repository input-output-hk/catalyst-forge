package command

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
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

func (cmd *Command) Exec() error {
	executorCmd, executorArgs := getLangExecutor(cmd.lang)
	if executorCmd == "" {
		return fmt.Errorf("only commands running with `sh` can be executed")
	}

	// check if the command is available
	if _, err := exec.LookPath(executorCmd); err != nil {
		return fmt.Errorf("command '%s' not found in PATH", executorCmd)
	}

	// start executing the command
	execCmd := exec.Command(executorCmd, formatArgs(executorArgs, cmd.content)...)

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := execCmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("error reading output:", err)
	}

	if err := execCmd.Wait(); err != nil {
		fmt.Println("error waiting for command:", err)
	}

	return nil
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

func (prog *Program) ProcessCmd(cmd string) error {
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

	return foundCmd.Exec()
}
