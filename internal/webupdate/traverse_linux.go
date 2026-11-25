//go:build linux

package webupdate

import "golang.org/x/sys/unix"

// O_PATH needs only search permission on each ancestor directory.
const traverseFlags = unix.O_PATH
