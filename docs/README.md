# asciix — documentation

`asciix` converts images and video to ASCII art in the terminal. Besides
printing the result on screen, it can save it as text, HTML or a **PNG/JPG
image** (the ASCII drawn with real characters).

It is a **single Go binary**: no virtualenv, no interpreter, no runtime
dependencies for image conversion.

## Index

| Document | Content |
| --- | --- |
| [installation.md](installation.md) | Install `asciix` and its dependencies |
| [images-to-ascii.md](images-to-ascii.md) | Convert images to ASCII: styles, sizes and presets |
| [image-export.md](image-export.md) | Export the ASCII to PNG or JPG |
| [video.md](video.md) | Play video as ASCII in the terminal |
| [xfetch.md](xfetch.md) | Generate small logos for `xfetch` |
| [reference.md](reference.md) | Full command-line reference |

## Quick start

```bash
# 1) See an image as ASCII in the terminal
asciix photo.jpg

# 2) Braille, in color, for more detail
asciix photo.jpg -m braille --color

# 3) Save a 2560 px wide JPG with the ASCII drawn
asciix photo.jpg -m ascii -w 160 -o portrait.jpg --px 2560

# 4) Interactive UI to pick a group of images
asciix tui ~/Pictures
```
