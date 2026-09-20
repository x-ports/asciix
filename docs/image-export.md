# Export the ASCII to PNG or JPG

`asciix` can take an image, convert it to ASCII and then **draw that ASCII back
as a real image** (characters with color on a background). That way you can use
the result as a wallpaper, avatar, cover, etc.

## How it is generated

1. `asciix` converts the image into a grid of characters (same as when showing
   it on screen).
2. It builds **Pango** markup with the foreground color (and background color,
   for `blocks`) of each character.
3. It rasterizes it with **ImageMagick** to the requested format and scales it
   to the requested pixels.

So image export requires ImageMagick with Pango support (see
[installation.md](installation.md)). The other formats (`txt`, `html`,
terminal) need nothing extra.

## Usage

The format is inferred from the `-o` extension:

```bash
asciix photo.jpg -m ascii -w 160 -o portrait.png --px 2560
asciix photo.jpg -m blocks -w 200 -o wallpaper.jpg --px 3840
```

Or explicitly with `-f`:

```bash
asciix photo.jpg -m braille -f png -o out.png --px 1600
asciix photo.jpg -m quad   -f jpg -o out.jpg
```

> `png`/`jpg` **require** `-o`: it makes no sense to “print” an image to the
> terminal.

## Export options

| Option | Default | Description |
| --- | --- | --- |
| `--px <n\|WxH>` | `0` | Final width (`3840`) or exact canvas (`3840x2160`) |
| `--cover` | off | Crop the source to the canvas aspect and fill it (no black bars) |
| `--font <name>` | `Adwaita Mono` | Monospaced font |
| `--font-size <n>` | `12` | Font size in points |
| `--bg <color>` | `#000000` | Background color |
| `--quality <n>` | `92` | JPEG quality (1-100) |
| `--letter-spacing <pt>` | `0` | Extra horizontal spacing between characters |
| `--line-spacing <px>` | `0` | Extra vertical spacing between lines |

Color is enabled for PNG/JPG (unless you pass `--no-color`). With `--no-color`
the result is white text on the background.

## Which mode to choose

| Goal | Recommended mode |
| --- | --- |
| See the ASCII **characters** | `ascii`, `short`, `quad`, `braille` |
| Look almost like the **photo** (pixel-art) | `blocks --color` |
| Maximum detail in black and white | `braille --no-color` |

Examples:

```bash
# Classic color ASCII, easy to read as characters
asciix photo.jpg -m ascii -w 160 -o ascii.png --px 2560

# Braille: looks like a photo made of dots
asciix photo.jpg -m braille -w 200 -o braille.jpg --px 3840

# Color blocks: reconstructs the image, ideal as a wallpaper
asciix photo.jpg -m blocks -w 240 -o wallpaper.jpg --px 3840 --quality 95

# Elegant monochrome
asciix photo.jpg -m quad -w 160 --no-color --bg '#0d1117' -o mono.png --px 2560
```

## 4K and exact canvases

`--px` accepts two forms:

- `--px 3840` → width is 3840, height follows the aspect ratio.
- `--px 3840x2160` → **exact 4K UHD canvas** (16:9). The ASCII is scaled to fit
  and centered on the background color (`--bg`), adding bars if the image is
  not 16:9.

Add **`--cover`** to avoid those bars: it crops the source image to the canvas
aspect ratio and then scales it to fill. Ideal for wallpapers.

```bash
# Exact 4K UHD, no black bars
asciix landscape.jpg -m blocks -w 240 -o ~/wallpapers/ascii-4k.jpg --px 3840x2160 --cover --quality 95

# Exact 4K with visible characters
asciix landscape.jpg -m ascii -w 220 -o ascii-4k.png --px 3840x2160 --cover

# Other canvases: 2K QHD, 1080p, phone portrait
asciix photo.jpg -o 2k.jpg  --px 2560x1440 --cover
asciix photo.jpg -o fhd.jpg --px 1920x1080 --cover
asciix photo.jpg -o phone.png --px 1080x2400 --cover
```

> The higher `-w` (more characters), the finer the drawing when scaled to 4K.
> For 4K, `-w` between 200 and 300 is a good range.

## Size and aspect ratio

- The image height follows the ASCII aspect ratio.
- `-w` controls the **detail** (how many characters); `--px` controls the
  **final size** in pixels.
- If you don't pass `--px`, the natural text size is used (handy for testing).

## Brightness and color

```bash
# Stronger, more colorful result (good for dark wallpapers)
asciix photo.jpg -m ascii -w 300 \
  --contrast --gamma 1.4 --brightness 1.4 --saturation 1.7 \
  --cover -o out.jpg --px 3840x2160
```

## Use it as a wallpaper

The result is a normal PNG/JPG, so it works with any wallpaper manager
(hyprpaper, swww, feh, etc.):

```bash
asciix ~/Pictures/landscape.jpg -m blocks -w 240 -o ~/wallpapers/ascii.jpg --px 3840x2160 --cover
hyprpaper   # or: swww img ~/wallpapers/ascii.jpg
```

## If the grid looks misaligned

This happens when the font lacks the block/braille glyphs and the system falls
back to another font with a different advance width. Fixes:

- Use a font that contains ASCII **and** blocks/braille: `Adwaita Mono`,
  `FreeMono`, `Noto Sans Mono` + `Noto Sans Symbols 2`.
- Pass it explicitly: `--font "Adwaita Mono"`.
- List candidates with `fc-list :charset=2800` (braille) and
  `fc-list :charset=2580` (blocks).
