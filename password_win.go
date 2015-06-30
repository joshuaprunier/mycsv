// +build windows

package main

import (
	"io"
	"os"
	"syscall"
)

// SetConsoleMode function can be used to change value of enableEchoInput:
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686033(v=vs.85).aspx
const enableEchoInput = 0x0004

// Windows specific readPassword() function
func readPassword() ([]byte, error) {
	hStdin := syscall.Handle(os.Stdin.Fd())
	var oldMode uint32

	var err error
	err = syscall.GetConsoleMode(hStdin, &oldMode)
	if err != nil {
		checkErr(err)
		//		return nil, err
	}

	var newMode = (oldMode &^ enableEchoInput)

	err = setConsoleMode(hStdin, newMode)
	defer setConsoleMode(hStdin, oldMode)
	if err != nil {
		checkErr(err)
		//		return nil, err
	}

	var buf [16]byte
	var ret []byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			checkErr(err)
			//			return nil, err
		}

		if n == 0 {
			if len(ret) == 0 {
				return nil, io.EOF
			}
			break
		}

		// Remove new line
		if buf[n-1] == '\n' {
			n--
		}

		// Remove carriage return
		if buf[n-1] == '\r' {
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
