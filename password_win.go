// +build windows

package main

import (
	"os"
	"syscall"
)

// SetConsoleMode function can be used to change value of ENABLE_ECHO_INPUT:
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686033(v=vs.85).aspx
const ENABLE_ECHO_INPUT = 0x0004

func readPassword() (string, error) {
	hStdin := syscall.Handle(os.Stdin.Fd())
	var oldMode uint32

	err = syscall.GetConsoleMode(hStdin, &oldMode)
	if err != nil {
		return
	}

	var newMode uint32 = (oldMode &^ ENABLE_ECHO_INPUT)

	err = setConsoleMode(hStdin, newMode)
	defer setConsoleMode(hStdin, oldMode)
	if err != nil {
		return
	}

	var buf [16]byte
	var ret []byte
	for {
		n, err := syscall.Read(0, buf[:])
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

func setConsoleMode(console syscall.Handle, mode uint32) (err error) {
	dll := syscall.MustLoadDLL("kernel32")
	proc := dll.MustFindProc("SetConsoleMode")
	r, _, err := proc.Call(uintptr(console), uintptr(mode))

	if r == 0 {
		return err
	}
	return nil
}
