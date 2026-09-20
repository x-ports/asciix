//go:build linux

package main

import (
	"syscall"
	"unsafe"
)

func ioctlPtr(fd uintptr, req uintptr, arg unsafe.Pointer) syscall.Errno {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	return errno
}

// enableRaw switches the terminal into raw mode with a 100ms read timeout and
// returns a function that restores the previous state.
func enableRaw(fd uintptr) (func(), error) {
	var t syscall.Termios
	if errno := ioctlPtr(fd, syscall.TCGETS, unsafe.Pointer(&t)); errno != 0 {
		return nil, errno
	}
	old := t
	t.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	t.Oflag &^= syscall.OPOST
	t.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	t.Cflag &^= syscall.CSIZE | syscall.PARENB
	t.Cflag |= syscall.CS8
	t.Cc[syscall.VMIN] = 0
	t.Cc[syscall.VTIME] = 1
	if errno := ioctlPtr(fd, syscall.TCSETS, unsafe.Pointer(&t)); errno != 0 {
		return nil, errno
	}
	return func() { ioctlPtr(fd, syscall.TCSETS, unsafe.Pointer(&old)) }, nil
}
