// +build linux

package main

import "syscall"

var ioctlReadTermios = uintptr(syscall.TCGETS)
var ioctlWriteTermios = uintptr(syscall.TCSETS)
