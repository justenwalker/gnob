package gnobtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/justenwalker/gnob/internal/gnoblib"
)

//go:generate go build ./cmd/main.go

func mainExec(t *testing.T) string {
	t.Helper()
	exe := "./main"
	if runtime.GOOS == "windows" {
		exe = ".\\main.exe"
	}
	_, err := os.Stat(exe)
	if os.IsNotExist(err) {
		t.Skipf("%q not found. run 'go generate' to regenerate the executable.", exe)
	}
	if err != nil {
		t.Fatalf("could not stat %q: %v", exe, err)
	}
	return exe
}

func TestExec(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exe := mainExec(t)
	c := gnoblib.Lib.Cmd.Exec(t.Context(), exe,
		"-exit", "0",
		"-stdout", "hello world",
	)
	c = c.Pipe(exe,
		"-exit", "0",
		"-stdin2out",
		"-stdout", "pipe1:",
		"-stderr", "pipe1:",
	)
	c = c.PipeOpt(gnoblib.Lib.Cmd.ExecOptions(
		gnoblib.Lib.Cmd.WithStdout(&stdout),
		gnoblib.Lib.Cmd.WithStderr(&stderr),
	), exe,
		"-exit", "0",
		"-stdin2out",
		"-stdout", "pipe2:",
		"-stderr", "pipe2:",
	)
	if err := c.Run(); err != nil {
		t.Fatalf("error: %v", err)
	}
	t.Logf("stdout: %s", stdout.String())
	t.Logf("stderr: %s", stderr.String())
}

func TestExecBasic(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  bool
	}{
		{
			name:     "exit_success",
			args:     []string{"-exit", "0"},
			wantCode: 0,
			wantErr:  false,
		},
		{
			name:     "exit_failure",
			args:     []string{"-exit", "1"},
			wantCode: 1,
			wantErr:  true,
		},
		{
			name:     "exit_high_code",
			args:     []string{"-exit", "42"},
			wantCode: 42,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gnoblib.Lib.Cmd.Exec(t.Context(), exe, tt.args...)
			err := c.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecWithOutput(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name         string
		args         []string
		wantStdout   string
		wantStderr   string
		wantContains bool
	}{
		{
			name:       "stdout_only",
			args:       []string{"-exit", "0", "-stdout", "hello stdout"},
			wantStdout: "hello stdout",
			wantStderr: "",
		},
		{
			name:       "stderr_only",
			args:       []string{"-exit", "0", "-stderr", "hello stderr"},
			wantStdout: "",
			wantStderr: "hello stderr",
		},
		{
			name:       "both_outputs",
			args:       []string{"-exit", "0", "-stdout", "out", "-stderr", "err"},
			wantStdout: "out",
			wantStderr: "err",
		},
		{
			name:       "empty_outputs",
			args:       []string{"-exit", "0"},
			wantStdout: "",
			wantStderr: "",
		},
		{
			name:         "multiline_output",
			args:         []string{"-exit", "0", "-stdout", "line1\nline2\nline3"},
			wantStdout:   "line1\nline2\nline3",
			wantContains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
				gnoblib.Lib.Cmd.WithStdout(&stdout),
				gnoblib.Lib.Cmd.WithStderr(&stderr),
			), exe, tt.args...)
			err := c.Run()
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			gotStdout := stdout.String()
			gotStderr := stderr.String()

			if tt.wantContains {
				if !strings.Contains(gotStdout, tt.wantStdout) {
					t.Errorf("stdout = %q, want to contain %q", gotStdout, tt.wantStdout)
				}
			} else {
				if gotStdout != tt.wantStdout {
					t.Errorf("stdout = %q, want %q", gotStdout, tt.wantStdout)
				}
			}

			if gotStderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", gotStderr, tt.wantStderr)
			}
		})
	}
}

