//go:build gnob

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

//go:generate go build -o gnob -tags gnob ./...

// convenience variables
var (
	gnob   = GnobLib.Main
	cmd    = GnobLib.Cmd
	logger = GnobLogger
)

func main() {
	ctx := context.Background()
	gnob.GoRebuildYourself("*.go")
	if err := buildDocumentation(ctx); err != nil {
		logger.Error("[example:docs] documentation build failed", "error", err)
		os.Exit(1)
	}
}

func buildDocumentation(ctx context.Context) error {
	type DocConfig struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	}

	// Read config into json object.
	var config DocConfig
	p := cmd.ExecOpt(ctx, cmd.WithStdoutJSONDecoder(&config),
		"cat", "config.json")
	p = p.Pipe("jq", "-r", ".title")
	if err := p.Run(); err != nil {
		return fmt.Errorf("config read failed: %w", err)
	}

	// Convert markdown to HTML
	h := sha256.New()
	q := cmd.Exec(ctx, "pandoc", "-f", "markdown", "-t", "html", "README.md")
	q = q.PipeOpt(cmd.WithStdout(h), "tee", "output.html")
	if err := q.Run(); err != nil {
		return fmt.Errorf("documentation build failed: %w", err)
	}

	logger.Info("[example:docs] documentation built successfully",
		"title", config.Title,
		"version", config.Version,
		"sha-256", hex.EncodeToString(h.Sum(nil)))

	return nil
}
