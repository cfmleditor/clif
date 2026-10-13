//go:build !unix

package vfs

import "errors"

func mkfifo(string) error { return errors.New("no FIFOs on this platform") }