func TestExecPipeStdout(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name       string
		stage1Args []string
		stage2Args []string
		wantOutput string
	}{
		{
			name:       "simple_pipe",
			stage1Args: []string{"-exit", "0", "-stdout", "hello"},
			stage2Args: []string{"-exit", "0", "-stdin2out"},
			wantOutput: "hello",
		},
		{
			name:       "pipe_with_prefix",
			stage1Args: []string{"-exit", "0", "-stdout", "data"},
			stage2Args: []string{"-exit", "0", "-stdin2out", "-stdout", "prefix:"},
			wantOutput: "prefix:data",
		},
		{
			name:       "empty_pipe",
			stage1Args: []string{"-exit", "0"},
			stage2Args: []string{"-exit", "0", "-stdin2out"},
			wantOutput: "",
		},
		{
			name:       "multiline_pipe",
			stage1Args: []string{"-exit", "0", "-stdout", "line1\nline2\nline3"},
			stage2Args: []string{"-exit", "0", "-stdin2out", "-stdout", "prefix:"},
			wantOutput: "prefix:line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			c1 := gnoblib.Lib.Cmd.Exec(t.Context(), exe, tt.stage1Args...)
			c2 := c1.PipeOpt(gnoblib.Lib.Cmd.ExecOptions(
				gnoblib.Lib.Cmd.WithStdout(&stdout),
			), exe, tt.stage2Args...)

			err := c2.Run()
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			got := stdout.String()
			if got != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestExecPipeStderr(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name       string
		stage1Args []string
		stage2Args []string
		wantOutput string
	}{
		{
			name:       "stderr_pipe",
			stage1Args: []string{"-exit", "0", "-stderr", "error message"},
			stage2Args: []string{"-exit", "0", "-stdin2out"},
			wantOutput: "error message",
		},
		{
			name:       "stderr_pipe_with_prefix",
			stage1Args: []string{"-exit", "0", "-stderr", "error"},
			stage2Args: []string{"-exit", "0", "-stdin2out", "-stdout", "ERROR:"},
			wantOutput: "ERROR:error",
		},
		{
			name:       "empty_stderr_pipe",
			stage1Args: []string{"-exit", "0"},
			stage2Args: []string{"-exit", "0", "-stdin2out"},
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			c1 := gnoblib.Lib.Cmd.Exec(t.Context(), exe, tt.stage1Args...)
			c2 := c1.Pipe2Opt(gnoblib.Lib.Cmd.ExecOptions(
				gnoblib.Lib.Cmd.WithStdout(&stdout),
			), exe, tt.stage2Args...)

			err := c2.Run()
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			got := stdout.String()
			if got != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestExecMultiplePipes(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name       string
		stages     [][]string
		wantOutput string
	}{
		{
			name: "three_stage_pipe",
			stages: [][]string{
				{"-exit", "0", "-stdout", "start"},
				{"-exit", "0", "-stdin2out", "-stdout", "middle:"},
				{"-exit", "0", "-stdin2out", "-stdout", "end:"},
			},
			wantOutput: "end:middle:start",
		},
		{
			name: "five_stage_pipe",
			stages: [][]string{
				{"-exit", "0", "-stdout", "1"},
				{"-exit", "0", "-stdin2out", "-stdout", "2"},
				{"-exit", "0", "-stdin2out", "-stdout", "3"},
				{"-exit", "0", "-stdin2out", "-stdout", "4"},
				{"-exit", "0", "-stdin2out", "-stdout", "5"},
			},
			wantOutput: "54321",
		},
		{
			name: "alternating_stdout_stderr",
			stages: [][]string{
				{"-exit", "0", "-stdout", "data1", "-stderr", "error1"},
				{"-exit", "0", "-stdin2out", "-stdout", "out:", "-stderr", "err:"},
			},
			wantOutput: "out:data1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			c := gnoblib.Lib.Cmd.Exec(t.Context(), exe, tt.stages[0]...)
			for i := 1; i < len(tt.stages); i++ {
				var opts gnoblib.ExecOption
				if i == len(tt.stages)-1 {
					opts = gnoblib.Lib.Cmd.WithStdout(&stdout)
				}
				c = c.PipeOpt(opts, exe, tt.stages[i]...)
			}

			err := c.Run()
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			got := stdout.String()
			if got != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestExecWithTeeOutput(t *testing.T) {
	exe := mainExec(t)
	// Test that when stdout is already set, Pipe creates a TeeReader
	var stage1Out, stage2Out, stage3Out bytes.Buffer

	c1 := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.WithStdout(&stage1Out),
		exe, "-exit", "0", "-stdout", "tee test")

	c2 := c1.PipeOpt(gnoblib.Lib.Cmd.WithStdout(&stage2Out),
		exe, "-exit", "0", "-stdin2out", "-stdout", "piped:")

	c3 := c2.PipeOpt(gnoblib.Lib.Cmd.WithStdout(&stage3Out),
		exe, "-exit", "0", "-stdin2out", "-stdout", "piped2:")

	err := c3.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// First stage output should be captured
	if stage1Out.String() != "tee test" {
		t.Errorf("stage1 stdout = %q, want %q", stage1Out.String(), "tee test")
	}

	// Second stage should get the piped input plus its own output
	if stage2Out.String() != "piped:tee test" {
		t.Errorf("stage2 stdout = %q, want %q", stage2Out.String(), "piped:tee test")
	}

	// Third stage should get the piped input plus its own output
	if stage3Out.String() != "piped2:piped:tee test" {
		t.Errorf("stage3 stdout = %q, want %q", stage3Out.String(), "piped2:piped:tee test")
	}
}

func TestExecPipeErrorHandling(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name      string
		stages    [][]string
		exitCodes []int
	}{
		{
			name: "first_stage_fails",
			stages: [][]string{
				{"-exit", "1"},
				{"-exit", "0", "-stdin2out"},
			},
			exitCodes: []int{1, 0},
		},
		{
			name: "second_stage_fails",
			stages: [][]string{
				{"-exit", "0", "-stdout", "data"},
				{"-exit", "2", "-stdin2out"},
			},
			exitCodes: []int{0, 2},
		},
		{
			name: "both_stages_fail",
			stages: [][]string{
				{"-exit", "1"},
				{"-exit", "2", "-stdin2out"},
			},
			exitCodes: []int{1, 2},
		},
		{
			name: "three_stage_failure",
			stages: [][]string{
				{"-exit", "1"},
				{"-exit", "2", "-stdin2out"},
				{"-exit", "3", "-stdin2out"},
			},
			exitCodes: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1 := gnoblib.Lib.Cmd.Exec(t.Context(), exe, tt.stages[0]...)
			for i := 1; i < len(tt.stages); i++ {
				c1 = c1.Pipe(exe, tt.stages[i]...)
			}

			err := c1.Run()
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			codes := c1.ExitCodes()
			t.Logf("Exit codes: %v", codes)
			if len(codes) != len(tt.exitCodes) {
				t.Fatalf("Exit Codes missmatches: got=%v, want=%v", codes, tt.exitCodes)
			}
			for i := range tt.exitCodes {
				if codes[i] != tt.exitCodes[i] {
					t.Fatalf("Exit Codes missmatches: got=%v, want=%v", codes, tt.exitCodes)
				}
			}
		})
	}
}

func TestExecStartWait(t *testing.T) {
	exe := mainExec(t)
	t.Run("start_then_wait", func(t *testing.T) {
		var stdout bytes.Buffer
		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(),
			gnoblib.Lib.Cmd.WithStdout(&stdout),
			exe, "-exit", "0", "-stdout", "async test")

		err := c.Start()
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		err = c.Wait()
		if err != nil {
			t.Fatalf("Wait() error = %v", err)
		}

		if stdout.String() != "async test" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "async test")
		}
	})

	t.Run("start_wait_with_pipe", func(t *testing.T) {
		var stdout bytes.Buffer
		c1 := gnoblib.Lib.Cmd.Exec(t.Context(), exe, "-exit", "0", "-stdout", "pipe async")
		c2 := c1.PipeOpt(
			gnoblib.Lib.Cmd.ExecOptions(
				gnoblib.Lib.Cmd.WithStdout(&stdout),
			),
			exe, "-exit", "0", "-stdin2out", "-stdout", "result:")

		err := c2.Start()
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		err = c2.Wait()
		if err != nil {
			t.Fatalf("Wait() error = %v", err)
		}

		if stdout.String() != "result:pipe async" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "result:pipe async")
		}
	})
}

