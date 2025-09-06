package gnoblib

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type _makefile struct {
}

// FileUpToDate returns a function that returns true if the target is up-to-date.
// The target is up-to-date if the target file is newer than all the sources.
// This can be used as the UpToDate function of a MakeTarget.
func (_makefile) FileUpToDate(target string, sources ...string) func(*Makefile) bool {
	var f _files
	return func(mf *Makefile) bool {
		return !f.TargetNeedsUpdate(target, sources...)
	}
}

// Makefile is a collection of targets.
// This can be used as a main function to make gnob behave like a Makefile.
type Makefile struct {
	name          string
	args          []string
	commandArgs   []string
	targets       []*MakeTarget
	defaultTarget int
	ctx           context.Context
}

// New construct a makefile from the given targets.
// The name of the program is taken from the first argument of os.Args.
// The argument list is taken from the second argument of os.Args.
func (m _makefile) New(targets ...MakeTarget) *Makefile {
	return m.NewEx(os.Args[0], os.Args[1:], targets...)
}

// NewEx is like New, but allows you to customize the name of the program and the argument list.
func (_makefile) NewEx(name string, args []string, targets ...MakeTarget) *Makefile {
	tgt := make([]*MakeTarget, 0, len(targets))
	for i := range targets {
		tgt = append(tgt, &targets[i])
	}
	td := Makefile{
		name:    name,
		args:    args,
		targets: tgt,
	}
	td.normalize()
	return &td
}

// Depend executes the targets with the given names.
// Execution is done in the order of the names.
// If any of the targets is not found, it returns an error.
// If any of the targets encounters an error, it returns the error immediately.
// If all targets are executed successfully, it returns nil.
func (mf *Makefile) Depend(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return nil
	}
	targets := make([]*MakeTarget, 0, len(names))
	for _, name := range names {
		found := mf.Find(name)
		if found != nil {
			targets = append(targets, found)
			continue
		}
		return fmt.Errorf("unknown target: %s", name)
	}
	for _, tgt := range targets {
		if err := tgt.exec(ctx, mf); err != nil {
			return err
		}
	}
	return nil
}

// Find returns the target with the given name.
// If the target is not found, it returns nil.
func (mf *Makefile) Find(name string) *MakeTarget {
	for _, tgt := range mf.targets {
		if strings.EqualFold(tgt.Name, name) {
			return tgt
		}
	}
	return nil
}

// Add adds more targets to the Makefile.
func (mf *Makefile) Add(tgts ...MakeTarget) {
	for _, tgt := range tgts {
		mf.targets = append(mf.targets, &tgt)
	}
	mf.normalize()
}

// TargetArgs returns the argument list for the target.
func (mf *Makefile) TargetArgs() []string {
	return mf.commandArgs
}

// Run runs the target.
// If it encounters an error, it logs the error and exits with status code 1.
func (mf *Makefile) Run(ctx context.Context) {
	mf.ctx = ctx
	if err := mf.RunE(ctx); err != nil {
		Logger.Error("[gnob:makefile] error running build target", "error", err)
		os.Exit(1)
	}
}

// RunE runs the target and returns an error if any.
func (mf *Makefile) RunE(ctx context.Context) error {
	if len(mf.args) == 0 {
		return mf.runDefault(ctx)
	}
	cmd := mf.args[0]
	mf.commandArgs = mf.args[1:]
	if cmd == "-help" {
		return mf.showHelp()
	}
	tgt := mf.Find(cmd)
	if tgt == nil {
		return fmt.Errorf("unknown target: %s", cmd)
	}
	return tgt.exec(ctx, mf)
}

func (mf *Makefile) runDefault(ctx context.Context) error {
	return mf.targets[mf.defaultTarget].exec(ctx, mf)
}

func (mf *Makefile) showHelp() error {
	if len(mf.commandArgs) > 0 {
		tgt := mf.Find(mf.commandArgs[0])
		if tgt == nil {
			return fmt.Errorf("unknown target: %s", mf.commandArgs[0])
		}
		return tgt.showHelp(mf)
	}
	maxLen := 0
	for _, tgt := range mf.targets {
		if tgt.Hidden {
			continue
		}
		maxLen = max(maxLen, len(tgt.Name))
	}
	fmt.Println("Usage: " + filepath.Base(mf.name) + " [-help] [target]")
	fmt.Println("Targets:")
	fmtStr := fmt.Sprintf("%%-%ds   %%s\n", maxLen)
	for i, tgt := range mf.targets {
		if tgt.Hidden {
			continue
		}
		if mf.defaultTarget == i {
			fmt.Printf("* "+fmtStr, tgt.Name, tgt.Desc)
			continue
		}
		fmt.Printf("  "+fmtStr, tgt.Name, tgt.Desc)
	}
	fmt.Println("\n* (default target)")
	return nil
}

func (mf *Makefile) normalize() {
	names := make(map[string]struct{})
	normalTargets := make([]*MakeTarget, 0, len(mf.targets))
	for i := range mf.targets {
		name := strings.ToLower(mf.targets[i].Name)
		if _, ok := names[name]; ok {
			Logger.Warn("[gnob] duplicate target", "name", name, "index", i)
			// skip duplicate targets
			continue
		}
		normalTargets = append(normalTargets, mf.targets[i])
		names[name] = struct{}{}
	}
	slices.SortFunc(normalTargets, func(a, b *MakeTarget) int {
		if a.Default {
			return -1
		}
		if b.Default {
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	mf.targets = normalTargets
	mf.defaultTarget = 0
	for i := range mf.targets {
		if mf.targets[i].Default {
			mf.defaultTarget = i
			break
		}
	}
}

// MakeTarget is a target that can be executed by a Makefile.
type MakeTarget struct {
	// Name is the name of the target.
	Name string
	// Desc is a short description of the target.
	// It is shown when listing all targets with `gnob -help`.
	Desc string
	// LongDesc is a long description of the target.
	// It is shown when running `gnob -help <target>`.
	LongDesc string
	// Hidden is true if the target should be hidden from listing.
	Hidden bool
	// Default is true if the target is the default target.
	// Only one target can be the default target.
	Default bool
	// UpToDate is a function that returns true if the target is up-to-date.
	// If the target is up-to-date, the target will not be executed.
	UpToDate func(mf *Makefile) bool
	// Body is the function that executes the target.
	// When the target is not up-to-date, this body will be executed.
	Body func(ctx context.Context, mf *Makefile) error
}

// Exec executes the target.
func (mt *MakeTarget) exec(ctx context.Context, mf *Makefile) error {
	Logger.Debug("[gnob:makefile] execute target", "target", mt.Name)
	if mt.UpToDate == nil || !mt.UpToDate(mf) {
		if err := mt.Body(ctx, mf); err != nil {
			Logger.Error("[gnob:makefile] error executing target", "target", mt.Name, "error", err)
			return err
		}
		return nil
	}
	Logger.Info("[gnob:makefile] target is up-to-date", "target", mt.Name)
	return nil
}

func (mt *MakeTarget) showHelp(mf *Makefile) error {
	Logger.Debug("[gnob:makefile] show help", "target", mt.Name)
	fmt.Printf("%s %s:\n", filepath.Base(mf.name), mt.Name)
	fmt.Println()
	fmt.Println(mt.Desc)
	if mt.LongDesc != "" {
		fmt.Println(mt.LongDesc)
	}
	return nil
}
