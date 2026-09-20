# Installation

## Requirements

| Component | Required? | For |
| --- | --- | --- |
| Go 1.24+ | Only to build | Compile the binary (auto-installed by `install.sh`) |
| ffmpeg / ffprobe | Only for `asciix video` | Read video frames |
| ImageMagick (`magick` or `convert`) + Pango | Only for `png`/`jpg` export | Rasterize ASCII to an image |
| A monospaced font with blocks/braille | Recommended | Keep the exported image grid aligned |

`asciix` uses **no external Go modules**: all image conversion (PNG/JPEG/GIF)
is in the standard library. So `go build` works offline and the resulting
binary needs nothing to print or save ASCII.

## Quick install (script)

```bash
curl -fsSL https://raw.githubusercontent.com/x-ports/asciix/main/install.sh | bash
```

The script detects your distro and installs everything needed:
ffmpeg (video), ImageMagick + Pango (PNG/JPG export), fonts — including Noto
Sans CJK for the `chinese`/`japanese` styles — and Go if it is missing or older
than 1.24 (distro package, falling back to the official Go toolchain in
`~/.local/share/asciix/go`). On Arch and Homebrew it also tries to install a
Nerd Font for the `nerd` style. At the end it prints a **feature check** so you
can see what is available.

`asciix` itself goes into `~/.local/bin`.

Useful flags (with `curl | bash`, pass them after `bash -s --`):

```bash
# don't touch system packages
curl -fsSL .../install.sh | bash -s -- --no-deps

# custom directory
curl -fsSL .../install.sh | bash -s -- --prefix ~/bin

# only dependencies
curl -fsSL .../install.sh | bash -s -- --deps-only
```

## Manual install

From the project folder:

```bash
# build
go build -o asciix .

# install into your PATH (recommended)
mkdir -p ~/.local/bin
cp asciix ~/.local/bin/asciix

# check
asciix --version
```

Or with `go install` (leaves the binary in `~/go/bin`):

```bash
go install github.com/x-ports/asciix@latest
```

If `~/.local/bin` is not in your `PATH`, add to `~/.zshrc` or `~/.bashrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Check the optional dependencies

```bash
command -v ffmpeg  magick convert   # video and image export
magick -list format | grep -i pango # should show PANGO
```

On Arch Linux:

```bash
sudo pacman -S go ffmpeg imagemagick adwaita-fonts
```

## Fonts

For PNG/JPG export, `asciix` uses **Adwaita Mono** by default, which includes
the block (`▀▄█░▒▓`) and braille (`⠿`) glyphs. If you don't have it, install a
font that does (for example `adwaita-fonts`, `ttf-freefont` or `noto-fonts`) or
pass another one with `--font`:

```bash
asciix image.png -m braille -o braille.png --font "FreeMono"
```

List the available monospaced fonts:

```bash
fc-list :mono | cut -d: -f2 | sort -u
```

### Chinese / Japanese / Nerd Font styles

- `chinese` and `japanese` need **Noto Sans CJK** (`noto-fonts-cjk` on Arch,
  `fonts-noto-cjk` on Debian/Ubuntu). The install script includes it.
- `nerd` needs a **Nerd Font** such as **Hack Nerd Font**. Install it from
  [nerdfonts.com](https://www.nerdfonts.com/) or your distro/AUR (for example
  `ttf-nerd-fonts-symbols` or `ttf-hack-nerd`). Without it, the icon glyphs may
  show as boxes.

```bash
asciix photo.jpg -m chinese -o cn.jpg --px 2560
asciix photo.jpg -m japanese -o jp.jpg --px 2560
asciix photo.jpg -m nerd -o nerd.jpg --px 2560
```
