// +build darwin linux
package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

func init() {
	// Trap for SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Syscall in case signal is sent during terminal echo off
	var oldState syscall.Termios
	syscall.Syscall6(syscall.SYS_IOCTL, uintptr(0), ioctlReadTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)

	var timer time.Time
	go func() {
		for sig := range sigChan {
			if time.Now().Sub(timer) < time.Second*5 {
				syscall.Syscall6(syscall.SYS_IOCTL, uintptr(0), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)
				os.Exit(0)
			}

			fmt.Println()
			fmt.Println(sig, "signal caught!")
			fmt.Println("Send signal again within 3 seconds to exit")

			timer = time.Now()
		}
	}()
}

// readPassword is borrowed from the crypto/ssh/terminal sub repo to accept a password from stdin without local echo.
// http://godoc.org/golang.org/x/crypto/ssh/terminal#Terminal.ReadPassword
func readPassword() ([]byte, error) {
	// Stdin
	fd := 0

	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 {
		return nil, err
	}

	newState := oldState
	newState.Lflag &^= syscall.ECHO
	newState.Lflag |= syscall.ICANON | syscall.ISIG
	newState.Iflag |= syscall.ICRNL
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return nil, err
	}

	defer func() {
		syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)
	}()

	var buf [16]byte
	var ret []byte
	for {
		n, err := syscall.Read(fd, buf[:])
		if err != nil {
			return nil, err
		}

		if n == 0 {
			if len(ret) == 0 {
				return nil, io.EOF
			}
			break
		}

		if buf[n-1] == '\n' {
			n--
		}

		ret = append(ret, buf[:n]...)
		if n < len(buf) {
			break
		}
	}

	return ret, nil
}
