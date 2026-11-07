#!/usr/bin/env bash
# Membangun binary Community lintas OS/arsitektur + sha256sums.txt di dist/.
# Pakai: scripts/release.sh v0.1.0   (jalankan `make studio` dulu agar Studio tersemat)
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?versi, mis. v0.1.0}"
[ -f pkg/ui/dist/index.html ] || { echo "Studio belum dibangun: jalankan 'make studio'" >&2; exit 1; }
rm -rf dist && mkdir -p dist
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  os="${target%/*}"; arch="${target#*/}"
  ext=""; [ "$os" = windows ] && ext=".exe"
  name="vibeasik-${VERSION}-${os}-${arch}"
  echo "→ $name"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X github.com/vibeasik/vibeasik/pkg/cli.Version=${VERSION}" \
    -o "dist/vibeasik${ext}" ./cmd/vibeasik
  if [ "$os" = windows ]; then (cd dist && zip -q "${name}.zip" vibeasik.exe && rm vibeasik.exe)
  else (cd dist && tar -czf "${name}.tar.gz" vibeasik && rm vibeasik); fi
done
(cd dist && sha256sum * > sha256sums.txt)
echo "Selesai: dist/ (tanda tangani sha256sums.txt sebelum diunggah ke Releases)"