func TestExecContextCancellation(t *testing.T) {
	exe := mainExec(t)
	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// This should timeout since we're not providing any way for the command to complete quickly
		c := gnoblib.Lib.Cmd.Exec(ctx, exe, "-exit", "0", "-sleep", "1000ms")
		err := c.Run()

		// Should get context deadline exceeded or similar
		if err == nil {
			t.Fatal("Expected error due to context timeout, got nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "signal: killed") {
			t.Logf("Got error (may be acceptable): %v", err)
		}
	})

	t.Run("context_cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		c := gnoblib.Lib.Cmd.Exec(ctx, exe, "-exit", "0")

		// Cancel before running
		cancel()

		err := c.Run()
		if err == nil {
			t.Error("Expected error due to context cancellation, got nil")
		}
	})
}

func TestExecEnvironmentVariables(t *testing.T) {
	exe := mainExec(t)
	tests := []struct {
		name      string
		envVars   map[string]string
		noInherit bool
	}{
		{
			name:    "with_env_vars",
			envVars: map[string]string{"TEST_VAR": "test_value", "ANOTHER": "value2"},
		},
		{
			name:      "no_inherit_env",
			envVars:   map[string]string{"ONLY_THIS": "value"},
			noInherit: true,
		},
		{
			name:    "empty_env_vars",
			envVars: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
				gnoblib.Lib.Cmd.WithEnvVars(tt.envVars),
				gnoblib.Lib.Cmd.WithNoInheritEnv(tt.noInherit),
			), exe, "-exit", "0")
			err := c.Run()
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func TestExecWorkingDirectory(t *testing.T) {
	exe := mainExec(t)
	c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.WithDir("."), exe, "-exit", "0")
	err := c.Run()
	if err != nil {
		t.Fatalf("Run() with workingDir error = %v", err)
	}
}

