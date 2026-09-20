# Video as ASCII

Play a video as color ASCII directly in the terminal.

## Usage

```bash
asciix video clip.mp4
asciix video clip.mp4 -m braille --color
asciix video clip.mp4 -w 100 -fps 24
```

`asciix` extracts frames with **ffmpeg** (scaled to the requested grid),
converts them with the same engine as images and draws them with ANSI escape
sequences. Press `Ctrl+C` to quit (the cursor is restored automatically).

## Options

| Option | Default | Description |
| --- | --- | --- |
| `-m`, `--mode <mode>` | `blocks` | Style (same as images) |
| `-w`, `--width <n>` | terminal width | Width in characters |
| `--fps <n>` | `15` | Frames per second |
| `--loop` | off | Repeat in a loop |
| `--color` / `--no-color` | auto | Force/disable 24-bit color |
| `--invert` | off | Invert luminance |
| `--threshold <n>` | `128` | Threshold for `blocks`/`quad`/`braille` |

Requires `ffmpeg` and `ffprobe` in `PATH`. Check with `command -v ffmpeg`.

## Notes

- For smooth 24 fps, use a GPU-accelerated terminal and a moderate `-w` (for
  example `80`–`120`). `braille` has more dots per character and uses more CPU.
- The height is computed to preserve the video aspect ratio and is clamped to
  about twice the terminal height.
- Audio is not played; `asciix` only draws the video.
- You can record a conversion to a file (it contains ANSI escapes):

  ```bash
  asciix video clip.mp4 --color > clip.ansi
  less -R clip.ansi
  ```
