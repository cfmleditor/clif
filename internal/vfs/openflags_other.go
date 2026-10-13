//go:build !unix

package vfs

// openFlags is nothing where there is no non-blocking open to ask for.
const openFlags = 0
