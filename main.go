package main

import (
	"context"
	"os"
	"path/filepath"
)

//go:generate go build -o gnob -tags gnob .

// convenience variables
var (
	gnob     = GnobLib.Main
	cmd      = GnobLib.Cmd
	files    = GnobLib.Files
	makefile = GnobLib.Makefile
	logger   = GnobLogger
	tmpl     = GnobLib.Template
)

func main() {
	gnob.GoRebuildYourself("*.go")
	mf := makefile.New(
		GnobMakeTarget{
			Name:    "all",
			Default: true,
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				return mf.Depend(ctx, "gnob.go", "example")
			},
		},
		GnobMakeTarget{
			Name:     "gnob.go",
			UpToDate: makefile.FileUpToDate("gnob.go", "internal/gnoblib/*.go"),
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				if err := mf.Depend(ctx, "internal/gnoblib/a.go"); err != nil {
					return err
				}
				logger.Info("Building gnob.go")
				if err := cmd.Exec(ctx, "go", "tool", "golang.org/x/tools/cmd/bundle",
					"-o", "gnob.go",
					"-dst", ".",
					"-pkg", "main",
					"-prefix", "Gnob",
					"-tags", "gnob",
					"./internal/gnoblib").Run(); err != nil {
					return err
				}
				return nil
			},
		},
		GnobMakeTarget{
			Name: "internal/gnoblib/a.go",
			UpToDate: makefile.FileUpToDate(
				"internal/gnoblib/a.go",
				"README.md",
				"templates/a.go.tpl"),
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				if err := mf.Depend(ctx, "README.md"); err != nil {
					return err
				}
				logger.Info("Building internal/gnoblib/a.go")
				tt, err := tmpl.ParseFile("templates/a.go.tpl")
				if err != nil {
					return err
				}
				if err = tmpl.WriteFile("internal/gnoblib/a.go", os.FileMode(0o644), tt, nil); err != nil {
					return err
				}
				return nil
			},
		},
		GnobMakeTarget{
			Name: "README.md",
			UpToDate: makefile.FileUpToDate("README.md",
				"gnob.go",
				"templates/*.tpl",
				"templates/cmdpipe/*",
				"templates/makefile/*",
				"templates/usage/*",
			),
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				logger.Info("Building README.md")
				templates := []string{
					filepath.Join("templates", "cmdpipe"),
					filepath.Join("templates", "makefile"),
					filepath.Join("templates", "usage"),
				}
				for _, tt := range templates {
					if err := files.CopyFile(filepath.Join(tt, "gnob.go"), "gnob.go", 0); err != nil {
						return err
					}
				}
				tpl, err := tmpl.ParseFile(filepath.Join("templates", "README.md.tpl"))
				if err != nil {
					return err
				}
				if err = tmpl.WriteFile("README.md", os.FileMode(0o644), tpl, nil); err != nil {
					return err
				}
				return nil
			},
		},
		GnobMakeTarget{
			Name: "test",
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				if err := cmd.Exec(ctx, "go", "generate", "./internal/gnobtest").Run(); err != nil {
					return err
				}
				return cmd.Exec(ctx, "go", "test", "-v", "-tags", "gnob", "-coverprofile", "cover.out", "-coverpkg", "./internal/gnoblib", "./...").Run()
			},
		},
		GnobMakeTarget{
			Name: "example",
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				return mf.Depend(ctx, "examples/docs", "examples/general", "examples/gnobmake")
			},
		})
	mf.Add(exampleTargets("docs")...)
	mf.Add(exampleTargets("general")...)
	mf.Add(exampleTargets("gnobmake")...)
	mf.Run(context.Background())
}

func exampleTargets(name string) []GnobMakeTarget {
	exampleDir := filepath.Join("examples", name)
	exampleGnob := filepath.Join(exampleDir, "gnob")
	exampleGnobGo := filepath.Join(exampleDir, "gnob.go")
	return []GnobMakeTarget{
		{
			Name:     exampleGnob,
			UpToDate: makefile.FileUpToDate(exampleGnobGo, "gnob.go"),
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				logger.Info("[example] Building example", "example", name)
				if err := files.CopyFile(exampleGnobGo, "gnob.go", 0); err != nil {
					return err
				}
				if err := cmd.Exec(ctx, "go", "generate", "-C", exampleDir, ".").Run(); err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name: exampleDir,
			Body: func(ctx context.Context, mf *GnobMakefile) error {
				if err := mf.Depend(ctx, exampleGnob); err != nil {
					return err
				}
				logger.Info("[example] Testing example", "example", name)
				if err := cmd.Exec(ctx, "make", "-C", exampleDir, "test").Run(); err != nil {
					return err
				}
				return nil
			},
		},
	}
}
