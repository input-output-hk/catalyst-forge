# Getting Started

In this tutorial, we will create a sample project that will be automatically validated, built, and published by Catalyst Forge.
During this process, we will encounter the various tools and configurations that power Catalyst Forge.
By the end of this tutorial, you will be equipped to begin developing your own projects.

We will be building a trivial program in the Go language in order to facilitate learning.
Understanding the Go language is not necessary, and more adventerous learners may choose to substitute the code with their language
of choice.
Catalyst Forge is language agnostic; all that is required is knowing how to build and validate your language of choice.

## Pre-requisites

!!! note

    External contributors will only be able to partially complete this tutorial.
    This is due to the fact that permissions on most repositories (including the playground) do not allow external contributors to
    arbitrarily merge code.
    If you're an external contributor, feel free to follow the tutorial up to the point where merging is required.

To improve the learning experience, it's recommended you work within an existing repository that has Catalyst Forge configured.
The [Catalyst Forge Playground](https://github.com/input-output-hk/catalyst-forge-playground) is one such repo that is free to be
used for learning and experimentation.
Prior to beginning this tutorial, clone the repository to your local system and open it up with the editor of your choice.

As mentioned in the introduction, this tutorial uses the Go programming language for demonstration purposes.
To get the best experience, it's recommended you install the latest version of Go on your local system.
If you plan to substitue your own language, keep in mind you will need to modify certain steps of the tutorial to match the language
of choice.

## Source

We will begin by creating a uniquely named subfolder within the root of the repository. From the root, execute the following
command:

```shell
$ mkdir my_project && cd my_project
```

Feel free to substitue the folder name with something different.
Inside of this folder, we will initiate a Go module:

```shell
$ go mod init my_project
go: creating new go.mod: module my_project
```

Then we will add a simple `main.go` to the directory:

```go
package main

import (
    "fmt"
    "os"
)

func greet(name string) string {
    return fmt.Sprintf("Hello, %s!", name)
}

func main() {
    fmt.Println(greet(os.Args[1]))
}
```

We can test everything is working by running the program:

```shell
$ go run main.go john
Hello, john!
```

Finally, we will add a trivial test in `main_test.go`:

```go
package main

import "testing"

func Test_greet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Test 1",
			input:    "Alice",
			expected: "Hello, Alice!",
		},
		{
			name:     "Test 2",
			input:    "Bob",
			expected: "Hello, Bob!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := greet(tt.input)
			if actual != tt.expected {
				t.Errorf("greet(%s): expected %s, actual %s", tt.input, tt.expected, actual)
			}
		})
	}
}
```

We can validate the test is passing with:

```shell
$ go test .
ok      my_project      0.001s
```

This is the extent of the Go code that we will use for this tutorial.
The remainder of the tutorial will demonstrate how to test, build, and publish our program.