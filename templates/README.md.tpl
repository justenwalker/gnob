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
{{ includeFile "templates/usage/main.go" }}
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
{{ includeFileRegion "templates/cmdpipe/examples.go" "--- simple command ---" | unindent 1 }}
```

Command with output capture:

```go
{{ includeFileRegion "templates/cmdpipe/examples.go" "--- output capture ---" | unindent 1  }}
```

Command pipeline (equivalent to: echo "hello" | tr '[:lower:]' '[:upper:]' | wc -c):

```go
{{ includeFileRegion "templates/cmdpipe/examples.go" "--- command pipeline ---" | unindent 1  }}
```


JSON processing in pipeline:

```go
{{ includeFileRegion "templates/cmdpipe/examples.go" "--- json processing ---" | unindent 1 }}
```

#### Full Example

```go
{{ includeFile "templates/cmdpipe/main.go" }}
```

### Makefile-Style Targets

While you can use the main function to build your entire project, it is often helpful
to have a Makefile-like target system so that you can create tasks that can be run.

```go
{{ includeFile "templates/makefile/main.go" }}
```

And the result of running the `default` target should be:

```shell
{{ includeFile "templates/makefile/main.txt" }}
```


