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
#   --no-deps        Do not install system dependencies (ffmpeg/ImageMagick/fonts)
#   --deps-only      Only install dependencies, not asciix
#   --source         Force building from source
#   --binary         Force downloading the prebuilt binary
#   --version VER    Version to install (default: latest)
#   -h, --help       Show this help
#
# Environment variables:
#   ASCIIX_REPO      repo without scheme (default x-ports/asciix on GitHub)
#   ASCIIX_VERSION   same as --version
#   ASCIIX_GO_DIR    where to unpack a downloaded Go toolchain
#                    (default ~/.local/share/asciix/go)
#
# Note: building from source needs Go >= 1.24. If your distro's Go is missing
# or too old, the script downloads the official toolchain automatically.
#
set -euo pipefail

REPO="${ASCIIX_REPO:-x-ports/asciix}"
VERSION="${ASCIIX_VERSION:-latest}"

PREFIX=""
INSTALL_DEPS=1
DEPS_ONLY=0
METHOD="auto"
GO_MIN_MAJOR=1
GO_MIN_MINOR=24

usage() {
	cat <<'EOF'
asciix installer

Usage: install.sh [options]

Options:
  --prefix DIR     Install directory (default ~/.local/bin)
  --no-deps        Do not install system dependencies (ffmpeg/ImageMagick/fonts)
  --deps-only      Only install dependencies, not asciix
  --source         Force building from source
  --binary         Force downloading the prebuilt binary
  --version VER    Version to install (default: latest)
  -h, --help       Show this help

Environment:
  ASCIIX_REPO      repo without scheme (default x-ports/asciix on GitHub)
  ASCIIX_VERSION   same as --version
  ASCIIX_GO_DIR    where to unpack a downloaded Go toolchain

If Go is missing or too old, it is installed automatically (distro package,
falling back to the official Go toolchain).

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
OSNAME="$(echo "$OS" | tr '[:upper:]' '[:lower:]')"

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
	GO_PKGS=(go git)
	MEDIA_PKGS=(ffmpeg imagemagick pango)
	FONT_PKGS=(adwaita-fonts noto-fonts noto-fonts-cjk)
	;;
apt)
	GO_PKGS=(golang-go git)
	MEDIA_PKGS=(ffmpeg imagemagick libpango-1.0-0 librsvg2-common)
	FONT_PKGS=(fonts-freefont-ttf fonts-noto-core fonts-dejavu-core fonts-noto-cjk)
	;;
dnf)
	GO_PKGS=(golang git)
	MEDIA_PKGS=(ffmpeg-free ImageMagick pango)
	FONT_PKGS=(gnu-free-mono-fonts google-noto-sans-mono-fonts google-noto-sans-cjk-fonts)
	;;
zypper)
	GO_PKGS=(go git)
	MEDIA_PKGS=(ffmpeg ImageMagick pango-tools)
	FONT_PKGS=(gnu-free-fonts google-noto-sans-mono-fonts noto-sans-cjk-fonts)
	;;
apk)
	GO_PKGS=(go git)
	MEDIA_PKGS=(ffmpeg imagemagick pango)
	FONT_PKGS=(font-noto font-freefont font-noto-cjk)
	;;
brew)
	GO_PKGS=(go git)
	MEDIA_PKGS=(ffmpeg imagemagick)
	FONT_PKGS=(font-dejavu font-noto-sans-mono font-noto-sans-cjk)
	;;
esac

# Optional: a Nerd Font (for the `nerd` style). Not all distros package one.
NERD_PKGS=()
case "$PM" in
pacman) NERD_PKGS=(ttf-nerd-fonts-symbols) ;;
brew) NERD_PKGS=(font-hack-nerd-font) ;;
esac

download() { # url dest
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$2" "$1"
	else
		return 1
	fi
}

install_dependencies() {
	if [ "$PM" = none ]; then
		warn "unknown package manager; install manually: ffmpeg, ImageMagick (with Pango) and a font with braille/block glyphs"
		return 0
	fi
	log "installing dependencies with $PM"
	pkg_install "${MEDIA_PKGS[@]}" || warn "could not install the image/video tools"
	pkg_install "${FONT_PKGS[@]}" || warn "could not install the fonts"
	if [ "${#NERD_PKGS[@]}" -gt 0 ]; then
		pkg_install "${NERD_PKGS[@]}" || warn "could not install a Nerd Font (optional, only for the 'nerd' style)"
	fi
}

