// +build linux darwin

package main

import (
	"os"
	"syscall"
)

// Check if Stdin has been redirected
func checkStdin() {
	fi, err := os.Stdin.Stat()
	checkErr(err)

	// Reset Stdin so we can prompt the user for a password
	if fi.Mode()&os.ModeDevice != os.ModeDevice {
		f, err := os.Open("/dev/tty")
		checkErr(err)
		fd := f.Fd()

		err = syscall.Dup2(int(fd), int(os.Stdin.Fd()))
		checkErr(err)
	}
}
