package main

import (
	"bytes"
	"context"
	"strings"
)

func examples(ctx context.Context) error {
	// --- simple command ---
	if err := GnobLib.Cmd.Exec(ctx, "go", "build", "./...").Run(); err != nil {
		GnobLogger.Error("failed to build", "error", err)
		return err
	}
	// --- simple command ---

	// --- output capture ---
	var output bytes.Buffer
	if err := GnobLib.Cmd.ExecOpt(ctx, GnobLib.Cmd.ExecOptions(
		GnobLib.Cmd.WithStdout(&output),
	), "git", "rev-parse", "HEAD").Run(); err != nil {
		return err
	}
	commitHash := strings.TrimSpace(output.String())
	GnobLogger.Info("commit hash", "hash", commitHash)
	// --- output capture ---

	// --- command pipeline ---
	var result bytes.Buffer
	p := GnobLib.Cmd.Exec(ctx, "echo", "hello")
	p = p.Pipe("tr", "[:lower:]", "[:upper:]")
	p = p.PipeOpt(GnobLib.Cmd.WithStdout(&result), "wc", "-c")
	if err := p.Run(); err != nil {
		return err
	}
	GnobLogger.Info("result", "result", result.String())
	// --- command pipeline ---

	// --- json processing ---
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
	// --- json processing ---

	var err error
	// --- logging ---
	GnobLogger.Info("starting build", "target", "production")
	GnobLogger.Warn("deprecated flag used", "flag", "--old-flag")
	GnobLogger.Error("build failed", "error", err, "step", "compilation")
	// --- logging ---
	return nil
}
