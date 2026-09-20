package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

const tiocgwinsz = 0x5413

// termSize returns the terminal columns/rows, falling back to 80x24.
func termSize() (int, int) {
	f, err := os.Open("/dev/tty")
	if err == nil {
		defer f.Close()
		var ws winsize
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), tiocgwinsz, uintptr(unsafe.Pointer(&ws)))
		if errno == 0 && ws.Col > 0 {
			return int(ws.Col), int(ws.Row)
		}
		// Fallback to stty on unusual platforms.
		out, err := exec.Command("stty", "size").Output()
		if err == nil {
			fields := strings.Fields(strings.TrimSpace(string(out)))
			if len(fields) == 2 {
				r, _ := strconv.Atoi(fields[0])
				c, _ := strconv.Atoi(fields[1])
				if c > 0 {
					return c, r
				}
			}
		}
	}
	return 80, 24
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
