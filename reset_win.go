// +build windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
)

// Check if Stdin has been redirected
func checkStdin() {
	_, err := os.Stdin.Stat()
	if err == nil {
		fmt.Println()
		fmt.Println("Stdin redirection is not supported in windows!")
		fmt.Println()

		os.Exit(1)

		// Reset Stdin so we can prompt the user for a password
		//		if fi.Mode()&os.ModeType == 0 {
		//			fd, err := syscall.Open("CONIN$", syscall.GENERIC_READ, 0)
		//			checkErr(err)
		//			fmt.Println("Setting FD to", fd)
		//
		//			err = setStdHandle(syscall.STD_INPUT_HANDLE, fd)
		//			checkErr(err)
		//		}
	}
}

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}