func TestExecInvalidCommand(t *testing.T) {
	exe := mainExec(t)
	t.Run("nonexistent_command", func(t *testing.T) {
		c := gnoblib.Lib.Cmd.Exec(t.Context(), "nonexistent_command_xyz")
		err := c.Run()
		if err == nil {
			t.Error("Expected error for nonexistent command, got nil")
		}
	})

	t.Run("invalid_arguments", func(t *testing.T) {
		c := gnoblib.Lib.Cmd.Exec(t.Context(), exe, "-invalid-flag")
		err := c.Run()
		if err == nil {
			t.Error("Expected error for invalid arguments, got nil")
		}
	})
}

func TestExecComplexPipeChains(t *testing.T) {
	exe := mainExec(t)
	t.Run("mixed_pipe_types", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		// Create a complex chain: stdout pipe -> stderr pipe -> stdout pipe
		c1 := gnoblib.Lib.Cmd.Exec(t.Context(), exe,
			"-exit", "0",
			"-stdout", "out1",
			"-stderr", "err1",
		)

		// Pipe stdout to next command
		c2 := c1.Pipe(exe,
			"-exit", "0",
			"-stdin2err", // Send stdin to stderr
			"-stdout", "out2",
		)

		// Pipe stderr to next command
		c3 := c2.Pipe2Opt(gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithStdout(&stdout),
			gnoblib.Lib.Cmd.WithStderr(&stderr),
		), exe,
			"-exit", "0",
			"-stdin2out", // Send stdin to stdout
			"-stdout", "final:",
		)

		err := c3.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// The final output should contain the piped stderr content
		if !strings.Contains(stdout.String(), "final:") {
			t.Errorf("stdout = %q, should contain 'final:'", stdout.String())
		}
	})

	t.Run("long_pipe_chain", func(t *testing.T) {
		var stdout bytes.Buffer

		c := gnoblib.Lib.Cmd.Exec(t.Context(), exe, "-exit", "0", "-stdout", "0")

		// Create a chain of 10 pipes, each adding its number
		for i := 1; i <= 10; i++ {
			var opts gnoblib.ExecOption
			if i == 10 {
				opts = gnoblib.Lib.Cmd.WithStdout(&stdout)
			}
			c = c.PipeOpt(opts, exe,
				"-exit", "0",
				"-stdin2out",
				"-stdout", fmt.Sprintf("%d", i))
		}

		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		expected := "10987654321" + "0"
		if stdout.String() != expected {
			t.Errorf("stdout = %q, want %q", stdout.String(), expected)
		}
	})
}

