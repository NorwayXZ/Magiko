#!/bin/sh
# Build a standalone macOS ARM64 app without touching installed software.
set -eu
cd "$(dirname "$0")/../.."
case "$(uname -s)-$(uname -m)" in Darwin-arm64) ;; *) echo 'Build on Apple Silicon macOS' >&2; exit 1;; esac
VERSION=${VERSION:-0.1.0-preview.1}
case "$VERSION" in *[!A-Za-z0-9.+-]*|'') echo 'Invalid version' >&2; exit 1;; esac
BUNDLE_VERSION=$(printf '%s' "$VERSION" | sed -E 's/^v//; s/[-+].*$//')
printf '%s\n' "$BUNDLE_VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || { echo 'Version must be semver' >&2; exit 1; }
OUT="$PWD/dist"
APP="$OUT/Magiko.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
export CGO_CFLAGS="-mmacosx-version-min=12.0"
export CGO_LDFLAGS="-mmacosx-version-min=12.0"
go build -tags 'production magiko' -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$APP/Contents/MacOS/magiko" .
cp build/magiko/magiko.icns "$APP/Contents/Resources/"
sed "s/@VERSION@/$BUNDLE_VERSION/g" build/magiko/Info.plist > "$APP/Contents/Info.plist"
plutil -lint "$APP/Contents/Info.plist"
codesign --force --deep --sign - "$APP"
codesign --verify --deep --strict "$APP"
ZIP="$OUT/Magiko-$VERSION-mac-arm64.zip"
ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIP"
(cd "$OUT"; shasum -a 256 "$(basename "$ZIP")" > "$(basename "$ZIP").sha256")
printf '\n%s\n' "$ZIP"
