package main

import (
	"fmt"
	"os"
	"strings"
)

// ---------------------------------------------------------------- keyboard ---

type keyKind int

const (
	kNone keyKind = iota
	kRune
	kUp
	kDown
	kLeft
	kRight
	kEnter
	kEsc
	kSpace
	kBackspace
	kTab
	kHome
	kEnd
	kCtrlC
)

type keyEvent struct {
	kind keyKind
	r    rune
}

func readByte(f *os.File) (byte, bool) {
	buf := make([]byte, 1)
	n, err := f.Read(buf)
	if err != nil || n != 1 {
		return 0, false
	}
	return buf[0], true
}

// readKey blocks until a key is pressed, decoding arrows and control keys.
func readKey(f *os.File) keyEvent {
	for {
		b, ok := readByte(f)
		if !ok {
			continue
		}
		switch b {
		case 0x1b: // ESC or escape sequence
			b2, ok := readByte(f)
			if !ok {
				return keyEvent{kind: kEsc}
			}
			if b2 != '[' && b2 != 'O' {
				return keyEvent{kind: kEsc}
			}
			b3, ok := readByte(f)
			if !ok {
				return keyEvent{kind: kEsc}
			}
			switch b3 {
			case 'A':
				return keyEvent{kind: kUp}
			case 'B':
				return keyEvent{kind: kDown}
			case 'C':
				return keyEvent{kind: kRight}
			case 'D':
				return keyEvent{kind: kLeft}
			case 'H':
				return keyEvent{kind: kHome}
			case 'F':
				return keyEvent{kind: kEnd}
			}
			return keyEvent{kind: kEsc}
		case '\r', '\n':
			return keyEvent{kind: kEnter}
		case ' ':
			return keyEvent{kind: kSpace}
		case 127, 8:
			return keyEvent{kind: kBackspace}
		case '\t':
			return keyEvent{kind: kTab}
		case 3:
			return keyEvent{kind: kCtrlC}
		}
		if b >= 32 {
			return keyEvent{kind: kRune, r: rune(b)}
		}
	}
}

// ----------------------------------------------------------------- canvas ----

type style uint8

const (
	stNormal style = iota
	stDim
	stBold
	stTitle
	stCursor
	stSelected
	stAccent
	stGood
)

func styleCode(s style) (start, end string) {
	switch s {
	case stDim:
		return "\x1b[2m", "\x1b[22m"
	case stBold:
		return "\x1b[1m", "\x1b[22m"
	case stTitle:
		return "\x1b[1;36m", "\x1b[0m"
	case stCursor:
		return "\x1b[7m", "\x1b[27m"
	case stSelected:
		return "\x1b[32m", "\x1b[39m"
	case stAccent:
		return "\x1b[33m", "\x1b[39m"
	case stGood:
		return "\x1b[1;32m", "\x1b[0m"
	default:
		return "", ""
	}
}

type cell struct {
	r rune
	s style
}

type canvas struct {
	w, h int
	c    [][]cell
}

func newCanvas(w, h int) *canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	c := &canvas{w: w, h: h, c: make([][]cell, h)}
	for y := range c.c {
		row := make([]cell, w)
		for x := range row {
			row[x] = cell{r: ' '}
		}
		c.c[y] = row
	}
	return c
}

func (c *canvas) put(x, y int, s string, st style) {
	for _, r := range s {
		if r == '\n' {
			continue
		}
		if x >= 0 && x < c.w && y >= 0 && y < c.h {
			c.c[y][x] = cell{r: r, s: st}
		}
		x++
	}
}

func (c *canvas) hline(y int, r rune, st style) {
	for x := 0; x < c.w; x++ {
		c.c[y][x] = cell{r: r, s: st}
	}
}

func (c *canvas) render() string {
	var sb strings.Builder
	sb.WriteString("\x1b[H")
	for y := 0; y < c.h; y++ {
		cur := stNormal
		for x := 0; x < c.w; x++ {
			cl := c.c[y][x]
			if cl.s != cur {
				if cur != stNormal {
					_, end := styleCode(cur)
					if end == "" {
						end = "\x1b[0m"
					}
					sb.WriteString(end)
				}
				start, _ := styleCode(cl.s)
				sb.WriteString(start)
				cur = cl.s
			}
			sb.WriteRune(cl.r)
		}
		if cur != stNormal {
			_, end := styleCode(cur)
			if end == "" {
				end = "\x1b[0m"
			}
			sb.WriteString(end)
		}
		if y < c.h-1 {
			sb.WriteString("\x1b[K\r\n")
		}
	}
	return sb.String()
}

// -------------------------------------------------------------- line input ---

// promptLine reads a line at the bottom of the screen using raw input.
func promptLine(in *os.File, out *os.File, label, current string) (string, bool) {
	w, h := termSize()
	buf := []rune(current)
	draw := func() {
		s := fmt.Sprintf("%s: %s", label, string(buf))
		runes := []rune(s)
		if len(runes) > w-1 {
			runes = runes[len(runes)-(w-1):]
		}
		fmt.Fprintf(out, "\x1b[%d;1H\x1b[K%s", h, string(runes))
	}
	draw()
	for {
		ev := readKey(in)
		switch ev.kind {
		case kEnter:
			fmt.Fprint(out, "\x1b[K")
			return strings.TrimSpace(string(buf)), true
		case kEsc, kCtrlC:
			fmt.Fprint(out, "\x1b[K")
			return "", false
		case kBackspace:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			draw()
		case kRune:
			buf = append(buf, ev.r)
			draw()
		case kSpace:
			buf = append(buf, ' ')
			draw()
		}
	}
}

func altEnter(out *os.File) {
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l\x1b[2J")
}

func altLeave(out *os.File) {
	fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
}
