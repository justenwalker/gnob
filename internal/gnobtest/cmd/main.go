package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

type flags struct {
	ExitCode  int
	StdOut    string
	StdErr    string
	StdIn2Err bool
	StdIn2Out bool
	Sleep     time.Duration
	PrintEnv  string
}

func main() {
	exit, err := run(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(128)
	}
	os.Exit(exit)
}

func run(ctx context.Context) (int, error) {
	var f flags
	fset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fset.IntVar(&f.ExitCode, "exit", 0, "exit code")
	fset.StringVar(&f.StdOut, "stdout", "", "stdout content")
	fset.StringVar(&f.StdErr, "stderr", "", "stderr content")
	fset.BoolVar(&f.StdIn2Err, "stdin2err", false, "copy stdin to stderr")
	fset.BoolVar(&f.StdIn2Out, "stdin2out", false, "copy stdin to stdout")
	fset.DurationVar(&f.Sleep, "sleep", 0, "sleep for the given duration")
	fset.StringVar(&f.PrintEnv, "printenv", "", "print the environment variables to the given stream (stdout|stderr)")
	if err := fset.Parse(os.Args[1:]); err != nil {
		return 0, fmt.Errorf("parsing flags: %v", err)
	}
	if f.Sleep > 0 {
		time.Sleep(f.Sleep)
	}
	var pipeOut io.Writer
	switch {
	case f.StdIn2Err && f.StdIn2Out: // Pipe Both
		pipeOut = io.MultiWriter(os.Stderr, os.Stdout)
	case f.StdIn2Out && !f.StdIn2Err: // Pipe Out
		pipeOut = os.Stdout
	case f.StdIn2Err && !f.StdIn2Out: // Pipe Err
		pipeOut = os.Stderr
	default:
		pipeOut = io.Discard
	}
	if f.StdOut != "" {
		if _, err := fmt.Fprint(os.Stdout, f.StdOut); err != nil {
			return f.ExitCode, fmt.Errorf("print stdout: %w", err)
		}
	}
	if f.StdErr != "" {
		if _, err := fmt.Fprint(os.Stderr, f.StdErr); err != nil {
			return f.ExitCode, fmt.Errorf("print stderr: %w", err)
		}
	}
	var printEnvOut io.StringWriter
	switch f.PrintEnv {
	case "stdout":
		printEnvOut = os.Stdout
	case "stderr":
		printEnvOut = os.Stderr
	}
	if printEnvOut != nil {
		for _, e := range os.Environ() {
			_, _ = printEnvOut.WriteString(e)
			_, _ = printEnvOut.WriteString("\n")
		}
	}
	if _, err := io.Copy(pipeOut, os.Stdin); err != nil {
		return f.ExitCode, fmt.Errorf("pipe out: %w", err)
	}
	return f.ExitCode, nil
}
