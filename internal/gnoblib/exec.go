package gnoblib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// ExecOption is the interface for options to customize the command.
type ExecOption interface {
	apply(*cmdOptions)
}

// ExecOptionFunc is an adapter to allow the use of ordinary functions as ExecOption.
type ExecOptionFunc func(*cmdOptions)

func (fn ExecOptionFunc) apply(opts *cmdOptions) {
	fn(opts)
}

// WithStdout sets the standard output for the command.
func (_cmd) WithStdout(stdout io.Writer) ExecOption {
	return ExecOptionFunc(func(opts *cmdOptions) {
		opts.stdout = stdout
	})
}

// WithStdoutJSONDecoder decodes the standard output into the given object.
// The object must be a pointer to a struct.
func (_cmd) WithStdoutJSONDecoder(out any) ExecOption {
	var buf bytes.Buffer
	return ExecOptionFunc(func(opts *cmdOptions) {
		opts.stdout = &buf
		opts.onExit = append(opts.onExit, func() {
			_ = json.Unmarshal(buf.Bytes(), out)
		})
	})
}

// WithStderr sets the standard error for the command.
func (_cmd) WithStderr(stderr io.Writer) ExecOption {
	return ExecOptionFunc(func(opts *cmdOptions) {
		opts.stderr = stderr
	})
}

// WithStdin sets the standard input for the command.
func (_cmd) WithStdin(stdin io.Reader) ExecOption {
	return ExecOptionFunc(func(opts *cmdOptions) {
		opts.stdin = stdin
	})
}

// WithDir sets the working directory for the command.
func (_cmd) WithDir(dir string) ExecOption {
	return ExecOptionFunc(func(opts *cmdOptions) {
		opts.workingDir = dir
	})
}

// WithEnvVars sets environment variables for the command.
func (_cmd) WithEnvVars(env map[string]string) ExecOption {
	return ExecOptionFunc(func(opts *cmdOptions) {
		if opts.envVars == nil {
			opts.envVars = make(map[string]string)
		}
		for k, v := range env {
			opts.envVars[k] = v
		}
	})
}

// WithNoInheritEnv disables inheriting the parent process's environment variables.
func (_cmd) WithNoInheritEnv(b bool) ExecOption {
	return ExecOptionFunc(func(opts *cmdOptions) {
		opts.noInheritEnv = b
	})
}

// ExecOptions combines multiple ExecOption into one.
func (_cmd) ExecOptions(opts ...ExecOption) ExecOption {
	return ExecOptionFunc(func(co *cmdOptions) {
		for _, o := range opts {
			o.apply(co)
		}
	})
}

type cmdOptions struct {
	noInheritEnv bool
	envVars      map[string]string
	workingDir   string
	stdout       io.Writer
	stderr       io.Writer
	stdin        io.Reader
	onExit       []func()
}

type Exec struct {
	prev      *Exec
	ctx       context.Context
	cmd       *exec.Cmd
	stderr    bytes.Buffer
	closers   []io.Closer
	onExit    []func()
	exitCodes []int
}

type _cmd struct {
}

// Exec creates a new command.
// The output of this command can be chained into other commands with Pipe and Pipe2.
// For example:
//
//	Cmd.Exec("echo", "hello").Pipe("wc").Run()
func (c _cmd) Exec(ctx context.Context, command string, args ...string) *Exec {
	return c.ExecOpt(ctx, nil, command, args...)
}

// ExecOpt is like Exec, but you can specify options to customize the command.
// You can also collect multiple options together with ExecOptions.
func (c _cmd) ExecOpt(ctx context.Context, opt ExecOption, command string, args ...string) *Exec {
	var o cmdOptions
	if opt != nil {
		opt.apply(&o)
	}
	execCmd := exec.CommandContext(ctx, command, args...)
	if o.workingDir != "" {
		execCmd.Dir = o.workingDir
	}
	environ := make([]string, 0, len(o.envVars))
	if !o.noInheritEnv {
		environ = append(environ, os.Environ()...)
	}
	if len(o.envVars) > 0 {
		for k, v := range o.envVars {
			environ = append(environ, fmt.Sprintf("%s=%s", k, v))
		}
	}
	execCmd.Stdin = o.stdin
	execCmd.Stdout = o.stdout
	execCmd.Stderr = o.stderr
	execCmd.Env = environ
	return &Exec{
		cmd:    execCmd,
		ctx:    ctx,
		onExit: o.onExit,
	}
}

// Pipe creates a new command that will be executed after the current command.
// The current command's stdout will be piped to the new command's stdin.
func (e *Exec) Pipe(command string, args ...string) *Exec {
	return e.PipeOpt(nil, command, args...)
}

