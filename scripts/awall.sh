#!/usr/bin/env bash
#
# Genera, de forma repetible, todas las variantes ASCII de las imágenes de un
# directorio. Ajusta los parámetros de abajo y vuelve a ejecutarlo cuando
# quieras: el resultado es siempre el mismo.
#
# Uso:
#   scripts/awall.sh <directorio_entrada> [directorio_salida]
#
# Ejemplo:
#   scripts/awall.sh ~/Downloads/Awall-2026.../Awall
#
set -euo pipefail

IN="${1:?uso: awall.sh <directorio_entrada> [directorio_salida]}"
OUT="${2:-$IN}"

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ASCIX="${ASCIX:-$(command -v asciix || true)}"
[ -z "$ASCIX" ] && ASCIX="$here/../asciix"

# ---- parámetros (todo el look se controla aquí) -----------------------------
STYLE="${STYLE:-dark}"    # dark (fondo negro) | light (fondo claro, invertido)
JPG_WIDTH=300           # columnas de las imágenes grandes
JPG_PX=3840x2160        # lienzo final (4K UHD). Usa "3840" para solo ancho
JPG_QUALITY=92
TXT_WIDTH=90            # columnas de los txt pequeños
BRIGHT=(--contrast --gamma 1.4 --brightness 1.4 --saturation 1.7)  # luz + color

if [ "$STYLE" = "light" ]; then
	BG='#f4f2ee'
	INV=(--invert)
else
	BG='#000000'
	INV=()
fi
# -----------------------------------------------------------------------------

mkdir -p "$OUT"

big() { # carpeta modo [flags...]
	local name="$1" mode="$2"
	shift 2
	mkdir -p "$OUT/$name"
	"$ASCIX" "$IN" -m "$mode" -w "$JPG_WIDTH" "${BRIGHT[@]}" "${INV[@]}" "$@" \
		--cover --bg "$BG" -f jpg -o "$OUT/$name" --px "$JPG_PX" --quality "$JPG_QUALITY"
}

small() { # carpeta modo [flags...]
	local name="$1" mode="$2"
	shift 2
	mkdir -p "$OUT/$name"
	"$ASCIX" "$IN" -m "$mode" -w "$TXT_WIDTH" "${BRIGHT[@]}" "$@" \
		-f txt -o "$OUT/$name"
}

echo ">> imágenes grandes (jpg, $JPG_PX)"
big  jpg          ascii
big  jpg-braille  braille
big  jpg-quad     quad --letter-spacing 4 --line-spacing 10
big  jpg-shade    shade
big  jpg-blocks   blocks

echo ">> texto pequeño (txt, $TXT_WIDTH columnas)"
small txts         ascii
small txt-braille  braille
small txt-blocks   blocks

echo "Listo. Variantes en: $OUT"
