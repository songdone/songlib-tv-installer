//go:build !linux

package webupdate

import "golang.org/x/sys/unix"

const traverseFlags = unix.O_RDONLY
