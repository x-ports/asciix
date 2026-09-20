#!/usr/bin/env bash
#
# asciix — installer
#
# Installs asciix and its dependencies. Works locally (from the repo) or
# remotely:
#
#   curl -fsSL https://raw.githubusercontent.com/x-ports/asciix/main/install.sh | bash
#
# Options (with curl | bash, pass them after `bash -s --`):
#   --prefix DIR     Install directory (default ~/.local/bin)
#   --no-deps        Do not install system dependencies
#   --deps-only      Only install dependencies, not asciix
#   --source         Force building from source
#   --binary         Force downloading the prebuilt binary
#   --version VER    Version to install (default: latest)
#   -h, --help       Show this help
#
# Environment variables:
#   ASCIIX_REPO      repo without scheme (default x-ports/asciix on GitHub)
#   ASCIIX_VERSION   same as --version
#
set -euo pipefail

REPO="${ASCIIX_REPO:-x-ports/asciix}"
VERSION="${ASCIIX_VERSION:-latest}"

PREFIX=""
INSTALL_DEPS=1
DEPS_ONLY=0
METHOD="auto"

usage() {
	cat <<'EOF'
asciix installer

Usage: install.sh [options]

Options:
  --prefix DIR     Install directory (default ~/.local/bin)
  --no-deps        Do not install system dependencies
  --deps-only      Only install dependencies, not asciix
  --source         Force building from source
  --binary         Force downloading the prebuilt binary
  --version VER    Version to install (default: latest)
  -h, --help       Show this help

Environment:
  ASCIIX_REPO      repo without scheme (default x-ports/asciix on GitHub)
  ASCIIX_VERSION   same as --version

Remote usage:
  curl -fsSL https://raw.githubusercontent.com/x-ports/asciix/main/install.sh | bash
  curl -fsSL .../install.sh | bash -s -- --no-deps
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
	--prefix)
		PREFIX="${2:?missing directory}"
		shift 2
		;;
	--no-deps)
		INSTALL_DEPS=0
		shift
		;;
	--deps-only)
		DEPS_ONLY=1
		shift
		;;
	--source)
		METHOD=source
		shift
		;;
	--binary)
		METHOD=binary
		shift
		;;
	--version)
		VERSION="${2:?missing version}"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "asciix: unknown option: $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

log() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m warning:\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31m error:\033[0m %s\n' "$*" >&2; exit 1; }

# ------------------------------------------------------------------ system ----
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$ARCH" in
x86_64 | amd64) ARCH=amd64 ;;
aarch64 | arm64) ARCH=arm64 ;;
esac

SUDO=""
if [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1; then
	SUDO="sudo"
fi

if [ -z "$PREFIX" ]; then
	if [ "$(id -u)" -eq 0 ]; then
		PREFIX="/usr/local/bin"
	else
		PREFIX="$HOME/.local/bin"
	fi
fi
BIN_DIR="$PREFIX"

detect_pm() {
	if command -v pacman >/dev/null 2>&1; then
		echo pacman
	elif command -v apt-get >/dev/null 2>&1; then
		echo apt
	elif command -v dnf >/dev/null 2>&1; then
		echo dnf
	elif command -v zypper >/dev/null 2>&1; then
		echo zypper
	elif command -v apk >/dev/null 2>&1; then
		echo apk
	elif command -v brew >/dev/null 2>&1; then
		echo brew
	fi
}
PM="$(detect_pm || true)"
[ -n "$PM" ] || PM=none

pkg_install() {
	[ "$#" -eq 0 ] && return 0
	case "$PM" in
	pacman) $SUDO pacman -S --needed --noconfirm "$@" ;;
	apt)
		$SUDO apt-get update -qq
		$SUDO apt-get install -y --no-install-recommends "$@"
		;;
	dnf) $SUDO dnf install -y "$@" ;;
	zypper) $SUDO zypper --non-interactive install "$@" ;;
	apk) $SUDO apk add --no-cache "$@" ;;
	brew) brew install "$@" ;;
	*) return 1 ;;
	esac
}

GO_PKGS=()
MEDIA_PKGS=()
FONT_PKGS=()
case "$PM" in
pacman)
	GO_PKGS=(go)
	MEDIA_PKGS=(ffmpeg imagemagick)
	FONT_PKGS=(adwaita-fonts noto-fonts)
	;;
apt)
	GO_PKGS=(golang-go git)
	MEDIA_PKGS=(ffmpeg imagemagick libpango-1.0-0 librsvg2-common)
	FONT_PKGS=(fonts-freefont-ttf fonts-noto-core fonts-dejavu-core)
	;;
dnf)
	GO_PKGS=(golang git)
	MEDIA_PKGS=(ffmpeg-free ImageMagick pango)
	FONT_PKGS=(gnu-free-mono-fonts google-noto-sans-mono-fonts)
	;;
zypper)
	GO_PKGS=(go git)
	MEDIA_PKGS=(ffmpeg ImageMagick pango-tools)
	FONT_PKGS=(gnu-free-fonts google-noto-sans-mono-fonts)
	;;
