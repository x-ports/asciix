# Convert images to ASCII

## The basics

```bash
asciix image.png
```

With no options, `asciix` uses the `ascii` style, fills the current terminal
width and shows the result in color when the output is a terminal.

## Styles (`-m`, `--mode`)

| Mode | Glyphs | Resolution per cell | Notes |
| --- | --- | --- | --- |
| `ascii` | 70-level ASCII ramp (`.` `,` `:` `;` letters, `@`, `#`) | 1×1 | Classic. Widest tonal range |
| `short` | `" .:-=+*#%@"` | 1×1 | 10 levels, cleaner |
| `shade` | `░▒▓█` | 1×1 | Shading blocks |
| `blocks` | `▀▄█` and half blocks | 1×2 | With color it reconstructs the image (great for wallpapers) |
| `quad` | `▖▗▘▝▚▞` | 2×2 | Quadrants, good balance |
| `braille` | `⠿` (Unicode dots) | 2×4 | Highest density, best detail |

```bash
asciix photo.jpg -m braille --color
asciix photo.jpg -m quad
asciix photo.jpg -m blocks --color
```

List them from the program:

```bash
asciix modes
```

## Size

- `-w`, `--width`: width in **characters**. Defaults to the terminal width.
- `--height`: height in lines. Defaults to a value that preserves the image
  aspect ratio (a terminal cell is ~2× taller than wide).

```bash
asciix photo.jpg -w 120          # 120 columns
asciix photo.jpg -w 80 --height 30
```

## Color

- Default: automatic (on for terminal, HTML and PNG/JPG; off for `.txt`).
- `--color`: force 24-bit color (useful when redirecting or saving).
- `--no-color`: disable color.

```bash
asciix photo.jpg --color | less -R
asciix photo.jpg -o photo.txt --no-color
```

## Save the result (text / HTML)

```bash
asciix photo.jpg -o photo.txt                 # plain text
asciix photo.jpg -o photo.html --color        # self-contained colored HTML
asciix *.png -o out/ -f html --color          # one HTML per image
asciix ~/Pictures -r -o out/                  # recursive
```

- `-o` may be a file or a **directory** (when there are several images, or the
  destination is already a directory, one file per image is created).
- `-f`, `--format` accepts `term`, `txt`, `html`, `png`, `jpg`. If omitted, it
  is inferred from the output file extension.

## Presets (`--preset`)

| Preset | Equivalent to | Good for |
| --- | --- | --- |
| `logo` | `-m ascii -w 64 --no-color` | Small logos (xfetch, motd) |
| `wallpaper` | `-m blocks -w <terminal width> --color` | Full-screen background |

```bash
asciix logo.png --preset logo -o logo.txt
asciix landscape.jpg --preset wallpaper -o background.jpg --px 2560
```

Presets only fill values you did **not** set: any explicit flag wins.

## Tone and color tweaks

```bash
asciix photo.jpg --contrast                    # auto-levels (dark images)
asciix photo.jpg --gamma 1.6                    # brighten midtones
asciix photo.jpg --brightness 1.5               # overall brightness
asciix photo.jpg --saturation 1.8               # more color
asciix photo.jpg --dither                       # ordered dithering (more tonal detail)
asciix photo.jpg --invert                       # invert light/dark
asciix photo.jpg --threshold 160                # threshold for blocks/quad/braille
asciix photo.jpg -m ascii --ramp " .:-=+*#%@"   # custom ramp
asciix photo.jpg --list                         # size/mode only, don't draw
```

The levels/gamma/brightness transfer is computed on **luminance** and applied
to RGB as a gain, so hue and saturation are preserved (colors are not washed
out). Use `--clip` to control the auto-levels clipping (default `0.5`%).

## Interactive UI (`tui`)

```bash
asciix                  # no args, on a terminal
asciix tui ~/Pictures   # explicit directory
```

The main menu offers: 1) convert images, 2) play video, 3) create an xfetch
logo, 4) quit.

For images the UI lets you:

1. Pick a **group of images** (`todo`, `1,3,5`, ranges `2-4`).
2. Choose a **preset** (`logo`/`wallpaper`) or style, width and color manually.
3. **Tone/color adjustments**: automatic or custom (contrast, gamma,
   brightness, saturation, invert, dither).
4. Preview in the terminal and/or **save** to `txt`, `html`, `png` or `jpg`
   (with canvas/`--px` and `cover`).
5. Change directory (`d`) or quit (`q`).
