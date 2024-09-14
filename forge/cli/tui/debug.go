package tui

import "os"

func MakeDebugFile() (*os.File, error) {
	f, err := os.Create("debug.log")
	if err != nil {
		return nil, err
	}
	return f, nil
}
