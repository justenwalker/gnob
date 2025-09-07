# Go No-Build Tool

## Overview

`gnob` is a lightweight build system that exists as a single Go file.
It allows you to orchestrate your builds using only the `go` command, and nothing more.

### Philosophy

`gnob` addresses a common problem in Go projects: the need for a build tool that doesn't
require additional installations beyond the Go toolchain itself. Unlike Make, Bazel,
or Mage, gnob requires no external dependencies.

### Key Features

  - Single-file distribution: Copy [gnob.go](gnob.go) into your project, no installation required
  - Self-rebuilding: Automatically rebuilds itself when source files change
  - Command chaining: Build complex command pipelines with proper error handling
  - Makefile-style targets: Create reusable build tasks with dependencies
  - JSON processing: Built-in support for processing JSON data in pipelines
  - Structured logging: Integrated logging for build process visibility

### Prior Art

This project is inspired primarily by [nob.h](https://github.com/tsoding/nob.h)
And also [Mage](https://magefile.org/).

## Installation

First, copy [gnob.go](gnob.go) into your project.
It's a good idea to put this into a `build/` subfolder
so it doesn't interact with the rest of your program.

**Example:**
```
.
|-- build/
|   |-- gnob.go
|   |-- main.go
|-- main.go
|-- ...
```

You can then create any number of files in the `build/` folder, using the `.go` extension.

## Usage

To use `gnob.go`, you should create `main.go` along side `gnob.go` with the follow skeleton:

```go
//go:build gnob

package main

//go:generate go build -o gnob -tags gnob ./...

// Most functionality is available through member objects of GnobLib
// This allows the functions to be isolated in a single namespace.
//
// You can create some convenience variable to alias these fields.
var gnob = GnobLib.Main

// The GnobLib.Main.GoRebuildYourself function automatically rebuilds the
//gnob binary when source files are newer than the executable:

func main() {
	// Rebuild if any *.go files are newer than the gnob binary.
	gnob.GoRebuildYourself("*.go")

	// Rebuild only if specific files are newer than the gnob binary.
	gnob.GoRebuildYourself("main.go", "build.go", "tasks.go")

	// Your build logic here
}

```

The `GnobLib.Main.GoRebuildYourself` function will rebuild your project if any of the files matching source are newer than the `gnob` binary.

To bootstrap the gnob binary, run

```sh
go generate -C ./build .
```

From now on, you can run `./build/gnob` and it will rebuild the `./build/gnob` binary if necessary before it runs your build tasks.

## Build Helpers

### Commands and Pipes

While you can already use the `exec` package to run external commands,
It is often helpful to have a more convenient way to run commands and pipes.

`gnob` provides a way to construct a chain of commands that pipe the outputs
of one command into the inputs of the next, and execute them all at once.

#### Command Execution Examples

Simple command execution:

```go
if err := GnobLib.Cmd.Exec(ctx, "go", "build", "./...").Run(); err != nil {
	GnobLogger.Error("failed to build", "error", err)
	return err
}

```

Command with output capture:

```go
var output bytes.Buffer
if err := GnobLib.Cmd.ExecOpt(ctx, GnobLib.Cmd.ExecOptions(
	GnobLib.Cmd.WithStdout(&output),
), "git", "rev-parse", "HEAD").Run(); err != nil {
	return err
}
commitHash := strings.TrimSpace(output.String())
GnobLogger.Info("commit hash", "hash", commitHash)

```

Command pipeline (equivalent to: echo "hello" | tr '[:lower:]' '[:upper:]' | wc -c):

```go
var result bytes.Buffer
p := GnobLib.Cmd.Exec(ctx, "echo", "hello")
p = p.Pipe("tr", "[:lower:]", "[:upper:]")
p = p.PipeOpt(GnobLib.Cmd.WithStdout(&result), "wc", "-c")
if err := p.Run(); err != nil {
	return err
}
GnobLogger.Info("result", "result", result.String())

```


JSON processing in pipeline:

```go
type Config struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
var config Config
if err := GnobLib.Cmd.ExecOpt(ctx, GnobLib.Cmd.WithStdoutJSONDecoder(&config),
	"cat", "package.json").Run(); err != nil {
}
GnobLogger.Info("config",
	"name", config.Name,
	"version", config.Version,
)

```

#### Full Example

```go
//go:build gnob

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

//go:generate go build -o gnob -tags gnob ./...

// convenience variables
var (
	logger = GnobLogger
	cmd    = GnobLib.Cmd
)

type MyObject struct {
	Msg string `json:"msg"`
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var (
		obj MyObject
		buf bytes.Buffer
	)

	// The following is essentially:
	// echo '{"msg":"hello world"} | cat | jq -r .msg
	p := cmd.ExecOpt(ctx,
		// Here we are tapping the output of the first command
		// and unmarshalling it into our 'out' object.
		// This is tougher to do in bash alone.
		cmd.WithStdoutJSONDecoder(&obj),
		"bash", "-c", `echo '{"msg":"hello world"}'`)
	p = p.Pipe("cat")
	p = p.PipeOpt(cmd.WithStdout(&buf),
		"jq", "-r", ".msg")
	if err := p.Run(); err != nil {
		return err
	}
	logger.Info("writing buffer.out", "msg", strings.TrimSpace(buf.String()))
	if err := os.WriteFile("buffer.out", buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write buffer.out: %w", err)
	}

	logger.Info("marshalling json object", "obj", obj)
	outJS, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	logger.Info("writing json", "js", string(outJS))
	if err = os.WriteFile("buffer.json", outJS, 0o644); err != nil {
		return fmt.Errorf("write buffer.json: %w", err)
	}
	return nil
}

```

### Makefile-Style Targets

While you can use the main function to build your entire project, it is often helpful
to have a Makefile-like target system so that you can create tasks that can be run.

```go
//go:build gnob

package main

import (
	"context"
	"fmt"
)

//go:generate go build -o gnob -tags gnob ./...

func main() {
	GnobLib.Main.GoRebuildYourself("*.go")
	mf := GnobLib.Makefile.New(Default, TaskOne, TaskTwo, TaskThree)
	mf.Run(context.Background())
}

var Default = GnobMakeTarget{
	Name:     "default",
	Desc:     "default target",
	LongDesc: "This is the default target",
	Default:  true,
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		// Run dependencies first
		if err := mf.Depend(ctx, "task1", "task2", "task3"); err != nil {
			return err
		}
		// Now perform the default action
		fmt.Println("Default Target")
		return nil
	},
}

var TaskOne = GnobMakeTarget{
	Name:     "task1",
	Desc:     "Task1",
	LongDesc: "This is the first task",
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		fmt.Println("Task #1")
		return nil
	},
}
var TaskTwo = GnobMakeTarget{
	Name:     "task2",
	Desc:     "Task2",
	LongDesc: "This is the second task",
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		fmt.Println("Task #2")
		return nil
	},
}
var TaskThree = GnobMakeTarget{
	Name:     "task3",
	Desc:     "Task3",
	LongDesc: "This is the third task",
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		fmt.Println("Task #3")
		return nil
	},
}

```

And the result of running the `default` target should be:

```shell
$ ./gnob -help ## Listing Targets
Usage: gnob [-help] [target]
Targets:
* default   default target
  task1     Task1
  task2     Task2
  task3     Task3

* (default target)

$ ./gnob -help default ## Help on Target 'default'
gnob default:

default target
This is the default target

$ ./gnob default ## Run 'default'
Task #1
Task #2
Task #3
Default Target
```


