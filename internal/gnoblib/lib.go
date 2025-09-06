package gnoblib

// Lib is the library of functions used by gnob.
var Lib _lib

// Lib is the library of functions used by gnob.
type _lib struct {
	// Main is the root of the library.
	Main _root
	// Template is a collection of operations using Go templates.
	Template _template
	// Files is a collection of operations on files.
	Files _files
	// Cmd is a collection of operations to run commands.
	Cmd _cmd
	// Make has operations to construct a Makefile and MakeTargets.
	Makefile _makefile
}
