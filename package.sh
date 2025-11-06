#!/usr/bin/env bash

set -euo pipefail
PATH="/opt/homebrew/bin:$PATH"

# fpm-based packager for Tailer
# - macOS: Packages bin/tailer-macos-{arch} as .pkg installer to /Applications
# - Linux: Packages bin/tailer-linux as .deb and .rpm installers
# - macOS: Installs as KrankyBearTailer.app bundle structure
# - Linux: Includes bin/Images -> /opt/local/bin/Resources/Images
# - Linux: Includes bin/Sounds -> /opt/local/bin/Resources/Sounds
# - Outputs to ./installers (configurable)

# Detect OS if not specified
if [[ -z "${TYPE:-}" ]]
then
  case "$(uname -s)" in
    Darwin)
      TYPE=macos
      ;;
    Linux)
      TYPE=linux
      ;;
    *)
      echo "Error: Cannot auto-detect OS type. Please set TYPE=macos or TYPE=linux" >&2
      exit 1
      ;;
  esac
fi

# Configurable via env vars (or override on the command line: VAR=value ./package.sh)
NAME=${NAME:-KrankyBearTailer}
VERSION=${VERSION:-0.1.0}
ITERATION=${ITERATION:-1}
OUTDIR=${OUTDIR:-./installers}
MAINTAINER=${MAINTAINER:-"amarillier@gmail.com"}
VENDOR=${VENDOR:-"KrankyBear"}
URL=${URL:-"https://github.com/amarillier/KrankyBearTailer"}
LICENSE=${LICENSE:-"MIT"}

# Architecture handling
# Provide ARCH=amd64 or ARCH=arm64 (default based on OS)
if [[ -z "${ARCH:-}" ]]
then
  if [[ "$TYPE" == "macos" ]]
  then
    # Default to native macOS arch
    ARCH=$(uname -m)
    [[ "$ARCH" == "x86_64" ]] && ARCH=amd64
  else
    ARCH=amd64
  fi
fi

case "$ARCH" in
  amd64|x86_64)
    DEB_ARCH=amd64
    RPM_ARCH=x86_64
    PKG_ARCH=amd64
    ;;
  arm64|aarch64)
    DEB_ARCH=arm64
    RPM_ARCH=aarch64
    PKG_ARCH=arm64
    ;;
  *)
    # Fall back to using the same string
    DEB_ARCH="$ARCH"
    RPM_ARCH="$ARCH"
    PKG_ARCH="$ARCH"
    ;;
esac

# Source assets - depends on TYPE
if [[ "$TYPE" == "macos" ]]
then
  SRC_BIN="bin/tailer-macos-${PKG_ARCH}"
else
  SRC_BIN="bin/tailer-linux"
fi
SRC_IMAGES="bin/Images"
SRC_SOUNDS="bin/Sounds"

usage() {
  cat <<EOF
Usage: [ENV_VARS] ./package.sh

Environment variables:
  TYPE        Package type: macos|linux (default: auto-detect from OS)
  NAME        Package name (default: $NAME)
  VERSION     Package version (default: $VERSION)
  ITERATION   Package iteration/release (default: $ITERATION)
  ARCH        Target arch: amd64|arm64 (default: native for macos, amd64 for linux)
  OUTDIR      Output directory (default: $OUTDIR)
  MAINTAINER  Maintainer (default: $MAINTAINER)
  VENDOR      Vendor (default: $VENDOR)
  URL         Project URL (default: $URL)
  LICENSE     License (default: $LICENSE)

Examples:
  # macOS .pkg
  TYPE=macos VERSION=1.2.3 ARCH=amd64 ./package.sh
  TYPE=macos VERSION=1.2.3 ARCH=arm64 ./package.sh
  
  # Linux .deb and .rpm
  TYPE=linux VERSION=1.2.3 ARCH=amd64 ./package.sh
  TYPE=linux VERSION=1.2.3 ARCH=arm64 OUTDIR=release ./package.sh
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || "${1:-}" == "-?" ]]
then
  usage
  exit 0
fi

if ! command -v fpm >/dev/null 2>&1
then
  echo "Error: fpm not found. Install with: gem install fpm" >&2
  exit 1
fi

# Validate sources
if [[ ! -f "$SRC_BIN" ]]
then
  if [[ "$TYPE" == "macos" ]]
  then
    echo "Error: Missing $SRC_BIN (build your macOS binary first with ./compile-mac.sh)." >&2
  else
    echo "Error: Missing $SRC_BIN (build your Linux binary first)." >&2
  fi
  exit 1
fi
if [[ ! -d "$SRC_IMAGES" ]]
then
  echo "Error: Missing directory $SRC_IMAGES" >&2
  exit 1
fi
if [[ ! -d "$SRC_SOUNDS" ]]
then
  echo "Error: Missing directory $SRC_SOUNDS" >&2
  exit 1
fi

mkdir -p "$OUTDIR"