func TestExecEdgeCases(t *testing.T) {
	exe := mainExec(t)
	t.Run("empty_command_output", func(t *testing.T) {
		var stdout bytes.Buffer
		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.WithStdout(&stdout), exe, "-exit", "0")
		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})

	t.Run("large_output", func(t *testing.T) {
		largeText := strings.Repeat("A", 10000)
		var stdout bytes.Buffer
		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.WithStdout(&stdout), exe,
			"-exit", "0", "-stdout", largeText,
		)
		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if stdout.String() != largeText {
			t.Errorf("stdout length = %d, want %d", len(stdout.String()), len(largeText))
		}
	})

	t.Run("pipe_with_no_input", func(t *testing.T) {
		var stdout bytes.Buffer
		c1 := gnoblib.Lib.Cmd.Exec(t.Context(), exe, "-exit", "0") // No output
		c2 := c1.PipeOpt(gnoblib.Lib.Cmd.WithStdout(&stdout), exe, "-exit", "0", "-stdin2out", "-stdout", "prefix:")
		err := c2.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if stdout.String() != "prefix:" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "prefix:")
		}
	})
}

func TestExecEnvironmentPrinting(t *testing.T) {
	exe := mainExec(t)

	t.Run("printenv_stdout_with_custom_vars", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		customVars := map[string]string{
			"CUSTOM_VAR1": "value1",
			"CUSTOM_VAR2": "value2",
			"TEST_ENV":    "test_value",
		}

		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithEnvVars(customVars),
			gnoblib.Lib.Cmd.WithStdout(&stdout),
			gnoblib.Lib.Cmd.WithStderr(&stderr),
		), exe, "-exit", "0", "-printenv", "stdout")

		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		output := stdout.String()

		// Check that our custom variables are present
		for key, value := range customVars {
			expectedLine := fmt.Sprintf("%s=%s", key, value)
			if !strings.Contains(output, expectedLine) {
				t.Errorf("stdout should contain %q, got:\n%s", expectedLine, output)
			}
		}

		// Stderr should be empty
		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})

	t.Run("printenv_stderr_with_custom_vars", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		customVars := map[string]string{
			"STDERR_VAR":  "stderr_value",
			"ANOTHER_VAR": "another_value",
		}
		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithEnvVars(customVars),
			gnoblib.Lib.Cmd.WithStdout(&stdout),
			gnoblib.Lib.Cmd.WithStderr(&stderr),
		), exe, "-exit", "0", "-printenv", "stderr")

		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		output := stderr.String()

		// Check that our custom variables are present in stderr
		for key, value := range customVars {
			expectedLine := fmt.Sprintf("%s=%s", key, value)
			if !strings.Contains(output, expectedLine) {
				t.Errorf("stderr should contain %q, got:\n%s", expectedLine, output)
			}
		}

		// Stdout should be empty
		if stdout.String() != "" {
			t.Errorf("stdout should be empty, got %q", stdout.String())
		}
	})

	t.Run("printenv_no_inherit_env", func(t *testing.T) {
		var stdout bytes.Buffer

		customVars := map[string]string{
			"ONLY_VAR": "only_value",
			"ISOLATED": "isolated_value",
		}

		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithEnvVars(customVars),
			gnoblib.Lib.Cmd.WithNoInheritEnv(true),
			gnoblib.Lib.Cmd.WithStdout(&stdout)),
			exe, "-exit", "0", "-printenv", "stdout")
		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// With noInheritEnv, we should only have our custom variables
		// Filter out empty lines
		var nonEmptyLines []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines = append(nonEmptyLines, line)
			}
		}

		if len(nonEmptyLines) != len(customVars) {
			t.Errorf("Expected exactly %d environment variables, got %d lines:\n%s",
				len(customVars), len(nonEmptyLines), output)
		}

		// Check that only our custom variables are present
		for key, value := range customVars {
			expectedLine := fmt.Sprintf("%s=%s", key, value)
			if !strings.Contains(output, expectedLine) {
				t.Errorf("stdout should contain %q, got:\n%s", expectedLine, output)
			}
		}

		// Should not contain common inherited environment variables
		commonVars := []string{"PATH", "HOME", "USER"}
		for _, commonVar := range commonVars {
			if strings.Contains(output, commonVar+"=") {
				t.Errorf("stdout should not contain inherited variable %q when noInheritEnv=true, got:\n%s",
					commonVar, output)
			}
		}
	})

	t.Run("printenv_with_inherited_and_custom_vars", func(t *testing.T) {
		var stdout bytes.Buffer

		customVars := map[string]string{
			"MIXED_VAR1": "mixed_value1",
			"MIXED_VAR2": "mixed_value2",
		}

		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithEnvVars(customVars),
			gnoblib.Lib.Cmd.WithNoInheritEnv(false), // Explicitly inherit
			gnoblib.Lib.Cmd.WithStdout(&stdout),
		),
			exe, "-exit", "0", "-printenv", "stdout")

		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		output := stdout.String()

		// Check that our custom variables are present
		for key, value := range customVars {
			expectedLine := fmt.Sprintf("%s=%s", key, value)
			if !strings.Contains(output, expectedLine) {
				t.Errorf("stdout should contain custom var %q, got:\n%s", expectedLine, output)
			}
		}

		// Should also contain some inherited variables (PATH is almost always present)
		if !strings.Contains(output, "PATH=") {
			t.Errorf("stdout should contain inherited PATH variable when noInheritEnv=false, got:\n%s", output)
		}
	})

	t.Run("printenv_invalid_stream", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		// Test with invalid printenv value - should not print environment
		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithStdout(&stdout),
			gnoblib.Lib.Cmd.WithStderr(&stderr),
			gnoblib.Lib.Cmd.WithEnvVars(map[string]string{"TEST_VAR": "test_value"}),
		), exe,
			"-exit", "0",
			"-printenv", "invalid",
			"-stdout", "normal output",
		)
		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// Should only have the normal stdout output, no environment variables
		if stdout.String() != "normal output" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "normal output")
		}

		if stderr.String() != "" {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})

	t.Run("printenv_empty_value", func(t *testing.T) {
		var stdout bytes.Buffer

		// Test with empty printenv value - should not print environment
		c := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithStdout(&stdout),
			gnoblib.Lib.Cmd.WithEnvVars(map[string]string{"TEST_VAR": "test_value"}),
		), exe, "-exit", "0", "-printenv", "", "-stdout", "normal output")

		err := c.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// Should only have the normal stdout output
		if stdout.String() != "normal output" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "normal output")
		}
	})
}

