package vfs

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// MaxFileSize is the most clif reads of any one file. The largest CFML file in
// any workspace measured is a few megabytes; a workspace file past this is not
// source anyone edits, and a read with no limit is how a symlink to /dev/zero
// took a scan to 17GB in eight seconds.
const MaxFileSize = 32 << 20

// ErrNotRegular is a path that names a device, a FIFO, a socket or a
// directory, whatever a symlink at it points to.
var ErrNotRegular = errors.New("not a regular file")

// ErrTooLarge is a file larger than MaxFileSize.
var ErrTooLarge = errors.New("file too large")

// ReadRegular reads the file at path when it is a regular file of at most
// MaxFileSize bytes, following a symlink to decide. Anything else is refused
// rather than read: a FIFO blocks a read forever, and a device reads without
// end.
//
// The file is opened non-blocking where the platform has the flag, because
// opening a FIFO for reading otherwise waits for a writer, and the check comes
// from the opened file, so nothing can be swapped in between the check and the
// read.
func ReadRegular(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|openFlags, 0) //nolint:gosec // the read every file goes through; callers choose the path
	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !info.Mode().IsRegular() {
		return nil, &os.PathError{Op: "read", Path: path, Err: ErrNotRegular}
	}

	if info.Size() > MaxFileSize {
		return nil, &os.PathError{Op: "read", Path: path, Err: fmt.Errorf("%w: %d bytes, the limit is %d", ErrTooLarge, info.Size(), MaxFileSize)}
	}

	// The size is a hint, and the limit holds whatever the file does after
	// the Stat: one byte past it says it grew.
	data := make([]byte, 0, info.Size()+1)

	data, err = readAllInto(data, io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return nil, err
	}

	if len(data) > MaxFileSize {
		return nil, &os.PathError{Op: "read", Path: path, Err: ErrTooLarge}
	}

	return data, nil
}

// readAllInto is io.ReadAll appending to buf, so a buffer sized from the Stat
// is filled without being regrown.
func readAllInto(buf []byte, r io.Reader) ([]byte, error) {
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}

		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]

		if errors.Is(err, io.EOF) {
			return buf, nil
		}

		if err != nil {
			return buf, err
		}
	}
}
