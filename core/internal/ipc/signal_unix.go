//go:build unix

package ipc

import "syscall"

var syscallSignalZero = syscall.Signal(0)