# For Linux packages built on macOS, create a staging directory with files
# stripped of extended attributes to avoid tar header compatibility issues
STAGING_DIR=""
if [[ "$TYPE" == "linux" && "$(uname -s)" == "Darwin" ]]
then
  echo "Creating staging directory without macOS extended attributes..."
  STAGING_DIR=$(mktemp -d -t fpm-staging.XXXXXX)
  cleanup_staging() { rm -rf "$STAGING_DIR"; }
  trap cleanup_staging EXIT INT TERM
  
  # Copy files to staging directory using cp -X which explicitly excludes
  # extended attributes (xattr) that cause issues with Ubuntu's dpkg
  cp -X "$SRC_BIN" "$STAGING_DIR/tailer-linux"
  cp -XR "$SRC_IMAGES" "$STAGING_DIR/Images"
  cp -XR "$SRC_SOUNDS" "$STAGING_DIR/Sounds"
  
  # Aggressively strip any remaining extended attributes from staging directory
  # This is critical for Ubuntu dpkg compatibility
  if command -v xattr >/dev/null 2>&1
  then
    xattr -cr "$STAGING_DIR" 2>/dev/null || true
    # Also remove any AppleDouble files (._*)
    find "$STAGING_DIR" -name '._*' -delete 2>/dev/null || true
  fi
  
  # Update source paths to point to staging directory
  SRC_BIN="$STAGING_DIR/tailer-linux"
  SRC_IMAGES="$STAGING_DIR/Images"
  SRC_SOUNDS="$STAGING_DIR/Sounds"
  
  # Also set environment variable as additional safeguard
  export COPYFILE_DISABLE=1
fi

COMMON_ARGS=(
  -s dir
  -n "$NAME"
  -v "$VERSION"
  --iteration "$ITERATION"
  --maintainer "$MAINTAINER"
  --vendor "$VENDOR"
  --url "$URL"
  --license "$LICENSE"
  --description "Kranky Bear Tailer - A cross-platform GUI log tail application"
  -f
)

if [[ "$TYPE" == "macos" ]]
then
  # macOS .pkg installer - installs to /Applications as .app bundle
  PKG_OUTFILE="$OUTDIR/krankybeartailer_${VERSION}-${ITERATION}_${PKG_ARCH}.pkg"
  echo "Building macOS .pkg ($PKG_ARCH) -> $PKG_OUTFILE..."
  
  # macOS .app bundle structure:
  # /Applications/KrankyBearTailer.app/Contents/MacOS/KrankyBearTailer (executable)
  # /Applications/KrankyBearTailer.app/Contents/MacOS/Resources/Images (resources)
  # /Applications/KrankyBearTailer.app/Contents/MacOS/Resources/Sounds (sounds)
  # Note: App looks for Resources/Images and Resources/Sounds relative to executable
  APP_NAME="KrankyBearTailer.app"
  APP_DIR="/Applications/$APP_NAME"
  MACOS_DIR="$APP_DIR/Contents/MacOS"
  RESOURCES_DIR="$MACOS_DIR/Resources"
  
  # Determine binary name - use app name without .app
  BIN_NAME="KrankyBearTailer"
  
  fpm \
    "${COMMON_ARGS[@]}" \
    -t osxpkg \
    -a "$PKG_ARCH" \
    --directories "$APP_DIR" \
    --directories "$APP_DIR/Contents" \
    --directories "$MACOS_DIR" \
    --directories "$RESOURCES_DIR" \
    --package "$PKG_OUTFILE" \
    "$SRC_BIN=$MACOS_DIR/$BIN_NAME" \
    "$SRC_IMAGES=$RESOURCES_DIR/Images" \
    "$SRC_SOUNDS=$RESOURCES_DIR/Sounds"
  
  ./setIcon.sh Resources/Images/KrankyBearHogwartsSorting.png "$PKG_OUTFILE"
  echo ""
  echo "Done. Package created:"
  echo "  $PKG_OUTFILE"
  
else
  # Linux .deb and .rpm installers
  DEB_OUTFILE="$OUTDIR/krankybeartailer_${VERSION}-${ITERATION}_${DEB_ARCH}.deb"
  RPM_OUTFILE="$OUTDIR/krankybeartailer_${VERSION}-${ITERATION}_${RPM_ARCH}.rpm"
  
  echo "Building .deb ($DEB_ARCH) -> $DEB_OUTFILE..."
  fpm \
    "${COMMON_ARGS[@]}" \
    -t deb \
    -a "$DEB_ARCH" \
    --deb-no-default-config-files \
    --directories /opt/local/bin \
    --directories /opt/local/bin/Resources \
    --package "$DEB_OUTFILE" \
    "$SRC_BIN=/opt/local/bin/krankybeartailer" \
    "$SRC_IMAGES=/opt/local/bin/Resources/Images" \
    "$SRC_SOUNDS=/opt/local/bin/Resources/Sounds"
  
  echo ""
  echo "Building .rpm ($RPM_ARCH) -> $RPM_OUTFILE..."
  fpm \
    "${COMMON_ARGS[@]}" \
    -t rpm \
    -a "$RPM_ARCH" \
    --rpm-os linux \
    --directories /opt/local/bin \
    --directories /opt/local/bin/Resources \
    --package "$RPM_OUTFILE" \
    "$SRC_BIN=/opt/local/bin/krankybeartailer" \
    "$SRC_IMAGES=/opt/local/bin/Resources/Images" \
    "$SRC_SOUNDS=/opt/local/bin/Resources/Sounds"
  
  echo ""
  echo "Done. Packages created:"
  echo "  $DEB_OUTFILE"
  echo "  $RPM_OUTFILE"
fi

# "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
