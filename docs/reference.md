# Command-line reference

## Synopsis

```
asciix [options] image1.png [image2.jpg ...]
asciix tui [directory]
asciix video [options] video.mp4
asciix xfetch image.png [name] [options]
asciix modes
asciix help | --help | -h
asciix version | --version | -V
```

Running `asciix` with no arguments opens the [interactive UI](images-to-ascii.md#interactive-ui-tui)
when there is a terminal; otherwise it prints the help.

## Convert images

| Option | Description | Default |
| --- | --- | --- |
| `-m`, `--mode <mode>` | Style: `ascii`, `short`, `shade`, `blocks`, `quad`, `braille` | `ascii` |
| `-w`, `--width <n>` | Width in characters | terminal width |
| `--height <n>` | Height in lines | proportional |
| `-o`, `--out <path>` | Output file or directory | terminal |
| `-f`, `--format <fmt>` | `term`, `txt`, `html`, `png`, `jpg` | from `-o` extension |
| `--color` | Force 24-bit color | auto |
| `--no-color` | Disable color | auto |
| `--invert` | Invert luminance | off |
| `--threshold <n>` | Threshold 0-255 (blocks/quad/braille) | `128` |
| `--ramp <text>` | Character ramp for `ramp` modes | the mode's |
| `--contrast` | Auto-levels (fixes dark images) | off |
| `--gamma <n>` | Gamma correction (>1 brightens) | `1.0` |
| `--brightness <n>` | Brightness (multiplier) | `1.0` |
| `--saturation <n>` | Saturation (multiplier; >1 more color) | `1.0` |
| `--dither` | Ordered dithering (more tonal detail) | off |
| `--clip <n>` | Clipping % for `--contrast` | `0.5` |
| `-r`, `--recursive` | Include subdirectories | off |
| `--preset <p>` | `logo` or `wallpaper` | — |
| `--list` | List size/mode without drawing | off |

### Image export (png/jpg)

| Option | Description | Default |
| --- | --- | --- |
| `--px <n\|WxH>` | Final width (`3840`) or exact canvas (`3840x2160`) | `0` |
| `--cover` | Crop to fill the canvas (no black bars) | off |
| `--font <name>` | Monospaced font | `Adwaita Mono` |
| `--font-size <n>` | Font size in points | `12` |
| `--bg <color>` | Background color | `#000000` |
| `--quality <n>` | JPEG quality 1-100 | `92` |
| `--letter-spacing <pt>` | Horizontal spacing between characters | `0` |
| `--line-spacing <px>` | Vertical spacing between lines | `0` |

## TUI

```
asciix tui [directory]
```

Main menu: 1) convert images, 2) play video, 3) create an xfetch logo, 4) quit.
For images it lets you pick a preset or style/width/color, tone/color
adjustments (automatic or custom), preview, and save to `txt`/`html`/`png`/`jpg`.

It covers almost everything from the CLI. Only in the CLI: `--threshold`,
`--ramp`, `--letter-spacing`, `--line-spacing`, `--bg`, `--quality` and `-r`.

## Video

| Option | Description | Default |
| --- | --- | --- |
| `-m`, `--mode <mode>` | Style | `blocks` |
| `-w`, `--width <n>` | Width in characters | terminal width |
| `--fps <n>` | Frames per second | `15` |
| `--loop` | Repeat in a loop | off |
| `--color` / `--no-color` | 24-bit color | auto |
| `--invert` | Invert luminance | off |
| `--threshold <n>` | Threshold | `128` |

## xfetch

| Option | Description | Default |
| --- | --- | --- |
| `-w`, `--width <n>` | Logo width in characters | `48` |
| `-m`, `--mode <mode>` | Logo style | `ascii` |
| `--force` | Overwrite an existing logo | off |
| `--set` | Update `"ascii"` in `config.jsonc` | off |

## Color behavior

| Output format | Default color |
| --- | --- |
| Terminal (`term`) | yes, if the output is a terminal |
| `txt` | no |
| `html` | yes (colored `<span>` tags) |
| `png` / `jpg` | yes |

`--color` and `--no-color` always override the automatic behavior.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | I/O error or an external tool error (ffmpeg/ImageMagick) |
| `2` | Incorrect argument usage |
