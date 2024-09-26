# Targets

The Forge CI system automatically scans and executes a preset number of Earthly targets.
This section describes each of those targets including any special logic associated with it.
Each target includes a brief example using the Go programming language.
This is only done for simplicity as each target can be used to support multiple languages/paradigms.

## Check

The `check` target is used for performing static analysis on a project's code.
This includes operations like validating formatting, linting, or scanning for vulnerable code.
It can also include running analyzers on supporting artifacts of a project, like spellchecking a project's README.

The goal of the `check` target is to provide quick feedback on the validity of a project.
This is why it is the first target that is run in the CI pipeline.
As such, the speed of the `check` target invocation should be prioritized above all else.
If possible, it should avoid building the underlying project or doing any computationally expensive tasks.
The faster the target runs, the quicker developers can get feedback on "easy" fixes within the codebase.

### Example

```earthly
check:
    # Assume we've copied the source code in a previous target
    FROM +src

    # Validate that the code is formatted correctly
    RUN gofmt -l . | grep . && exit 1 || exit 0

    # Run a simple lint on the code
    RUN go vet ./...
```

## Build

The `build` target is used for building the artifacts of a project.
The concept of building is unique to a project and can include anything ranging from compiling a binary, to archiving interpreted
code into a portable artifact, or generating static assets for a web frontend.
In some cases, like with library code, making use of this target does not make sense and it should be skipped.

The primary purpose of the `build` target is to optimize caching.
By forcing the project to build during this target, all future targets which rely on the `build` target will be optimized via
caching.
For example, the `test` target often relies on the project being built prior to running tests.
Since the `test` target occurs after the `build` target it will immediately benefit from the build being cached.

The `build` target should prefer to always produce an artifact, where possible.
Meaning, the target should use `SAVE ARTIFACT` on the resulting build at the end of execution.
Dependent projects will often pull the artifact directly from this target.

### Example

```earthly
build:
    # Assume we've copied the source code in a previous target
    FROM +src

    # Compile our binary
    RUN go build -o bin/program cmd/main.go

    # Save the resulting binary as an artifact
    SAVE ARTIFACT bin/program program
```