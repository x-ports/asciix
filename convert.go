package main

import (
	"image"
	"image/color"
	"math"
)

type Options struct {
	Mode       string
	Width      int
	Height     int
	Color      bool
	Invert     bool
	Threshold  int
	Ramp       string
	Contrast   bool
	Gamma      float64
	Clip       float64
	Brightness float64
	Saturation float64
	Dither     bool
	CropAR     float64
}

// bayer4 is a 4x4 ordered-dither matrix (values 0..15).
var bayer4 = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

type Cell struct {
	Ch    rune
	Fg    color.RGBA
	Bg    color.RGBA
	HasBg bool
}

type Grid struct {
	Cells [][]Cell
	W, H  int
}

// Convert turns an image into a Grid of terminal cells.
func Convert(img image.Image, o Options) *Grid {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 || o.Width < 1 {
		return &Grid{}
	}

	// Crop the source to the target aspect ratio so the result fills the
	// canvas without letterbox bars.
	if o.CropAR > 0 {
		img = cropToAR(img, o.CropAR)
		b = img.Bounds()
		iw, ih = b.Dx(), b.Dy()
	}

	m := modeByName(o.Mode)
	ratio := float64(ih) / float64(iw)

	w := o.Width
	h := o.Height
	if h < 1 {
		// A terminal cell is roughly twice as tall as it is wide.
		h = int(math.Round(float64(w) * ratio * 0.5))
		if h < 1 {
			h = 1
		}
	}

	small := resizeArea(img, w*m.Cols, h*m.Rows)
	adj := o.Contrast || (o.Gamma > 0 && o.Gamma != 1) ||
		(o.Brightness > 0 && o.Brightness != 1) || (o.Saturation > 0 && o.Saturation != 1)
	if adj {
		applyLevels(small, o.Contrast, o.Gamma, o.Clip, o.Brightness, o.Saturation)
	}
	ramp := m.Ramp
	if o.Ramp != "" {
		ramp = o.Ramp
	}
	if m.Kind == "ramp" && len([]rune(ramp)) < 2 {
		ramp = asciiRamp
	}
	rampRunes := []rune(ramp)
	thr := float64(o.Threshold)
	if thr <= 0 || thr > 255 {
		thr = 128
	}

	g := &Grid{W: w, H: h, Cells: make([][]Cell, h)}

	levels := len(rampRunes)
	step := 255.0 / float64(levels-1)
	idx := func(v float64, x, y int) int {
		l := v
		if o.Invert {
			l = 255 - l
		}
		if o.Dither {
			d := (float64(bayer4[y&3][x&3]) + 0.5) / 16.0
			l += (d - 0.5) * step
		}
		i := int(l/step + 0.5)
		if i < 0 {
			i = 0
		}
		if i >= levels {
			i = levels - 1
		}
		return i
	}
	lit := func(v float64) bool {
		if o.Invert {
			return v <= thr
		}
		return v > thr
	}
	avg := func(px ...color.RGBA) color.RGBA {
		if len(px) == 0 {
			return color.RGBA{}
		}
		var r, gg, bb, n int
		for _, p := range px {
			r += int(p.R)
			gg += int(p.G)
			bb += int(p.B)
			n++
		}
		return color.RGBA{uint8(r / n), uint8(gg / n), uint8(bb / n), 255}
	}

	for y := 0; y < h; y++ {
		row := make([]Cell, w)
		for x := 0; x < w; x++ {
			switch m.Kind {
			case "half":
				top := small.RGBAAt(x, 2*y)
				bot := small.RGBAAt(x, 2*y+1)
				if o.Color {
					row[x] = Cell{Ch: '▀', Fg: top, Bg: bot, HasBg: true}
				} else {
					t, b := lit(luminance(top)), lit(luminance(bot))
					switch {
					case t && b:
						row[x] = Cell{Ch: '█', Fg: top}
					case t:
						row[x] = Cell{Ch: '▀', Fg: top}
					case b:
						row[x] = Cell{Ch: '▄', Fg: bot}
					default:
						row[x] = Cell{Ch: ' '}
					}
				}
			case "quad":
				var sub [4]color.RGBA
				var mask int
				var litPx []color.RGBA
				sub[0] = small.RGBAAt(2*x, 2*y)
				sub[1] = small.RGBAAt(2*x+1, 2*y)
				sub[2] = small.RGBAAt(2*x, 2*y+1)
				sub[3] = small.RGBAAt(2*x+1, 2*y+1)
				for i, p := range sub {
					if lit(luminance(p)) {
						mask |= 1 << i
						litPx = append(litPx, p)
					}
				}
				ch := quadChars[mask]
				fg := avg(sub[:]...)
				if len(litPx) > 0 {
					fg = avg(litPx...)
				}
				row[x] = Cell{Ch: ch, Fg: fg}
			case "braille":
				var bits uint8
				var litPx []color.RGBA
				for dy := 0; dy < 4; dy++ {
					for dx := 0; dx < 2; dx++ {
						p := small.RGBAAt(2*x+dx, 4*y+dy)
						if lit(luminance(p)) {
							bits |= brailleBits[dy][dx]
							litPx = append(litPx, p)
						}
					}
				}
				fg := avg(litPx...)
				row[x] = Cell{Ch: rune(0x2800) + rune(bits), Fg: fg}
			default: // ramp
				p := small.RGBAAt(x, y)
				row[x] = Cell{Ch: rampRunes[idx(luminance(p), x, y)], Fg: p}
			}
		}
		g.Cells[y] = row
	}
	return g
}

