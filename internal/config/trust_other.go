//go:build !unix

package config

import "io/fs"

// Untrusted always allows the file here. Windows has no owner and mode bits
// to read this way, and a wasm build reads through the editor.
func Untrusted(fs.FileInfo) string {
	return ""
}
