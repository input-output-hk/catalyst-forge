package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"forge":   Run,
		"earthly": mockEarthly,
	}))
}

func TestValidate(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/validate",
	})
}
func TestRun(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/run",
	})
}

func TestScan(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/scan",
	})
}

func TestDevX(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/devx",
	})
}

func mockEarthly() int {
	for _, arg := range os.Args {
		fmt.Println(arg)
	}

	secrets := os.Getenv("EARTHLY_SECRETS")
	if secrets != "" {
		fmt.Println("EARTHLY_SECRETS=" + secrets)
	}

	stdout, err := os.ReadFile("earthly_stdout.txt")
	if err == nil {
		fmt.Println(string(stdout))
	}

	return 0
}
