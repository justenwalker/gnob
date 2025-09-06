package gnoblib

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type _files struct{}

// CopyDirectory copies a directory recursively from src to dst.
// This will overwrite any files in dst if they already exist.
func (f _files) CopyDirectory(dst, src string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		newPath := filepath.Join(dst, strings.TrimPrefix(path, src))

		// Directories
		if info.IsDir() {
			return os.MkdirAll(newPath, info.Mode())
		}

		// Irregular Files
		if !info.Mode().IsRegular() {
			switch d.Type() {
			case os.ModeSymlink:
				var link string
				link, err = os.Readlink(path)
				if err != nil {
					return err
				}
				return f.Symlink(link, newPath)
			default:
				Logger.Warn("[gnob:Copydirectory] ignoring irregular file type", "type", d.Type().String(), "path", path)
				// Skip other irregular file types.
				return nil
			}
		}
		// Regular files
		return f.CopyFile(newPath, path, info.Mode())
	})
}

// Symlink creates a symlink at 'path' pointing to 'link'.
// If 'path' already exists, it will be removed first.
func (f _files) Symlink(link string, path string) error {
	if f.Exists(path) {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return os.Symlink(link, path)
}

func (f _files) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CopyFile copies a file from src to dst.
// if mode is 0, the file is copied with the original permissions of the source.
func (f _files) CopyFile(dst, src string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %w", src, err)
	}
	defer srcFile.Close()
	if mode == 0 {
		stat, err := srcFile.Stat()
		if err != nil {
			return fmt.Errorf("unable to stat file %q: %w", src, err)
		}
		mode = stat.Mode()
	}
	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("unable to create dest file %q: %w", dst, err)
	}
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		_ = dstFile.Close()
		_ = os.Remove(dst)
		return fmt.Errorf("unable to copy %q -> %q: %w", src, dst, err)
	}
	if err = dstFile.Close(); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("unable to close dest file %q: %w", dst, err)
	}
	return nil
}

// LatestTimestamp expands glob patterns and returns the maximum modification
// time among all matched files. If no files match, it returns a Zero time.
// If a glob pattern is malformed or a file stat fails unexpectedly, it returns an error.
func (f _files) LatestTimestamp(files ...string) time.Time {
	var maxTs time.Time
	matchedAny := false

	for _, pattern := range files {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return time.Time{}
		}
		for _, p := range matches {
			ts := f.modTime(p)
			matchedAny = true
			if ts.After(maxTs) {
				maxTs = ts
			}
		}
	}

	if !matchedAny {
		// No matches: no error per function contract comment.
		return time.Time{}
	}
	return maxTs
}

func (f _files) modTime(file string) time.Time {
	fi, statErr := os.Stat(file)
	if statErr != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// TargetNeedsUpdate returns true if the target file is older than any of the
// sources.
func (f _files) TargetNeedsUpdate(target string, sources ...string) bool {
	a := f.modTime(target)
	b := f.LatestTimestamp(sources...)
	return b.After(a)
}
