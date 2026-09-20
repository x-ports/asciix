package main

import (
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	// Register decoders.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
}

func isImage(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.New("no se pudo decodificar " + filepath.Base(path) + ": " + err.Error())
	}
	return img, nil
}

// listImages returns image files in dir (optionally recursive), sorted.
func listImages(dir string, recursive bool) ([]string, error) {
	var out []string
	if recursive {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && isImage(path) {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() && isImage(e.Name()) {
				out = append(out, filepath.Join(dir, e.Name()))
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// resizeArea scales src to w x h using an area (box) average filter, which
// works well for the strong downscaling ASCII conversion requires.
func resizeArea(src image.Image, w, h int) *image.RGBA {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0 := b.Min.Y + y*sh/h
		y1 := b.Min.Y + (y+1)*sh/h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*sw/w
			x1 := b.Min.X + (x+1)*sw/w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var sr, sg, sb, sa, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					r, g, bl, a := src.At(xx, yy).RGBA()
					sr += uint64(r)
					sg += uint64(g)
					sb += uint64(bl)
					sa += uint64(a)
					n++
				}
			}
			if n == 0 {
				continue
			}
			ar, ag, ab, aa := sr/n, sg/n, sb/n, sa/n
			c := color.RGBA{A: uint8(aa >> 8)}
			if aa > 0 {
				c.R = uint8((ar * 0xffff / aa) >> 8)
				c.G = uint8((ag * 0xffff / aa) >> 8)
				c.B = uint8((ab * 0xffff / aa) >> 8)
			}
			dst.SetRGBA(x, y, c)
		}
	}
	return dst
}

func luminance(c color.RGBA) float64 {
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

// cropped is a lightweight centered sub-view of an image.
type cropped struct {
	img image.Image
	r   image.Rectangle
}

func (c cropped) ColorModel() color.Model { return c.img.ColorModel() }
func (c cropped) Bounds() image.Rectangle { return c.r }
func (c cropped) At(x, y int) color.Color { return c.img.At(x, y) }

// cropToAR returns img cropped in the center to the given aspect ratio
// (width/height).
func cropToAR(img image.Image, ar float64) image.Image {
	if ar <= 0 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	cur := float64(w) / float64(h)
	var cw, ch int
	if cur > ar {
		ch = h
		cw = int(math.Round(float64(h) * ar))
	} else {
		cw = w
		ch = int(math.Round(float64(w) / ar))
	}
	if cw < 1 {
		cw = 1
	}
	if ch < 1 {
		ch = 1
	}
	x0 := b.Min.X + (w-cw)/2
	y0 := b.Min.Y + (h-ch)/2
	return cropped{img: img, r: image.Rect(x0, y0, x0+cw, y0+ch)}
}
