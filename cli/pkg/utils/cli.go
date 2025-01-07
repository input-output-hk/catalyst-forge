package utils

import (
	"encoding/json"
	"fmt"
)

// PrintJson prints the given data as a JSON string.
func PrintJson(data interface{}, pretty bool) {
	var out []byte
	var err error

	if pretty {
		out, err = json.MarshalIndent(data, "", "  ")
	} else {
		out, err = json.Marshal(data)
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(out))
}
