//go:build darwin || linux

package main

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

// termWidth returns the terminal's column count via the TIOCGWINSZ ioctl, or 0
// if it cannot be determined.
func termWidth(f *os.File) int {
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), tiocgwinsz, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0
	}
	return int(ws.Col)
}
