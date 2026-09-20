# asciix

Turn images and video into ASCII art — in the terminal, as text, as HTML, or as
a **real PNG/JPG image** (the ASCII drawn with characters). Single Go binary,
no runtime dependencies.

```bash
asciix photo.jpg -m braille --color
```

## Features

- **6 styles**: `ascii`, `short`, `shade` (░▒▓█), `blocks` (▀▄█), `quad`,
  `braille` (highest density).
- **24-bit color** plus tone controls: `--contrast`, `--gamma`, `--brightness`,
  `--saturation`, `--dither`, `--invert`.
- **Exports**: terminal, `.txt`, `.html`, `.png`, `.jpg` (up to 4K UHD, with
  `--cover` to avoid black bars).
- **Batch** whole directories (`-r` for recursive) and **presets**
  (`logo`, `wallpaper`).
- **Video** playback as ASCII (`asciix video`).
- **xfetch** logo generator (`asciix xfetch`).
- **Interactive UI** (`asciix tui`).

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/x-ports/asciix/main/install.sh | bash
```

Or build from source:

```bash
go build -o ~/.local/bin/asciix .
```

Video needs `ffmpeg`; PNG/JPG export needs ImageMagick with Pango. The install
script takes care of them. See [docs/installation.md](docs/installation.md).

## Usage

```bash
asciix image.png                                  # preview in the terminal
asciix image.png -m braille --color               # detail + color
asciix *.png -w 120 -o out/ -f html --color       # batch to HTML
asciix image.png -m ascii -w 160 -o art.png --px 2560          # export to PNG
asciix image.png -m blocks -w 240 -o 4k.jpg --px 3840x2160 --cover
asciix tui ~/Pictures                             # interactive UI
asciix video clip.mp4 -m blocks                   # video
asciix xfetch logo.png mylogo --set               # xfetch logo
asciix modes                                      # list styles
```

Full documentation in [docs/](docs/README.md).

## License

MIT — see [LICENSE](LICENSE).
