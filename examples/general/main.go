//go:build gnob

package main

import (
	"bytes"
	"context"
)

//go:generate go build -o gnob -tags gnob ./...

// convenience variables
var (
	gnob     = GnobLib.Main
	cmd      = GnobLib.Cmd
	makefile = GnobLib.Makefile
	logger   = GnobLogger
)

func main() {
	ctx := context.Background()
	gnob.GoRebuildYourself("*.go")
	mf := makefile.New(
		GnobMakeTarget{
			Name: "default",
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				if err := mf.Depend(ctx, "out.json", "hello.out"); err != nil {
					return err
				}
				GnobLogger.Info("[example:general] default target")
				return nil
			},
		},
		GnobMakeTarget{
			Name:     "out.json",
			Hidden:   true,
			UpToDate: GnobLib.Makefile.FileUpToDate("out.json", "example.json"),
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				logger.Info("[example:general] building out.json")
				type MyObject struct {
					Msg string `json:"msg"`
				}
				var obj MyObject
				p := cmd.Exec(ctx, "cat", "example.json")
				p = p.PipeOpt(cmd.WithStdoutJSONDecoder(&obj), "jq", `{"msg": .msg}`)
				p = p.Pipe("tee", "out.json")
				return p.Run()
			},
		},
		GnobMakeTarget{
			Name:     "hello.out",
			Hidden:   true,
			UpToDate: GnobLib.Makefile.FileUpToDate("hello.out", "out.json"),
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				if err := mf.Depend(ctx, "out.json"); err != nil {
					return err
				}
				logger.Info("[example:general] building hello.out")
				var buf bytes.Buffer
				p := cmd.Exec(ctx, "cat", "out.json")
				p = p.PipeOpt(cmd.WithStdout(&buf), "jq", "-r", ".msg")
				p = p.Pipe("tee", "hello.out")
				return p.Run()
			},
		})
	mf.Run(ctx)
}