# Prints which optional features are available after installation.
verify_features() {
	echo
	echo "Feature check:"
	if command -v ffmpeg >/dev/null 2>&1; then
		printf '  [x] video (ffmpeg)\n'
	else
		printf '  [ ] video (ffmpeg) — install ffmpeg to use "asciix video"\n'
	fi
	if command -v magick >/dev/null 2>&1 || command -v convert >/dev/null 2>&1; then
		if (magick -list format 2>/dev/null || convert -list format 2>/dev/null || true) | grep -i pango >/dev/null 2>&1; then
			printf '  [x] png/jpg export (ImageMagick + Pango)\n'
		else
			printf '  [ ] png/jpg export — ImageMagick installed but without Pango\n'
		fi
	else
		printf '  [ ] png/jpg export — install ImageMagick with Pango\n'
	fi
	local fonts
	fonts="$(fc-list 2>/dev/null || true)"
	if printf '%s' "$fonts" | grep -i "CJK" >/dev/null 2>&1; then
		printf '  [x] Chinese/Japanese fonts (Noto Sans CJK)\n'
	else
		printf '  [ ] Chinese/Japanese fonts — install noto CJK\n'
	fi
	if printf '%s' "$fonts" | grep -i "nerd" >/dev/null 2>&1; then
		printf '  [x] Nerd Font (nerd style)\n'
	else
		printf '  [ ] Nerd Font (nerd style) — install a Nerd Font\n'
	fi
}

# ---------------------------------------------------------------- Go --------
go_version_ok() {
	command -v go >/dev/null 2>&1 || return 1
	local v maj min rest
	v="$(go env GOVERSION 2>/dev/null || go version 2>/dev/null | awk '{print $3}')"
	v="${v#go}"
	maj="${v%%.*}"
	rest="${v#*.}"
	min="${rest%%[!0-9]*}"
	[ -n "$maj" ] && [ -n "$min" ] || return 1
	if [ "$maj" -gt "$GO_MIN_MAJOR" ]; then return 0; fi
	[ "$maj" -eq "$GO_MIN_MAJOR" ] && [ "$min" -ge "$GO_MIN_MINOR" ]
}

install_go_tarball() {
	local ver url dir tmp
	command -v tar >/dev/null 2>&1 || return 1
	log "fetching the latest Go version"
	ver="$(download "https://go.dev/VERSION?m=text" /dev/stdout 2>/dev/null | head -1 || true)"
	case "$ver" in
	go*) ;;
	*) ver="go${GO_MIN_MAJOR}.${GO_MIN_MINOR}.0" ;;
	esac
	url="https://go.dev/dl/${ver}.${OSNAME}-${ARCH}.tar.gz"
	dir="${ASCIIX_GO_DIR:-$HOME/.local/share/asciix/go}"
	tmp="$(mktemp -d)"
	log "downloading $url"
	download "$url" "$tmp/go.tar.gz" || {
		rm -rf "$tmp"
		return 1
	}
	mkdir -p "$dir"
	tar -C "$dir" --strip-components=1 -xzf "$tmp/go.tar.gz" || {
		rm -rf "$tmp"
		return 1
	}
	rm -rf "$tmp"
	export GOROOT="$dir"
	export PATH="$dir/bin:$PATH"
	log "Go installed at $dir ($("$dir/bin/go" env GOVERSION 2>/dev/null))"
}

# Ensures a usable Go toolchain: existing -> distro package -> official tarball.
ensure_go() {
	if go_version_ok; then
		return 0
	fi
	if [ "$PM" != none ]; then
		log "installing Go with $PM"
		pkg_install "${GO_PKGS[@]}" || true
		if go_version_ok; then
			return 0
		fi
		warn "the package manager did not provide Go >= ${GO_MIN_MAJOR}.${GO_MIN_MINOR}; using the official toolchain"
	fi
	install_go_tarball || return 1
	go_version_ok
}

# ------------------------------------------------------------------ source ----
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
	if [ "$VERSION" = latest ]; then
		echo "https://github.com/$REPO/releases/latest/download/asciix_${OSNAME}_${ARCH}.tar.gz"
	else
		echo "https://github.com/$REPO/releases/download/${VERSION}/asciix_${OSNAME}_${ARCH}.tar.gz"
	fi
}

build_here() { # dir
	mkdir -p "$BIN_DIR"
	(cd "$1" && go build -o "$BIN_DIR/asciix" .)
}

install_from_source() {
	local src="$1" tmp=""
	ensure_go || return 1
	if [ -z "$src" ]; then
		command -v git >/dev/null 2>&1 || {
			warn "git is required to fetch the source"
			return 1
		}
		tmp="$(mktemp -d)"
		log "cloning $REPO"
		git clone --depth 1 "https://github.com/$REPO.git" "$tmp/asciix" || {
			rm -rf "$tmp"
			return 1
		}
		src="$tmp/asciix"
	fi
	log "building asciix"
	build_here "$src"
	if [ -n "$tmp" ]; then rm -rf "$tmp"; fi
	return 0
}

install_from_release() {
	local url tmp bin
	url="$(release_url)"
	tmp="$(mktemp -d)"
	log "downloading binary: $url"
	download "$url" "$tmp/asciix.tar.gz" || {
		rm -rf "$tmp"
		return 1
	}
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

verify_features

echo
echo "Examples:"
echo "    asciix photo.jpg -m braille --color"
echo "    asciix tui ~/Pictures"
echo "    asciix video clip.mp4"
echo "    asciix --help"
