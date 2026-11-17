//go:build !unix

package flock

import "os"

// Raise does nothing here: the caller exits with a code.
func Raise(os.Signal) {}
