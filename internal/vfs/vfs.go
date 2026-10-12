// Package vfs defines a filesystem interface for portability across native and WASM builds.
package vfs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS abstracts filesystem operations.
type FS interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Walk(root string, fn filepath.WalkFunc) error
}

// StringReader is an FS that can hand over a file's text as a string without
// copying it: the batch content cache holds each file's text once, and every
// reader converting the bytes it returned was a second copy of the file.
type StringReader interface {
	ReadFileString(path string) (string, error)
}

// ReadString is the text of the file at path: from f's ReadFileString when it
// has one, else its ReadFile, converted.
func ReadString(f FS, path string) (string, error) {
	if sr, ok := f.(StringReader); ok {
		return sr.ReadFileString(path)
	}

	data, err := f.ReadFile(path)

	return string(data), err
}

// OS is the native filesystem implementation using the os package.
type OS struct{}

// ReadFile reads the named file.
func (OS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // the file-system layer every read goes through; callers choose the path
}

// Stat returns file info for the named path.
func (OS) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

// ReadDir reads the named directory.
func (OS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

// Walk walks the file tree rooted at root.
func (OS) Walk(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}
