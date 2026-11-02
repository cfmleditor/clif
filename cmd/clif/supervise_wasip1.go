//go:build wasip1

package main

// serveLanguage runs the language server: wasm cannot start a child process
// to supervise it.
func serveLanguage() { runServer() }
