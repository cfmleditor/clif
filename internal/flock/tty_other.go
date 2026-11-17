//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd)

package flock

// InForeground: not asked here (Windows, where a console's Ctrl-C reaches
// every attached process; AIX, Solaris, wasm), so it reports true, and a
// caller passes no Ctrl-C on a second time.
func InForeground() bool { return true }
