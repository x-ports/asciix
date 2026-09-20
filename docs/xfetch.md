# xfetch logos

`xfetch` uses a **plain text** file with ASCII art as its logo. `asciix` can
generate it from any image, at a size that fits next to the system information.

## Usage

```bash
asciix xfetch logo.png              # creates ~/.config/xfetch/logos/logo.txt
asciix xfetch logo.png myLogo       # custom name
asciix xfetch logo.png myLogo --set # also updates config.jsonc
```

## Options

| Option | Default | Description |
| --- | --- | --- |
| `-w`, `--width <n>` | `48` | Logo width in characters |
| `-m`, `--mode <mode>` | `ascii` | Logo style |
| `--force` | off | Overwrite the logo if it already exists |
| `--set` | off | Change the `"ascii"` key in `~/.config/xfetch/config.jsonc` |

The logo is always saved as plain text (no color), which is what `xfetch`
understands.

## Adjusting the size

`xfetch` shows the logo on the left and information on the right. As a guide:

- `-w 40` to `-w 56`: usual sizes so it doesn't overflow.
- A square logo `40` wide is usually ~20 lines tall.

```bash
# A bit bigger
asciix xfetch logo.png myLogo -w 56 --force

# Black-and-white logo from a photo
asciix xfetch portrait.jpg face -w 44 --force
```

## Enable it manually

If you prefer not to use `--set`, edit `~/.config/xfetch/config.jsonc` and point
the `"ascii"` key to your file:

```jsonc
{
    "ascii": "/home/user/.config/xfetch/logos/myLogo.txt",
    // ...
}
```

`--set` makes exactly this change for you (it replaces the path of the
`"ascii"` key).

## Tips

- For logos, `ascii` or `short` give readable characters; `braille` has more
  detail but can look dotted at small sizes.
- If the source is too dark, raise the contrast with `--contrast` or try
  `--invert`.
