// +build darwin

package main

import "syscall"

var ioctlReadTermios = uintptr(syscall.TIOCGETA)
var ioctlWriteTermios = uintptr(syscall.TIOCSETA)