apk)
	GO_PKGS=(go git)
	MEDIA_PKGS=(ffmpeg imagemagick pango)
	FONT_PKGS=(font-noto font-freefont)
	;;
brew)
	GO_PKGS=(go)
	MEDIA_PKGS=(ffmpeg imagemagick)
	FONT_PKGS=(font-dejavu font-noto-sans-mono)
	;;
esac

install_dependencies() {
	if [ "$PM" = none ]; then
		warn "unknown package manager; install manually: ffmpeg, ImageMagick (with Pango) and a font with braille/block glyphs"
		return 0
	fi
	log "installing dependencies with $PM"
	pkg_install "${MEDIA_PKGS[@]}" || warn "could not install the image/video tools"
	pkg_install "${FONT_PKGS[@]}" || warn "could not install the fonts"
}

need_go() {
	if command -v go >/dev/null 2>&1; then
		return 0
	fi
	[ "$PM" = none ] && return 1
	log "installing Go"
	pkg_install "${GO_PKGS[@]}" || return 1
	command -v go >/dev/null 2>&1
}

# ------------------------------------------------------------------ source ----
# Are we inside the repo (local script)?
find_source() {
	local d
	for d in "$PWD" "$(dirname "${BASH_SOURCE[0]:-$0}")"; do
		[ -n "$d" ] || continue
		if [ -f "$d/go.mod" ] && [ -f "$d/main.go" ] && grep -q '^module ' "$d/go.mod" 2>/dev/null; then
			(cd "$d" && pwd)
			return 0
		fi
	done
	return 1
}

release_url() {
	local osname
	osname="$(echo "$OS" | tr '[:upper:]' '[:lower:]')"
	if [ "$VERSION" = latest ]; then
		echo "https://github.com/$REPO/releases/latest/download/asciix_${osname}_${ARCH}.tar.gz"
	else
		echo "https://github.com/$REPO/releases/download/${VERSION}/asciix_${osname}_${ARCH}.tar.gz"
	fi
}

install_from_source() {
	local src="$1" tmp=""
	if [ -z "$src" ]; then
		command -v git >/dev/null 2>&1 || return 1
		need_go || return 1
		tmp="$(mktemp -d)"
		log "cloning $REPO"
		git clone --depth 1 "https://github.com/$REPO.git" "$tmp/asciix" || {
			rm -rf "$tmp"
			return 1
		}
		src="$tmp/asciix"
	else
		need_go || return 1
	fi
	log "building asciix"
	mkdir -p "$BIN_DIR"
	(cd "$src" && go build -o "$BIN_DIR/asciix" .)
	if [ -n "$tmp" ]; then rm -rf "$tmp"; fi
	return 0
}

install_from_release() {
	local url tmp bin
	url="$(release_url)"
	command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1 || return 1
	tmp="$(mktemp -d)"
	log "downloading binary: $url"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$tmp/asciix.tar.gz" || {
			rm -rf "$tmp"
			return 1
		}
	else
		wget -qO "$tmp/asciix.tar.gz" "$url" || {
			rm -rf "$tmp"
			return 1
		}
	fi
	tar -xzf "$tmp/asciix.tar.gz" -C "$tmp" || {
		rm -rf "$tmp"
		return 1
	}
	bin="$(find "$tmp" -type f -name asciix | head -1)"
	[ -n "$bin" ] || {
		rm -rf "$tmp"
		return 1
	}
	mkdir -p "$BIN_DIR"
	install -m 0755 "$bin" "$BIN_DIR/asciix"
	rm -rf "$tmp"
	return 0
}

# -------------------------------------------------------------------- main ----
SRC="$(find_source || true)"

if [ "$INSTALL_DEPS" -eq 1 ]; then
	install_dependencies
fi
if [ "$DEPS_ONLY" -eq 1 ]; then
	log "dependencies installed. Exiting (--deps-only)."
	exit 0
fi

installed=0
if [ "$METHOD" = "binary" ]; then
	install_from_release || die "could not download the binary (does the release exist?)"
	installed=1
elif [ "$METHOD" = "source" ]; then
	install_from_source "$SRC" || die "could not build from source"
	installed=1
else
	if [ -n "$SRC" ]; then
		install_from_source "$SRC" || die "could not build"
		installed=1
	elif install_from_release; then
		installed=1
	elif install_from_source ""; then
		installed=1
	fi
fi

if [ "$installed" -ne 1 ]; then
	die "could not install asciix (try --source or --binary)"
fi

log "asciix installed at $BIN_DIR/asciix"
"$BIN_DIR/asciix" version 2>/dev/null || true

case ":$PATH:" in
*":$BIN_DIR:"*) ;;
*)
	warn "$BIN_DIR is not in your PATH. Add to ~/.zshrc or ~/.bashrc:"
	printf '    export PATH="%s:$PATH"\n' "$BIN_DIR"
	;;
esac

echo
echo "Examples:"
echo "    asciix photo.jpg -m braille --color"
echo "    asciix tui ~/Pictures"
echo "    asciix video clip.mp4"
echo "    asciix --help"
