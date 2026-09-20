package main

import (
	"fmt"
	"image/color"
	"strings"
)

// Plain renders the grid without colors, trimming trailing spaces.
func (g *Grid) Plain() string {
	var sb strings.Builder
	for _, row := range g.Cells {
		line := make([]rune, len(row))
		for i, c := range row {
			line[i] = c.Ch
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ANSI renders the grid with 24-bit color escape sequences.
func (g *Grid) ANSI() string {
	var sb strings.Builder
	for _, row := range g.Cells {
		var curFg, curBg string
		last := -1
		for i, c := range row {
			if c.Ch != ' ' || c.HasBg {
				last = i
			}
		}
		for i := 0; i <= last; i++ {
			c := row[i]
			if c.Ch == ' ' && !c.HasBg {
				sb.WriteByte(' ')
				continue
			}
			fg := ansiFg(c.Fg)
			if fg != curFg {
				sb.WriteString(fg)
				curFg = fg
			}
			if c.HasBg {
				bg := ansiBg(c.Bg)
				if bg != curBg {
					sb.WriteString(bg)
					curBg = bg
				}
			} else if curBg != "" {
				sb.WriteString("\x1b[49m")
				curBg = ""
			}
			sb.WriteRune(c.Ch)
		}
		if last >= 0 {
			sb.WriteString("\x1b[0m")
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func ansiFg(c color.RGBA) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

func ansiBg(c color.RGBA) string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

// HTML renders the grid as a standalone document.
func (g *Grid) HTML(title string, color bool) string {
	var sb strings.Builder
	sb.WriteString("<!doctype html>\n<meta charset=\"utf-8\">\n")
	sb.WriteString("<title>")
	sb.WriteString(htmlEscape(title))
	sb.WriteString("</title>\n")
	sb.WriteString("<style>body{background:#111;margin:0;display:flex;align-items:center;justify-content:center;min-height:100vh}" +
		"pre{font-family:'DejaVu Sans Mono','Cascadia Mono',monospace;font-size:10px;line-height:1.0;letter-spacing:0}</style>\n")
	sb.WriteString("<pre>\n")
	for _, row := range g.Cells {
		last := -1
		for i, c := range row {
			if c.Ch != ' ' || c.HasBg {
				last = i
			}
		}
		for i := 0; i <= last; i++ {
			c := row[i]
			ch := htmlEscape(string(c.Ch))
			if color && (c.Ch != ' ' || c.HasBg) {
				style := fmt.Sprintf("color:#%02x%02x%02x", c.Fg.R, c.Fg.G, c.Fg.B)
				if c.HasBg {
					style += fmt.Sprintf(";background:#%02x%02x%02x", c.Bg.R, c.Bg.G, c.Bg.B)
				}
				sb.WriteString(`<span style="`)
				sb.WriteString(style)
				sb.WriteString(`">`)
				sb.WriteString(ch)
				sb.WriteString("</span>")
			} else {
				sb.WriteString(ch)
			}
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("</pre>\n")
	return sb.String()
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}
