package gnoblib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	EnvRebuildDisable = "GNOB_REBUILD_DISABLE"
	EnvLogLevel       = "GNOB_LOG_LEVEL"
)

var (
	BinaryName string
	GoCommand  = "go"
)

func (r _root) GoRebuildYourself(sources ...string) {
	if BinaryName == "" {
		switch runtime.GOOS {
		case "windows":
			BinaryName = "gnob.exe"
		default:
			BinaryName = "gnob"
		}
	}
	if err := r.RebuildYourself(context.Background(), sources...); err != nil {
		Logger.Error("[gnob:rebuild] failed to rebuild", "error", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

type _root struct {
}

func (r _root) RebuildYourself(ctx context.Context, sources ...string) error {
	if os.Getenv(EnvRebuildDisable) != "" {
		Logger.DebugContext(ctx, "[gnob:rebuild] rebuild disabled")
		return nil
	}
	binary, err := filepath.Abs(os.Args[0])
	if err != nil {
		return fmt.Errorf("could not get absolute path of %s: %v", os.Args[0], err)
	}
	name := filepath.Base(binary)
	Logger.DebugContext(ctx, "[gnob:rebuild] binary name", "name", name)

	if name != BinaryName { // run via go run or some other mechanism
		Logger.DebugContext(ctx, "[gnob:rebuild] expected binary name did not match", "name", name, "expected", BinaryName)
		_, err = os.Stat(BinaryName)
		if os.IsNotExist(err) {
			Logger.DebugContext(ctx, "[gnob:rebuild] binary does not exist, rebuilding", "binary", BinaryName)
			return r.rebuild(ctx, BinaryName, sources)
		}
		if err != nil {
			return fmt.Errorf("failed to stat %s: %v", BinaryName, err)
		}
		if err = r.runBinary(ctx, BinaryName); err != nil {
			return fmt.Errorf("failed to run %s: %v", BinaryName, err)
		}
		os.Exit(0)
	}
	sources, err = r.normalizeSources(binary, sources)
	if err != nil {
		return err
	}
	var f _files
	Logger.DebugContext(ctx, "[gnob:rebuild] testing if gnob needs to be rebuilt", "sources", sources)
	if f.TargetNeedsUpdate(binary, sources...) {
		Logger.DebugContext(ctx, "[gnob:rebuild] gnob must be rebuilt")
		if err = r.rebuild(ctx, binary, sources); err != nil {
			return fmt.Errorf("failed to build %s: %v", binary, err)
		}
		if err = r.runBinary(ctx, binary); err != nil {
			return fmt.Errorf("failed to run %s: %v", binary, err)
		}
		os.Exit(0)
	}
	Logger.DebugContext(ctx, "[gnob:rebuild] gnob is up to date")
	return nil
}

func (r _root) normalizeSources(bin string, sources []string) ([]string, error) {
	dir := filepath.Dir(bin)
	out := make([]string, 0, len(sources))

	for _, s := range sources {
		if !filepath.IsAbs(s) {
			snew, err := filepath.Abs(filepath.Join(dir, s))
			if err != nil {
				return nil, fmt.Errorf("unable to make %q an absolute path: %w", s, err)
			}
			s = snew
		}
		if strings.ContainsRune(s, '*') { // is glob
			matches, err := filepath.Glob(s)
			if err != nil {
				return nil, fmt.Errorf("unable to expand glob %q: %w", s, err)
			}
			Logger.Debug("[gnob:rebuild] glob matches", "glob", s, "matches", matches)
			for _, m := range matches {
				out = append(out, m)
			}
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (r _root) runBinary(ctx context.Context, binary string) error {
	Logger.DebugContext(ctx, "[gnob:rebuild] executing", "binary", binary)
	cmd := exec.CommandContext(ctx, binary, os.Args[1:]...)
	cmd.Env = append(os.Environ(),
		"GNOB_REBUILD_DISABLE=1",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (r _root) rebuild(ctx context.Context, binary string, sources []string) error {
	Logger.DebugContext(ctx, "[gnob:rebuild] rebuilding", "binary", binary)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	tags := "gnob"
	if rbi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range rbi.Settings {
			if s.Key == "-tags" {
				tags = s.Value
				break
			}
		}
	}
	args := []string{"build",
		"-tags", tags,
		"-o", binary,
	}
	var stderr bytes.Buffer
	args = append(args, sources...)
	cmd := exec.Command(GoCommand, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	if err != nil {
		return err
	}
	go r.forwardSignals(ctx, cmd)
	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("failed to build %s: %w\n%s", binary, err, stderr.String())
	}
	return nil
}

func (r _root) forwardSignals(ctx context.Context, cmd *exec.Cmd) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh)
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigCh:
			_ = cmd.Process.Signal(sig)
		}
	}
}
