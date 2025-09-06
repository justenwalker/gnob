//go:build gnob

package main

import (
	"context"
)

//go:generate go build -o gnob -tags gnob ./...

// convenience variables
var (
	gnob     = GnobLib.Main
	makefile = GnobLib.Makefile
	logger   = GnobLogger
)

func main() {
	gnob.GoRebuildYourself("*.go")
	mf := makefile.New(Default, TaskOne, TaskTwo, TaskThree)
	mf.Run(context.Background())
}

var Default = GnobMakeTarget{
	Name:     "default",
	Desc:     "default target",
	LongDesc: "This is the default target",
	Default:  true,
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		// Run dependencies first
		if err := mf.Depend(ctx, "task1", "task2", "task3"); err != nil {
			return err
		}
		// Now perform the default action
		logger.Info("[example:gnobmake] Default Target")
		return nil
	},
}

var TaskOne = GnobMakeTarget{
	Name:     "task1",
	Desc:     "Task1",
	LongDesc: "This is the first task",
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		logger.Info("[example:gnobmake] Task 1")
		return nil
	},
}
var TaskTwo = GnobMakeTarget{
	Name:     "task2",
	Desc:     "Task2",
	LongDesc: "This is the second task",
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		logger.Info("[example:gnobmake] Task 2")
		return nil
	},
}
var TaskThree = GnobMakeTarget{
	Name:     "task3",
	Desc:     "Task3",
	LongDesc: "This is the third task",
	Body: func(ctx context.Context, mf *GnobMakefile) error {
		logger.Info("[example:gnobmake] Task 3")
		return nil
	},
}
