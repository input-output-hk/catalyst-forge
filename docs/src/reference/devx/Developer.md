# Example Developer.md file

This is an example of a Developer.md file.
It is used to help explain, informally, the structure of Developer.md files.

First level headings are just documentation.

## Second level headings are command names

They are normalized, so this command would become: `second-level-headings-are-command-names`.

Only the first matching code block that has a recognized type is used.

This command uses `sh` which means, run the command in the shell of the caller.
Because it's not possible to know exactly what shell the caller is using,
these kinds of commands should be simple and not have any logic.

```sh
echo "This is a command run in the shell of the caller"
```

## A command that uses bash

To ensure that a known shell is used, `bash` can be used to ensure the command is run inside a `bash` shell.
The system must have `bash` installed, or it will fail.

Scripts are run together, not as a distinct set of commands.
So it's easy to do loops or multi-line statements without using backslash.
Comments can be used to better explain the script.

```bash
for i in 1 2 3; do
    echo "Executing command $i"
    # Pause for half a second to make it easier to see the output.
    sleep 0.5 
done
```

## A command that uses python 3

Currently, only `sh`, `bash` and `python` are intended to be supported.

This would never get executed.
Its just documentation.
The `python` script below could be written in `rust` like so:

```rust
use std::thread;
use std::time::Duration;

fn main() {
    for i in 1..=3 {
        println!("Executing command {}", i);
        // Pause for half a second to make it easier to see the output.
        thread::sleep(Duration::from_secs_f64(0.5));
    }
}
```

Similar to bash, commands can be run inside a python interpreter.
The system must have `python` installed, or it will fail.
These scripts should restrict themselves to the python standard library.

```python
import time
print("This is a command run inside python")

for i in range(1, 4):
    print("Executing command", i)
    # Pause for half a second to make it easier to see the output.
    time.sleep(0.5)   
```

The list of supported interpreted/scripting languages could grow.
It will never include complied languages like C, Rust, Go, etc.

## What about parameters

Currently, parameters are not defined.
However, Environment variables will be passed through from the caller to a command.

Environment variables can be used to parameterize any command.
If they are, they should be documented in the Developer.md file with the command.

This command will show all the current caller's environment variables.

```python
import os

for key, value in os.environ.items():
    print(f"Key: {key}, Value: {value}")
```

## System-specific commands

Sometimes different systems require different commands.
This can be accommodated by placing a `platforms` table before a command.

IF the current platform matches one in the list, the command is used.
Otherwise, it is skipped as being documentation.

The ***FIRST*** command to match the users platform will run.
All others are ignored.
Therefore, specific platforms should be listed first.

### Linux/Mac on ARM

| Platforms |
| --- |
| linux/aarch64 |
| darwin/aarch64 |

This only is executed on Linux and Mac if the CPU is ARM-Based.

If we had specified `linux` or `darwin` by themselves, then the CPU type would not matter for that platform.

If we specified `aarch64` by itself, then any platform using an Arm processor would match.

```sh
echo "This only runs on Linux or Mac if the CPU is ARM Based."
```

### That somewhat popular OS from Redmond

| Platforms |
| --- |
| windows |

This is executed on all variants of Windows.

```sh
@echo off
echo This only runs on Windows.
```

### Default

Third level headings are just documentation, they have no special meaning.

But in this case, they can be used to break up the platforms to make the intention clearer.
This would still execute the same without the third level headings.

As this is the last command block, if none of the above executed, it will execute.

It has to be listed last.
Otherwise it will match first and none of the platform-specific commands will run.

```sh
echo "This will run on all other platforms."
```