func TestExecEnvironmentPrintingInPipes(t *testing.T) {
	exe := mainExec(t)

	t.Run("printenv_in_pipe_chain", func(t *testing.T) {
		var stdout bytes.Buffer

		// First stage prints environment, second stage processes it
		c1 := gnoblib.Lib.Cmd.ExecOpt(t.Context(),
			gnoblib.Lib.Cmd.WithEnvVars(map[string]string{
				"PIPE_VAR1": "pipe_value1",
				"PIPE_VAR2": "pipe_value2",
			}), exe,
			"-exit", "0",
			"-printenv", "stdout")

		c2 := c1.PipeOpt(gnoblib.Lib.Cmd.WithStdout(&stdout), exe,
			"-exit", "0",
			"-stdin2out",
			"-stdout", "ENV_OUTPUT:")
		err := c2.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		output := stdout.String()

		// Should start with our prefix
		if !strings.HasPrefix(output, "ENV_OUTPUT:") {
			t.Errorf("output should start with 'ENV_OUTPUT:', got: %s", output)
		}

		// Should contain our custom environment variables
		if !strings.Contains(output, "PIPE_VAR1=pipe_value1") {
			t.Errorf("output should contain PIPE_VAR1=pipe_value1, got: %s", output)
		}

		if !strings.Contains(output, "PIPE_VAR2=pipe_value2") {
			t.Errorf("output should contain PIPE_VAR2=pipe_value2, got: %s", output)
		}
	})

	t.Run("printenv_different_stages_different_envs", func(t *testing.T) {
		var stage1Out, stage2Out bytes.Buffer

		// Each stage has different environment variables
		c1 := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithEnvVars(map[string]string{"STAGE1_VAR": "stage1_value"}),
			gnoblib.Lib.Cmd.WithStdout(&stage1Out),
		), exe,
			"-exit", "0",
			"-printenv", "stdout")

		c2 := c1.PipeOpt(gnoblib.Lib.Cmd.ExecOptions(
			gnoblib.Lib.Cmd.WithEnvVars(map[string]string{"STAGE2_VAR": "stage2_value"}),
			gnoblib.Lib.Cmd.WithStdout(&stage2Out),
		), exe, "-exit", "0",
			"-printenv", "stdout")

		err := c2.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// First stage should have its environment variable
		stage1Output := stage1Out.String()
		if !strings.Contains(stage1Output, "STAGE1_VAR=stage1_value") {
			t.Errorf("stage1 output should contain STAGE1_VAR=stage1_value, got: %s", stage1Output)
		}

		// Second stage should have its environment variable
		stage2Output := stage2Out.String()
		if !strings.Contains(stage2Output, "STAGE2_VAR=stage2_value") {
			t.Errorf("stage2 output should contain STAGE2_VAR=stage2_value, got: %s", stage2Output)
		}
	})

	t.Run("printenv_stderr_in_pipe2", func(t *testing.T) {
		var stdout bytes.Buffer

		// First stage prints to stderr, second stage pipes stderr to stdout
		c1 := gnoblib.Lib.Cmd.ExecOpt(t.Context(), gnoblib.Lib.Cmd.WithEnvVars(map[string]string{"STDERR_PIPE_VAR": "stderr_pipe_value"}), exe,
			"-exit", "0",
			"-printenv", "stderr",
		)
		c2 := c1.Pipe2Opt(gnoblib.Lib.Cmd.WithStdout(&stdout), exe,
			"-exit", "0",
			"-stdin2out",
			"-stdout", "STDERR_ENV:",
		)

		err := c2.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		output := stdout.String()

		// Should contain the environment variable that was printed to stderr and piped
		if !strings.Contains(output, "STDERR_PIPE_VAR=stderr_pipe_value") {
			t.Errorf("output should contain STDERR_PIPE_VAR=stderr_pipe_value, got: %s", output)
		}

		// Should have our prefix
		if !strings.HasPrefix(output, "STDERR_ENV:") {
			t.Errorf("output should start with 'STDERR_ENV:', got: %s", output)
		}
	})
}
