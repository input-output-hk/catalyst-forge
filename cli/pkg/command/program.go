package command

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
)

type program struct {
	name   string
	groups []commandGroup
}

type commandGroup struct {
	name     string
	commands []command
}

type command struct {
	content  string
	lang     *string
	platform *string
}

func (cmd *command) Exec() error {
	executorCmd, executorArgs := GetLangExecutor(cmd.lang)
	if executorCmd == "" {
		return fmt.Errorf("only commands running with `sh` can be executed")
	}

	// check if the command is available
	if _, err := exec.LookPath(executorCmd); err != nil {
		return fmt.Errorf("command '%s' not found in PATH", executorCmd)
	}

	// start executing the command
	execCmd := exec.Command(executorCmd, FormatArgs(executorArgs, cmd.content)...)

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

func (cg *commandGroup) GetId() string {
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