// PipeOpt is like Pipe, but you can specify cmdOptions to customize the command.
func (e *Exec) PipeOpt(opt ExecOption, command string, args ...string) *Exec {
	next := Lib.Cmd.ExecOpt(e.ctx, opt, command, args...)
	if e.cmd.Stdout != nil {
		if c, ok := e.cmd.Stdout.(io.Closer); ok {
			e.closers = append(e.closers, c)
		}
		pr, pw, err := os.Pipe()
		if err != nil {
			panic(err)
		}
		e.closers = append(e.closers, pw)
		tr := io.TeeReader(pr, e.cmd.Stdout)
		e.cmd.Stdout = pw
		next.cmd.Stdin = tr
		next.closers = append(next.closers, pr)
	} else {
		var err error
		next.cmd.Stdin, err = e.cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
	}
	return &Exec{
		prev: e,
		ctx:  e.ctx,
		cmd:  next.cmd,
	}
}

// Pipe2 creates a new command that will be executed after the current command.
// The current command's stderr will be piped to the new command's stdin.
func (e *Exec) Pipe2(command string, args ...string) *Exec {
	return e.Pipe2Opt(nil, command, args...)
}

// Pipe2Opt is like Pipe2, but you can specify cmdOptions to customize the command.
func (e *Exec) Pipe2Opt(opt ExecOption, command string, args ...string) *Exec {
	next := Lib.Cmd.ExecOpt(e.ctx, opt, command, args...)
	if e.cmd.Stderr != nil {
		if c, ok := e.cmd.Stderr.(io.Closer); ok {
			e.closers = append(e.closers, c)
		}
		pr, pw, err := os.Pipe()
		if err != nil {
			panic(err)
		}
		e.closers = append(e.closers, pw)
		tr := io.TeeReader(pr, e.cmd.Stderr)
		e.cmd.Stderr = pw
		next.cmd.Stdin = tr
		next.closers = append(next.closers, pr)
	} else {
		var err error
		next.cmd.Stdin, err = e.cmd.StderrPipe()
		if err != nil {
			panic(err)
		}
	}
	return &Exec{
		prev: e,
		ctx:  e.ctx,
		cmd:  next.cmd,
	}
}

// Run runs the command chain and waits for it to finish.
func (e *Exec) Run() error {
	if err := e.Start(); err != nil {
		return err
	}
	return e.Wait()
}

// Start starts the command chain.
// It returns the first error encountered.
// It does not wait for the command to finish, to wait for the command to finish, use Wait.
func (e *Exec) Start() error {
	this := e
	if this.cmd.Stderr == nil {
		this.cmd.Stderr = &this.stderr
	} else {
		this.cmd.Stderr = io.MultiWriter(&this.stderr, e.cmd.Stderr)
	}
	var chain []*exec.Cmd
	for this != nil {
		chain = append(chain, this.cmd)
		this = this.prev
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if err := chain[i].Start(); err != nil {
			return err
		}
	}
	return nil
}

// Wait waits for the command chain to finish.
// It returns the entire error chain joined by errors.Join.
// To get the exit code of the last command, use ExitCode.
// To get the exit codes of all commands, use ExitCodes.
func (e *Exec) Wait() error {
	this := e
	var chain []*Exec
	for this != nil {
		chain = append(chain, this)
		this = this.prev
	}
	var waitErrs []error
	exitCodes := make([]int, 0, len(chain))
	for i := len(chain) - 1; i >= 0; i-- {
		errCh := make(chan error, 1)
		go func() {
			for _, c := range chain[i].closers {
				_ = c.Close()
			}
			errCh <- chain[i].cmd.Wait()
		}()
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		case err := <-errCh:
			for _, f := range chain[i].onExit {
				f()
			}
			waitErrs = append(waitErrs, err)
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCodes = append(exitCodes, exitErr.ExitCode())
				continue
			}
			exitCodes = append(exitCodes, 0)
		}
	}
	e.exitCodes = exitCodes
	if err := errors.Join(waitErrs...); err != nil {
		return fmt.Errorf("command failed (%v): %w\n%s", e.cmd.Args, err, e.stderr.String())
	}
	return nil
}

// ExitCode returns the exit code of the last command.
func (e *Exec) ExitCode() int {
	if len(e.exitCodes) == 0 {
		return -1
	}
	return e.exitCodes[len(e.exitCodes)-1]
}

// ExitCodes returns the exit codes of all commands in the chain.
func (e *Exec) ExitCodes() []int {
	return e.exitCodes
}
