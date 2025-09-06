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
