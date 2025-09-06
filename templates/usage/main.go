//go:build gnob

package main

//go:generate go build -o gnob -tags gnob ./...

// Most functionality is available through member objects of GnobLib
// This allows the functions to be isolated in a single namespace.
//
// You can create some convenience variable to alias these fields.
var gnob = GnobLib.Main

// The GnobLib.Main.GoRebuildYourself function automatically rebuilds the
//gnob binary when source files are newer than the executable:

func main() {
	// Rebuild if any *.go files are newer than the gnob binary.
	gnob.GoRebuildYourself("*.go")

	// Rebuild only if specific files are newer than the gnob binary.
	gnob.GoRebuildYourself("main.go", "build.go", "tasks.go")

	// Your build logic here
}