// applyLevels adjusts tone while preserving hue and saturation: the
// auto-levels/gamma/brightness transfer is computed on luminance and then
// applied to RGB as a gain, so colors are not washed out or hue-shifted.
// A final saturation multiplier boosts (or reduces) chroma.
func applyLevels(img *image.RGBA, auto bool, gamma, clip, brightness, saturation float64) {
	if gamma <= 0 {
		gamma = 1
	}
	if brightness <= 0 {
		brightness = 1
	}
	if saturation <= 0 {
		saturation = 1
	}
	if clip < 0 {
		clip = 0
	}
	if clip > 10 {
		clip = 10
	}

	lo, hi := 0, 255
	if auto {
		var hist [256]int
		total := 0
		for i := 0; i < len(img.Pix); i += 4 {
			l := int(0.2126*float64(img.Pix[i]) +
				0.7152*float64(img.Pix[i+1]) +
				0.0722*float64(img.Pix[i+2]))
			if l < 0 {
				l = 0
			}
			if l > 255 {
				l = 255
			}
			hist[l]++
			total++
		}
		if total == 0 {
			return
		}
		cut := int(float64(total) * clip / 100)
		acc := 0
		for i := 0; i < 256; i++ {
			acc += hist[i]
			if acc > cut {
				lo = i
				break
			}
		}
		acc = 0
		for i := 255; i >= 0; i-- {
			acc += hist[i]
			if acc > cut {
				hi = i
				break
			}
		}
		if hi <= lo {
			hi = lo + 1
		}
	}

	// Luminance transfer table.
	var lut [256]float64
	inv := 1.0 / gamma
	span := float64(hi - lo)
	for i := 0; i < 256; i++ {
		v := float64(i-lo) / span * 255
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		if gamma != 1 {
			v = 255 * math.Pow(v/255, inv)
		}
		v *= brightness
		if v > 255 {
			v = 255
		}
		lut[i] = v
	}

	clamp8 := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v + 0.5)
	}
	for i := 0; i < len(img.Pix); i += 4 {
		r := float64(img.Pix[i])
		g := float64(img.Pix[i+1])
		b := float64(img.Pix[i+2])
		l := 0.2126*r + 0.7152*g + 0.0722*b
		var gain float64
		if l > 0.5 {
			li := int(l + 0.5)
			if li > 255 {
				li = 255
			}
			gain = lut[li] / l
		}
		nr, ng, nb := r*gain, g*gain, b*gain
		if saturation != 1 {
			nl := 0.2126*nr + 0.7152*ng + 0.0722*nb
			nr = nl + (nr-nl)*saturation
			ng = nl + (ng-nl)*saturation
			nb = nl + (nb-nl)*saturation
		}
		img.Pix[i] = clamp8(nr)
		img.Pix[i+1] = clamp8(ng)
		img.Pix[i+2] = clamp8(nb)
	}
}
