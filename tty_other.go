//go:build !linux

package main

import "errors"

// enableRaw is only implemented on Linux; elsewhere the TUI reports an error
// and the CLI remains available.
func enableRaw(fd uintptr) (func(), error) {
	return nil, errors.New("interactive UI requires Linux (use the CLI flags)")
}
