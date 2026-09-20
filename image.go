package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// rasterFormats are the image formats we can export by rasterizing the text
// with Pango + ImageMagick.
var rasterFormats = map[string]bool{"png": true, "jpg": true, "jpeg": true}

func isRasterFormat(f string) bool {
	return rasterFormats[strings.ToLower(f)]
}

// pangoMarkup converts the grid into Pango markup so ImageMagick can render
// the characters with real colors and backgrounds.
func (g *Grid) pangoMarkup(color bool, font string, sizePt int, letterSpacing float64) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<span font_family="%s" size="%d"`, xmlAttr(font), sizePt*1024)
	if letterSpacing > 0 {
		fmt.Fprintf(&sb, ` letter_spacing="%d"`, int(letterSpacing*1024))
	}
	sb.WriteByte('>')
	for _, row := range g.Cells {
		last := -1
		for i, c := range row {
			if c.Ch != ' ' || c.HasBg {
				last = i
			}
		}
		if !color {
			for i := 0; i <= last; i++ {
				sb.WriteString(xmlText(string(row[i].Ch)))
			}
			if last >= 0 {
				sb.WriteByte('\n')
			}
			continue
		}
		for i := 0; i <= last; {
			c := row[i]
			j := i
			for j <= last && row[j].Fg == c.Fg && row[j].Bg == c.Bg && row[j].HasBg == c.HasBg {
				j++
			}
			fmt.Fprintf(&sb, `<span foreground="#%02x%02x%02x"`, c.Fg.R, c.Fg.G, c.Fg.B)
			if c.HasBg {
				fmt.Fprintf(&sb, ` background="#%02x%02x%02x"`, c.Bg.R, c.Bg.G, c.Bg.B)
			}
			sb.WriteByte('>')
			for k := i; k < j; k++ {
				sb.WriteString(xmlText(string(row[k].Ch)))
			}
			sb.WriteString("</span>")
			i = j
		}
		if last >= 0 {
			sb.WriteByte('\n')
		}
	}
	sb.WriteString("</span>")
	return sb.String()
}

// WriteImage rasterizes the grid to a PNG/JPEG file using ImageMagick's Pango
// renderer. pxSpec is either a width ("3840") or an exact canvas ("3840x2160").
func (g *Grid) WriteImage(path string, color bool, font string, sizePt int, pxSpec string, quality int, bg string, letterSpacing float64, lineSpacing int, cover bool) error {
	if g.W == 0 || g.H == 0 {
		return fmt.Errorf("nada que exportar")
	}
	bin, err := findMagick()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "asciix-*.markup")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	markup := g.pangoMarkup(color, font, sizePt, letterSpacing)
	if dbg := os.Getenv("ASCIIX_DEBUG"); dbg != "" {
		os.WriteFile(dbg, []byte(markup), 0o644)
	}
	if _, err := tmp.WriteString(markup); err != nil {
		return err
	}
	tmp.Close()

	targetW, targetH, err := parseSize(pxSpec)
	if err != nil {
		return err
	}

	args := []string{"-background", bg}
	if lineSpacing != 0 {
		args = append(args, "-interline-spacing", strconv.Itoa(lineSpacing))
	}
	if !color {
		args = append(args, "-fill", "#ffffff")
	}
	args = append(args, "pango:@"+tmp.Name())
	switch {
	case targetW > 0 && targetH > 0:
		if cover {
			// Scale to cover the canvas and crop the overflow (no bars).
			args = append(args,
				"-resize", fmt.Sprintf("%dx%d^", targetW, targetH),
				"-gravity", "center",
				"-extent", fmt.Sprintf("%dx%d", targetW, targetH),
			)
		} else {
			// Scale to fit, then pad to the exact canvas (centered).
			args = append(args,
				"-resize", fmt.Sprintf("%dx%d", targetW, targetH),
				"-gravity", "center",
				"-extent", fmt.Sprintf("%dx%d", targetW, targetH),
			)
		}
	case targetW > 0:
		args = append(args, "-resize", strconv.Itoa(targetW)+"x")
	}
	if quality > 0 && (strings.HasSuffix(strings.ToLower(path), ".jpg") || strings.HasSuffix(strings.ToLower(path), ".jpeg")) {
		args = append(args, "-quality", strconv.Itoa(quality))
	}
	args = append(args, path)

	cmd := exec.Command(bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("imagen no generada: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// parseSize understands "", "W" and "WxH" (also "WXH" and "WxH!").
func parseSize(spec string) (int, int, error) {
	spec = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(spec), "!"))
	if spec == "" {
		return 0, 0, nil
	}
	parts := strings.FieldsFunc(spec, func(r rune) bool { return r == 'x' })
	switch len(parts) {
	case 1:
		w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || w <= 0 {
			return 0, 0, fmt.Errorf("ancho inválido en --px: %q", spec)
		}
		return w, 0, nil
	case 2:
		w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
			return 0, 0, fmt.Errorf("tamaño inválido en --px: %q (usa ANCHOxALTO)", spec)
		}
		return w, h, nil
	default:
		return 0, 0, fmt.Errorf("tamaño inválido en --px: %q", spec)
	}
}

func findMagick() (string, error) {
	for _, name := range []string{"magick", "convert"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("se requiere ImageMagick (magick o convert) para exportar imágenes")
}

func xmlText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func xmlAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
