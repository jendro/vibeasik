#!/usr/bin/env sh
# Pasang vibeasik (Linux/macOS):  curl -fsSL https://vibeasik.com/install.sh | sh
# Variabel: VIBEASIK_VERSION (bawaan: terbaru), VIBEASIK_BASE_URL (bawaan: https://vibeasik.com/dl), INSTALL_DIR (bawaan: ~/.local/bin)
set -eu
SITE="${VIBEASIK_SITE:-https://vibeasik.com/dl}"
DIR="${INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in linux|darwin) ;; *) echo "OS $os belum didukung installer ini (Windows: unduh .zip dari halaman Releases)"; exit 1;; esac
arch="$(uname -m)"
case "$arch" in x86_64|amd64) arch=amd64;; aarch64|arm64) arch=arm64;; *) echo "Arsitektur $arch belum didukung"; exit 1;; esac

version="${VIBEASIK_VERSION:-}"
if [ -z "$version" ]; then
  version="$(curl -fsSL "$SITE/latest" | tr -d '[:space:]')"
  case "$version" in v[0-9]*) ;; *) echo "Tidak bisa menentukan versi terbaru"; exit 1;; esac
fi
name="vibeasik-${version}-${os}-${arch}.tar.gz"
base="${VIBEASIK_BASE_URL:-$SITE/$version}"

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
echo "→ Mengunduh $name"
curl -fsSL "$base/$name" -o "$tmp/$name"
curl -fsSL "$base/sha256sums.txt" -o "$tmp/sha256sums.txt"
want="$(grep " $name\$" "$tmp/sha256sums.txt" | cut -d' ' -f1)"
[ -n "$want" ] || { echo "Checksum $name tidak ada di sha256sums.txt"; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then got="$(sha256sum "$tmp/$name" | cut -d' ' -f1)"; else got="$(shasum -a 256 "$tmp/$name" | cut -d' ' -f1)"; fi
[ "$got" = "$want" ] || { echo "Checksum tidak cocok — pemasangan dibatalkan"; exit 1; }

mkdir -p "$DIR"
tar -xzf "$tmp/$name" -C "$tmp"
install -m 0755 "$tmp/vibeasik" "$DIR/vibeasik"
echo "✅ vibeasik $version terpasang di $DIR/vibeasik"
case ":$PATH:" in *":$DIR:"*) ;; *) echo "⚠ Tambahkan ke PATH:  export PATH=\"$DIR:\$PATH\"";; esac

echo
echo "Langkah berikut:"
echo "  vibeasik serve        # jalankan engine (Studio: vibeasik open)"
echo "  vibeasik login        # opsional: masuk akun (gratis) untuk Pusat Pengetahuan"
echo "  vibeasik mcp --help   # hubungkan agent coding (Claude Code, Cursor, …)"
